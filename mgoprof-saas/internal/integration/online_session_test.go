package integration

// Module 1: Heartbeat / online_sessions
//
// Tests cover:
//   - connect creates session, returns session_uuid
//   - ping updates last_ping_at, returns 200
//   - disconnect closes session, duration_seconds populated
//   - double-connect (reload) closes old session, creates fresh one
//   - ping on a closed session returns 410
//   - timeout worker closes stale sessions
//   - admin stats endpoint returns active count and totals
//   - hybrid: online_sessions and QR check-in are independent channels

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"mgoprof-saas/internal/model"
)

// ── connect / ping / disconnect happy path ────────────────────────────────────

func TestOnlineSession_ConnectPingDisconnect(t *testing.T) {
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "online")
	token := registerAndVerify(t, env, "sess_happy@example.com", eventID)

	// 1. Connect
	connectResp := postWithToken(t, env.Server, pathSession(eventID, "connect"), token, nil)
	mustStatus(t, connectResp, http.StatusOK)

	var connectBody map[string]interface{}
	decodeJSON(t, connectResp, &connectBody)

	sessionUUID, ok := connectBody["session_uuid"].(string)
	if !ok || sessionUUID == "" {
		t.Fatal("connect: expected session_uuid in response")
	}

	// 2. Ping
	pingResp := postWithToken(t, env.Server, pathSession(eventID, "ping"), token,
		map[string]string{"session_uuid": sessionUUID})
	mustStatus(t, pingResp, http.StatusOK)

	// 3. Disconnect
	discResp := postWithToken(t, env.Server, pathSession(eventID, "disconnect"), token,
		map[string]string{"session_uuid": sessionUUID})
	mustStatus(t, discResp, http.StatusOK)

	// Verify session is closed in DB with end_reason=explicit and duration_seconds set.
	var endReason string
	var durationSeconds *int
	err := env.DB.QueryRowContext(context.Background(), `
		SELECT end_reason, duration_seconds
		FROM online_sessions
		WHERE session_uuid = $1`,
		sessionUUID,
	).Scan(&endReason, &durationSeconds)
	if err != nil {
		t.Fatalf("query session after disconnect: %v", err)
	}
	if endReason != "explicit" {
		t.Errorf("end_reason: want explicit, got %q", endReason)
	}
	if durationSeconds == nil {
		t.Error("duration_seconds must be set after explicit disconnect")
	}
}

// ── double connect (reload / dual-tab) ───────────────────────────────────────

func TestOnlineSession_ReconnectClosesStaleSession(t *testing.T) {
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "online")
	token := registerAndVerify(t, env, "sess_reconnect@example.com", eventID)

	// First connect
	r1 := postWithToken(t, env.Server, pathSession(eventID, "connect"), token, nil)
	mustStatus(t, r1, http.StatusOK)
	var b1 map[string]interface{}
	decodeJSON(t, r1, &b1)
	uuid1, _ := b1["session_uuid"].(string)

	// Second connect (simulates reload) — must close uuid1 and return a new UUID.
	r2 := postWithToken(t, env.Server, pathSession(eventID, "connect"), token, nil)
	mustStatus(t, r2, http.StatusOK)
	var b2 map[string]interface{}
	decodeJSON(t, r2, &b2)
	uuid2, _ := b2["session_uuid"].(string)

	if uuid1 == "" || uuid2 == "" {
		t.Fatal("both connects must return a session_uuid")
	}
	if uuid1 == uuid2 {
		t.Error("reconnect must produce a new session_uuid, not reuse the old one")
	}

	// First session must now be closed with end_reason=error
	var endReason1 string
	err := env.DB.QueryRowContext(context.Background(), `
		SELECT COALESCE(end_reason, '') FROM online_sessions WHERE session_uuid = $1`, uuid1,
	).Scan(&endReason1)
	if err != nil {
		t.Fatalf("query session1: %v", err)
	}
	if endReason1 != "error" {
		t.Errorf("stale session end_reason: want error, got %q", endReason1)
	}

	// Second session must be open
	var endReason2 *string
	err = env.DB.QueryRowContext(context.Background(), `
		SELECT end_reason FROM online_sessions WHERE session_uuid = $1`, uuid2,
	).Scan(&endReason2)
	if err != nil {
		t.Fatalf("query session2: %v", err)
	}
	if endReason2 != nil {
		t.Errorf("new session must be open (end_reason NULL), got %q", *endReason2)
	}
}

// ── ping on closed session returns 410 ───────────────────────────────────────

func TestOnlineSession_PingOnClosedSessionReturns410(t *testing.T) {
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "online")
	token := registerAndVerify(t, env, "sess_stale_ping@example.com", eventID)

	r := postWithToken(t, env.Server, pathSession(eventID, "connect"), token, nil)
	mustStatus(t, r, http.StatusOK)
	var b map[string]interface{}
	decodeJSON(t, r, &b)
	uuid, _ := b["session_uuid"].(string)

	// Disconnect first
	disc := postWithToken(t, env.Server, pathSession(eventID, "disconnect"), token,
		map[string]string{"session_uuid": uuid})
	mustStatus(t, disc, http.StatusOK)

	// Now ping the closed session — must return 410 Gone
	pingResp := postWithToken(t, env.Server, pathSession(eventID, "ping"), token,
		map[string]string{"session_uuid": uuid})
	mustStatus(t, pingResp, http.StatusGone)
}

// ── timeout worker ────────────────────────────────────────────────────────────

func TestOnlineSession_TimeoutWorkerClosesStale(t *testing.T) {
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "online")
	token := registerAndVerify(t, env, "sess_timeout@example.com", eventID)

	// Connect
	r := postWithToken(t, env.Server, pathSession(eventID, "connect"), token, nil)
	mustStatus(t, r, http.StatusOK)
	var b map[string]interface{}
	decodeJSON(t, r, &b)
	uuid, _ := b["session_uuid"].(string)

	// Force last_ping_at into the past so the session looks stale.
	_, err := env.DB.ExecContext(context.Background(), `
		UPDATE online_sessions
		SET last_ping_at = NOW() - INTERVAL '120 seconds'
		WHERE session_uuid = $1`, uuid)
	if err != nil {
		t.Fatalf("backdate last_ping_at: %v", err)
	}

	// Run timeout job
	n, err := env.OnlineSessionSvc.TimeoutStaleSessions(context.Background())
	if err != nil {
		t.Fatalf("TimeoutStaleSessions: %v", err)
	}
	if n < 1 {
		t.Errorf("expected at least 1 session timed out, got %d", n)
	}

	// Verify the session is now closed with end_reason=timeout
	var endReason string
	var durSec *int
	err = env.DB.QueryRowContext(context.Background(), `
		SELECT COALESCE(end_reason,''), duration_seconds
		FROM online_sessions WHERE session_uuid = $1`, uuid,
	).Scan(&endReason, &durSec)
	if err != nil {
		t.Fatalf("query after timeout: %v", err)
	}
	if endReason != "timeout" {
		t.Errorf("end_reason: want timeout, got %q", endReason)
	}
	if durSec == nil {
		t.Error("duration_seconds must be populated after timeout")
	}
}

// ── admin stats ───────────────────────────────────────────────────────────────

func TestOnlineSession_AdminStats(t *testing.T) {
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "online")

	// Register two participants.
	tok1 := registerAndVerify(t, env, "sess_admin1@example.com", eventID)
	tok2 := registerAndVerify(t, env, "sess_admin2@example.com", eventID)

	// Both connect.
	r1 := postWithToken(t, env.Server, pathSession(eventID, "connect"), tok1, nil)
	mustStatus(t, r1, http.StatusOK)
	r2 := postWithToken(t, env.Server, pathSession(eventID, "connect"), tok2, nil)
	mustStatus(t, r2, http.StatusOK)

	// Admin stats via service (direct call — avoids needing admin JWT in test).
	stats, err := env.OnlineSessionSvc.AdminStats(context.Background(), eventID)
	if err != nil {
		t.Fatalf("AdminStats: %v", err)
	}

	if stats.ActiveNow != 2 {
		t.Errorf("active_now: want 2, got %d", stats.ActiveNow)
	}
	if len(stats.Registrations) < 2 {
		t.Errorf("registrations: want at least 2, got %d", len(stats.Registrations))
	}
}

// ── hybrid: independent channels ─────────────────────────────────────────────

func TestOnlineSession_HybridChannelsAreIndependent(t *testing.T) {
	env := newTestEnv(t)
	// Hybrid event — supports both online stream and physical attendance.
	eventID := seedEvent(t, env.DB, "hybrid")

	token := registerAndVerify(t, env, "sess_hybrid@example.com", eventID)

	// Online: connect
	connResp := postWithToken(t, env.Server, pathSession(eventID, "connect"), token, nil)
	mustStatus(t, connResp, http.StatusOK)

	// Verify: one open online_session, zero attendance_events
	var onlineCount int
	if err := env.DB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM online_sessions WHERE event_id = $1 AND ended_at IS NULL`,
		eventID).Scan(&onlineCount); err != nil {
		t.Fatal(err)
	}
	var offlineCount int
	if err := env.DB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM reg_attendance_events WHERE event_id = $1`,
		eventID).Scan(&offlineCount); err != nil {
		t.Fatal(err)
	}

	if onlineCount != 1 {
		t.Errorf("online sessions: want 1, got %d", onlineCount)
	}
	if offlineCount != 0 {
		t.Errorf("attendance_events: want 0 (online connect must not touch offline channel), got %d", offlineCount)
	}
}

// ── Module 2: ResolveEventLink unit test (no DB required) ────────────────────

func TestResolveEventLink_AllRoles(t *testing.T) {
	// We test the Go function directly — this is the canonical source.
	// The SQL CASE in ListByUserVerified must mirror this logic.
	type tc struct {
		role        string
		speakerLink string
		viewerLink  string
		cabinetLink string
		wantLink    string
	}

	cases := []tc{
		{
			role: "speaker", speakerLink: "https://spk.example.com",
			viewerLink: "https://view.example.com", cabinetLink: "https://cab.example.com",
			wantLink: "https://spk.example.com",
		},
		{
			role: "moderator", speakerLink: "https://spk.example.com",
			viewerLink: "https://view.example.com", cabinetLink: "https://cab.example.com",
			wantLink: "https://spk.example.com",
		},
		{
			role: "viewer", speakerLink: "https://spk.example.com",
			viewerLink: "https://view.example.com", cabinetLink: "https://cab.example.com",
			wantLink: "https://view.example.com",
		},
		{
			role: "delegate", speakerLink: "https://spk.example.com",
			viewerLink: "https://view.example.com", cabinetLink: "https://cab.example.com",
			wantLink: "https://view.example.com",
		},
		{
			role: "guest", speakerLink: "https://spk.example.com",
			viewerLink: "https://view.example.com", cabinetLink: "https://cab.example.com",
			wantLink: "https://view.example.com",
		},
		// Fallback: speaker link empty → viewer_link used for speakers
		{
			role: "speaker", speakerLink: "",
			viewerLink: "https://view.example.com", cabinetLink: "https://cab.example.com",
			wantLink: "https://view.example.com",
		},
		// Fallback: both stream links empty → cabinet_link
		{
			role: "viewer", speakerLink: "",
			viewerLink: "", cabinetLink: "https://cab.example.com",
			wantLink: "https://cab.example.com",
		},
	}

	for _, tc := range cases {
		got := model.ResolveEventLink(tc.role, tc.speakerLink, tc.viewerLink, tc.cabinetLink)
		if got != tc.wantLink {
			t.Errorf("role=%q speakerLink=%q viewerLink=%q → got %q, want %q",
				tc.role, tc.speakerLink, tc.viewerLink, got, tc.wantLink)
		}
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func pathSession(eventID int, action string) string {
	return fmt.Sprintf("/api/session/%d/%s", eventID, action)
}
