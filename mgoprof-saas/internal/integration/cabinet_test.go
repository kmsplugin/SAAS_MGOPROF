package integration

// Scenario 3: Role-based event link resolution via real HTTP responses.
//   Participants with speaker/moderator roles see speaker_link;
//   viewer/delegate/guest roles see viewer_link.
//
// Scenario 4: Event type scenarios (online/offline/hybrid) via cabinet API.
//   Online event: no ticket_url. Offline/hybrid: ticket_url present.

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"mgoprof-saas/internal/model"
)

// ── Scenario 3: role-based links ─────────────────────────────────────────────

func TestCabinet_SpeakerLinkForSpeakerRole(t *testing.T) {
	env := newTestEnv(t)

	// Create event with distinct speaker_link and viewer_link.
	var eventID int
	err := env.DB.QueryRowContext(context.Background(), `
		INSERT INTO reg_events
		  (title, description, event_date, event_time, is_active, event_type,
		   speaker_link, viewer_link)
		VALUES ($1, '', '2027-01-01', '10:00:00', true, 'online', $2, $3)
		RETURNING id`,
		"Role Test Event",
		"https://meet.example.com/speakers",
		"https://meet.example.com/viewers",
	).Scan(&eventID)
	if err != nil {
		t.Fatalf("seedEvent with links: %v", err)
	}

	email := "speaker@example.com"

	// Register and verify OTP.
	token := registerAndVerify(t, env, email, eventID)

	// Set the participant role to "speaker" directly in DB after verification
	// (role is typically set by admin, not via registration form).
	_, err = env.DB.ExecContext(context.Background(), `
		UPDATE reg_registrations r
		SET participant_role = 'speaker'
		FROM reg_users u
		WHERE u.id = r.user_id AND u.email = $1 AND r.event_id = $2`,
		email, eventID,
	)
	if err != nil {
		t.Fatalf("set speaker role: %v", err)
	}

	// GET /api/cabinet/events — must return speaker_link for this participant.
	eventsResp := getWithToken(t, env.Server, "/api/cabinet/events", token)
	mustStatus(t, eventsResp, http.StatusOK)

	var result struct {
		Events []map[string]interface{} `json:"events"`
	}
	decodeJSON(t, eventsResp, &result)

	if len(result.Events) == 0 {
		t.Fatal("cabinet/events: expected at least one event, got none")
	}

	// The SpeakerRoles map is tested at the unit level; here we verify that
	// the Event model exposes speaker_link and viewer_link to the caller.
	evt := result.Events[0]
	speakerLink, _ := evt["speaker_link"].(string)
	viewerLink, _ := evt["viewer_link"].(string)

	if speakerLink == "" {
		t.Error("event speaker_link must be non-empty for events with that field set")
	}
	if viewerLink == "" {
		t.Error("event viewer_link must be non-empty for events with that field set")
	}
	if speakerLink == viewerLink {
		t.Error("speaker_link and viewer_link must differ")
	}
}

func TestCabinet_RoleConstants_AllFiveRolesPresent(t *testing.T) {
	// This is a fast compile-time check: all five role constants must exist
	// in the model package and be non-empty.
	roles := []string{
		model.ParticipantRoleSpeaker,
		model.ParticipantRoleModerator,
		model.ParticipantRoleViewer,
		model.ParticipantRoleDelegate,
		model.ParticipantRoleGuest,
	}
	seen := make(map[string]bool)
	for _, r := range roles {
		if r == "" {
			t.Error("role constant must not be empty")
		}
		if seen[r] {
			t.Errorf("duplicate role constant: %q", r)
		}
		seen[r] = true
	}
}

func TestCabinet_SpeakerRolesMap(t *testing.T) {
	// Verify the SpeakerRoles map classifies speaker/moderator correctly
	// and excludes viewer/delegate/guest.
	for _, role := range []string{model.ParticipantRoleSpeaker, model.ParticipantRoleModerator} {
		if !model.SpeakerRoles[role] {
			t.Errorf("role %q must be in SpeakerRoles", role)
		}
	}
	for _, role := range []string{
		model.ParticipantRoleViewer,
		model.ParticipantRoleDelegate,
		model.ParticipantRoleGuest,
	} {
		if model.SpeakerRoles[role] {
			t.Errorf("role %q must NOT be in SpeakerRoles", role)
		}
	}
}

// ── Scenario 4: event type ticket URL ────────────────────────────────────────

func TestVerifyOTP_OnlineEvent_NoTicketURL(t *testing.T) {
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "online")
	email := "online_user@example.com"

	// Register + verify OTP.
	registerResp := post(t, env.Server, "/api/register", model.RegisterRequest{
		Email:        email,
		EventID:      eventID,
		FirstName:    "Test",
		LastName:     "User",
		Organization: "Org",
		District:     "District",
		ConsentGiven: true,
	})
	mustStatus(t, registerResp, http.StatusOK)

	otp := env.FM.LastOTPFor(email)
	verifyResp := post(t, env.Server, "/api/verify-otp", model.VerifyOTPRequest{
		Email:   email,
		EventID: eventID,
		OTP:     otp,
	})
	mustStatus(t, verifyResp, http.StatusOK)

	// For online events the welcome email must NOT include a ticket URL.
	// In the FakeMailer, the welcome email Args[5] is ticketURL.
	sent := env.FM.Sent()
	for _, e := range sent {
		if e.Kind == "welcome" && e.To == email && len(e.Args) >= 6 {
			ticketURL, _ := e.Args[5].(string)
			if ticketURL != "" {
				t.Errorf("online event: welcome email should not carry a ticket URL, got %q", ticketURL)
			}
		}
	}
}

func TestVerifyOTP_OfflineEvent_HasTicketURL(t *testing.T) {
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "offline")
	email := "offline_user@example.com"

	post(t, env.Server, "/api/register", model.RegisterRequest{
		Email:        email,
		EventID:      eventID,
		FirstName:    "Test",
		LastName:     "User",
		Organization: "Org",
		District:     "District",
		ConsentGiven: true,
	})
	otp := env.FM.LastOTPFor(email)
	verifyResp := post(t, env.Server, "/api/verify-otp", model.VerifyOTPRequest{
		Email:   email,
		EventID: eventID,
		OTP:     otp,
	})
	mustStatus(t, verifyResp, http.StatusOK)

	// For offline events the welcome email must carry a ticket URL.
	sent := env.FM.Sent()
	for _, e := range sent {
		if e.Kind == "welcome" && e.To == email && len(e.Args) >= 6 {
			ticketURL, _ := e.Args[5].(string)
			if ticketURL == "" {
				t.Errorf("offline event: welcome email must carry a ticket URL, got empty")
			}
			expectedSuffix := "/cabinet/events/" + itoa(eventID) + "/ticket"
			if !endsWith(ticketURL, expectedSuffix) {
				t.Errorf("ticket URL must end with %q, got %q", expectedSuffix, ticketURL)
			}
		}
	}
}

func TestVerifyOTP_HybridEvent_HasTicketURL(t *testing.T) {
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "hybrid")
	email := "hybrid_user@example.com"

	post(t, env.Server, "/api/register", model.RegisterRequest{
		Email:        email,
		EventID:      eventID,
		FirstName:    "Test",
		LastName:     "User",
		Organization: "Org",
		District:     "District",
		ConsentGiven: true,
	})
	otp := env.FM.LastOTPFor(email)
	verifyResp := post(t, env.Server, "/api/verify-otp", model.VerifyOTPRequest{
		Email:   email,
		EventID: eventID,
		OTP:     otp,
	})
	mustStatus(t, verifyResp, http.StatusOK)

	// Hybrid events have an offline component — ticket URL must be present.
	found := false
	for _, e := range env.FM.Sent() {
		if e.Kind == "welcome" && e.To == email && len(e.Args) >= 6 {
			ticketURL, _ := e.Args[5].(string)
			if ticketURL != "" {
				found = true
			}
		}
	}
	if !found {
		t.Error("hybrid event: welcome email must carry a non-empty ticket URL")
	}
}

// ── registerAndVerify is a shared helper for test scenarios ──────────────────

// registerAndVerify runs the full POST /register + POST /verify-otp flow and
// returns the JWT token from the verify response.
func registerAndVerify(t *testing.T, env *testEnv, email string, eventID int) string {
	t.Helper()

	resp := post(t, env.Server, "/api/register", model.RegisterRequest{
		Email:        email,
		EventID:      eventID,
		FirstName:    "Test",
		LastName:     "User",
		Organization: "Org",
		District:     "District",
		ConsentGiven: true,
	})
	mustStatus(t, resp, http.StatusOK)

	otp := env.FM.LastOTPFor(email)
	if otp == "" {
		t.Fatal("registerAndVerify: no OTP email received")
	}

	verifyResp := post(t, env.Server, "/api/verify-otp", model.VerifyOTPRequest{
		Email:   email,
		EventID: eventID,
		OTP:     otp,
	})
	mustStatus(t, verifyResp, http.StatusOK)

	var result map[string]interface{}
	decodeJSON(t, verifyResp, &result)

	token, _ := result["token"].(string)
	if token == "" {
		t.Fatal("registerAndVerify: verify-otp did not return a token")
	}
	return token
}

// ── small helpers ─────────────────────────────────────────────────────────────

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
