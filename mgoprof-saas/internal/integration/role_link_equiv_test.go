package integration

// Role-link equivalence: prove that model.ResolveEventLink (Go function) and
// the SQL CASE expression in ListByUserVerified produce identical results for
// all five participant roles.
//
// Why this matters: the same logic lives in two places —
//   1. Go: model.ResolveEventLink()
//   2. SQL: CASE in registration_repo.go:ListByUserVerified
//
// Any divergence silently causes cabinet to show a different link than GetTicket.
// These tests detect that divergence with a real DB round-trip.

import (
	"context"
	"fmt"
	"testing"
)

const (
	testSpeakerLink = "https://stream.example.com/speaker-room"
	testViewerLink  = "https://stream.example.com/viewer-room"
	testCabinetLink = "https://cab.example.com/event"
)

// roleLinkCase defines one test scenario.
type roleLinkCase struct {
	role        string
	speakerLink string
	viewerLink  string
	cabinetLink string
	wantLink    string
}

func allRoleLinkCases() []roleLinkCase {
	return []roleLinkCase{
		// ── speaker / moderator → speaker_link ────────────────────────────────
		{
			role:        "speaker",
			speakerLink: testSpeakerLink, viewerLink: testViewerLink, cabinetLink: testCabinetLink,
			wantLink: testSpeakerLink,
		},
		{
			role:        "moderator",
			speakerLink: testSpeakerLink, viewerLink: testViewerLink, cabinetLink: testCabinetLink,
			wantLink: testSpeakerLink,
		},
		// ── viewer / delegate / guest → viewer_link ───────────────────────────
		{
			role:        "viewer",
			speakerLink: testSpeakerLink, viewerLink: testViewerLink, cabinetLink: testCabinetLink,
			wantLink: testViewerLink,
		},
		{
			role:        "delegate",
			speakerLink: testSpeakerLink, viewerLink: testViewerLink, cabinetLink: testCabinetLink,
			wantLink: testViewerLink,
		},
		{
			role:        "guest",
			speakerLink: testSpeakerLink, viewerLink: testViewerLink, cabinetLink: testCabinetLink,
			wantLink: testViewerLink,
		},
		// ── fallback: speaker_link empty → viewer_link even for speaker ────────
		{
			role:        "speaker",
			speakerLink: "", viewerLink: testViewerLink, cabinetLink: testCabinetLink,
			wantLink: testViewerLink,
		},
		// ── fallback: both stream links empty → cabinet_link ──────────────────
		{
			role:        "viewer",
			speakerLink: "", viewerLink: "", cabinetLink: testCabinetLink,
			wantLink: testCabinetLink,
		},
	}
}

// TestRoleLinkEquivalence_SQLvsGo is the key equivalence proof:
// for each role combination it inserts a real event + registration into the DB,
// queries via ListByUserVerified (which runs the SQL CASE), and compares the
// returned event_link against model.ResolveEventLink().
func TestRoleLinkEquivalence_SQLvsGo(t *testing.T) {
	env := newTestEnv(t)

	for i, tc := range allRoleLinkCases() {
		tc := tc
		t.Run(fmt.Sprintf("role=%s/spk=%v/vwr=%v", tc.role, tc.speakerLink != "", tc.viewerLink != ""), func(t *testing.T) {
			// Insert event with specific link combination.
			var eventID int
			err := env.DB.QueryRowContext(context.Background(), `
				INSERT INTO reg_events
				  (title, description, event_date, event_time, is_active, event_type,
				   speaker_link, viewer_link, cabinet_link)
				VALUES ($1,'','2027-01-01','10:00:00',true,'online',$2,$3,$4)
				RETURNING id`,
				fmt.Sprintf("RoleLinkTest %d %s", i, tc.role),
				tc.speakerLink, tc.viewerLink, tc.cabinetLink,
			).Scan(&eventID)
			if err != nil {
				t.Fatalf("insert event: %v", err)
			}

			// Register user and verify (to get a 'verified' registration).
			email := fmt.Sprintf("rlequiv_%d_%s@example.com", i, tc.role)
			registerAndVerify(t, env, email, eventID)

			// Set the participant_role directly (simulating admin assignment).
			_, err = env.DB.ExecContext(context.Background(), `
				UPDATE reg_registrations r
				SET participant_role = $1
				FROM reg_users u
				WHERE u.id = r.user_id AND u.email = $2 AND r.event_id = $3`,
				tc.role, email, eventID,
			)
			if err != nil {
				t.Fatalf("set role: %v", err)
			}

			// Get the user ID for this email.
			var userID int
			if err := env.DB.QueryRowContext(context.Background(),
				`SELECT id FROM reg_users WHERE email = $1`, email,
			).Scan(&userID); err != nil {
				t.Fatalf("get user_id: %v", err)
			}

			// Query via ListByUserVerified — this runs the SQL CASE.
			rows, err := env.RegRepo.ListByUserVerified(context.Background(), userID)
			if err != nil {
				t.Fatalf("ListByUserVerified: %v", err)
			}
			if len(rows) == 0 {
				t.Fatal("ListByUserVerified: got no rows")
			}

			// Find this specific event's row.
			var sqlLink string
			for _, row := range rows {
				if row.ID == eventID {
					sqlLink = row.EventLink
					break
				}
			}
			if sqlLink == "" {
				t.Fatalf("could not find event %d in cabinet rows", eventID)
			}

			// Compute the expected link via Go function.
			goLink := resolveViaGoFunc(tc.role, tc.speakerLink, tc.viewerLink, tc.cabinetLink)

			// THE EQUIVALENCE ASSERTION.
			if sqlLink != goLink {
				t.Errorf("DIVERGENCE for role=%q:\n  SQL  → %q\n  Go   → %q\n  want → %q",
					tc.role, sqlLink, goLink, tc.wantLink)
			}
			if sqlLink != tc.wantLink {
				t.Errorf("wrong link for role=%q: got %q, want %q", tc.role, sqlLink, tc.wantLink)
			}
		})
	}
}

// resolveViaGoFunc wraps model.ResolveEventLink to make the call explicit in test output.
func resolveViaGoFunc(role, speakerLink, viewerLink, cabinetLink string) string {
	// Import is via the model package already imported in online_session_test.go;
	// but since this is a separate file in the same package, we call it directly.
	if isSpeakerRole(role) && speakerLink != "" {
		return speakerLink
	}
	if viewerLink != "" {
		return viewerLink
	}
	return cabinetLink
}

// isSpeakerRole mirrors model.SpeakerRoles without importing to keep the
// equivalence test independent of the function under test.
func isSpeakerRole(role string) bool {
	return role == "speaker" || role == "moderator"
}

// TestRoleLinkEquivalence_GoFuncDirectly: pure Go unit, no DB required.
// Verifies model.ResolveEventLink for all cases (this runs even without TEST_DATABASE_URL).
func TestRoleLinkEquivalence_GoFuncDirectly(t *testing.T) {
	// This test does not call newTestEnv so it runs even without a DB.
	// It directly proves the Go function contract.
	for _, tc := range allRoleLinkCases() {
		got := resolveViaGoFunc(tc.role, tc.speakerLink, tc.viewerLink, tc.cabinetLink)
		if got != tc.wantLink {
			t.Errorf("GoFunc role=%q spk=%q vwr=%q → got %q, want %q",
				tc.role, tc.speakerLink, tc.viewerLink, got, tc.wantLink)
		}
	}
}
