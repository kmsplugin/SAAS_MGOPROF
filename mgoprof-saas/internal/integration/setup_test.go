// Package integration contains smoke / integration tests that exercise the
// full HTTP → service → repository → PostgreSQL stack.
//
// These tests require a real PostgreSQL database. Set TEST_DATABASE_URL to run
// them:
//
//	TEST_DATABASE_URL="postgres://user:pass@localhost:5432/mgoprof_test?sslmode=disable" \
//	  go test ./internal/integration/... -v
//
// When TEST_DATABASE_URL is not set the entire package is skipped so that
// `go test ./...` (without external dependencies) still passes.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"mgoprof-saas/internal/handler"
	"mgoprof-saas/internal/middleware"
	"mgoprof-saas/internal/repository"
	"mgoprof-saas/internal/service"
)

// ── Constants used across all tests ──────────────────────────────────────────

const (
	testJWTSecret = "integration_test_secret_minimum_32_characters_ok"
	testSiteURL   = "https://test.example.com"
)

// ── FakeMailer ────────────────────────────────────────────────────────────────

// FakeMailer captures every outgoing email and can be forced to fail.
// It satisfies both service.RegistrationMailer and service.AuthMailer.
type FakeMailer struct {
	mu       sync.Mutex
	sent     []FakeEmail
	failMode bool
}

// FakeEmail holds a captured outgoing email.
type FakeEmail struct {
	Kind   string        // "registration" | "welcome" | "new_password" | "data_export"
	To     string
	Args   []interface{} // method-specific positional args
	SentAt time.Time
}

// SetFailMode forces the next send call(s) to return an error.
func (f *FakeMailer) SetFailMode(fail bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failMode = fail
}

// Sent returns a snapshot of all captured emails (safe to call from any goroutine).
func (f *FakeMailer) Sent() []FakeEmail {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]FakeEmail, len(f.sent))
	copy(out, f.sent)
	return out
}

// Reset discards all captured emails.
func (f *FakeMailer) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = nil
}

// SentTo returns true if at least one email of the given kind was sent to addr.
func (f *FakeMailer) SentTo(kind, to string) bool {
	for _, e := range f.Sent() {
		if e.Kind == kind && e.To == to {
			return true
		}
	}
	return false
}

// LastOTPFor returns the OTP code from the most recent registration email to addr.
func (f *FakeMailer) LastOTPFor(to string) string {
	sent := f.Sent()
	for i := len(sent) - 1; i >= 0; i-- {
		e := sent[i]
		if e.Kind == "registration" && e.To == to && len(e.Args) >= 2 {
			if otp, ok := e.Args[1].(string); ok {
				return otp
			}
		}
	}
	return ""
}

// WelcomePasswordFor returns the password from the most recent welcome email to addr.
func (f *FakeMailer) WelcomePasswordFor(to string) string {
	sent := f.Sent()
	for i := len(sent) - 1; i >= 0; i-- {
		e := sent[i]
		if e.Kind == "welcome" && e.To == to && len(e.Args) >= 2 {
			if pwd, ok := e.Args[1].(string); ok {
				return pwd
			}
		}
	}
	return ""
}

// ── service.RegistrationMailer implementation ─────────────────────────────────

func (f *FakeMailer) SendRegistration(to, firstName, otp, eventTitle string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failMode {
		return fmt.Errorf("fake mailer: forced failure on SendRegistration")
	}
	f.sent = append(f.sent, FakeEmail{
		Kind:   "registration",
		To:     to,
		Args:   []interface{}{firstName, otp, eventTitle},
		SentAt: time.Now(),
	})
	return nil
}

func (f *FakeMailer) SendWelcome(to, firstName, password string, userID, regID int, cabinetURL, ticketURL string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failMode {
		return fmt.Errorf("fake mailer: forced failure on SendWelcome")
	}
	f.sent = append(f.sent, FakeEmail{
		Kind:   "welcome",
		To:     to,
		Args:   []interface{}{firstName, password, userID, regID, cabinetURL, ticketURL},
		SentAt: time.Now(),
	})
	return nil
}

// ── service.AuthMailer implementation ─────────────────────────────────────────

func (f *FakeMailer) SendNewPassword(to, firstName, password, loginURL string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failMode {
		return fmt.Errorf("fake mailer: forced failure on SendNewPassword")
	}
	f.sent = append(f.sent, FakeEmail{
		Kind:   "new_password",
		To:     to,
		Args:   []interface{}{firstName, password, loginURL},
		SentAt: time.Now(),
	})
	return nil
}

func (f *FakeMailer) SendDataExport(to, firstName, dataJSON string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failMode {
		return fmt.Errorf("fake mailer: forced failure on SendDataExport")
	}
	f.sent = append(f.sent, FakeEmail{
		Kind:   "data_export",
		To:     to,
		Args:   []interface{}{firstName, dataJSON},
		SentAt: time.Now(),
	})
	return nil
}

// ── Test environment ──────────────────────────────────────────────────────────

// testEnv bundles the live test server and all collaborators needed in tests.
type testEnv struct {
	Server           *httptest.Server
	DB               *sqlx.DB
	FM               *FakeMailer
	AuthSvc          *service.AuthService
	RegRepo          *repository.RegistrationRepository
	UserRepo         *repository.UserRepository
	OnlineSessionSvc *service.OnlineSessionService
}

// newTestEnv opens a test DB, applies migrations (idempotent), cleans state,
// wires the full Gin router, and returns a running httptest.Server.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping integration tests")
	}

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatalf("connect to test DB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	applyMigrations(t, db)
	cleanTables(t, db)
	t.Cleanup(func() { cleanTables(t, db) })

	fm := &FakeMailer{}
	logger := zap.NewNop()

	// ── Repositories ──────────────────────────────────────────────────────────
	userRepo     := repository.NewUserRepository(db)
	eventRepo    := repository.NewEventRepository(db)
	regRepo      := repository.NewRegistrationRepository(db)
	logRepo      := repository.NewLogRepository(db)
	fieldRepo    := repository.NewFieldRepository(db)
	consentRepo  := repository.NewConsentRepository(db)
	trackingRepo := repository.NewTrackingRepository(db)
	scanRepo          := repository.NewScanRepository(db)
	refListRepo       := repository.NewRefListRepository(db)
	onlineSessionRepo := repository.NewOnlineSessionRepository(db)

	// ── Services ──────────────────────────────────────────────────────────────
	geo      := service.NewGeoResolver("", "", logger)
	authSvc  := service.NewAuthService(userRepo, regRepo, consentRepo, logRepo, fm, geo, logger,
		testJWTSecret, testSiteURL)
	fieldSvc         := service.NewFieldService(fieldRepo, logger)
	regSvc           := service.NewRegistrationService(userRepo, eventRepo, regRepo, fieldRepo,
		logRepo, consentRepo, fm, authSvc, geo, logger, testSiteURL)
	trackingSvc      := service.NewTrackingService(trackingRepo, regRepo, geo, logger)
	scanSvc          := service.NewScanService(scanRepo, eventRepo, logger)
	onlineSessionSvc := service.NewOnlineSessionService(onlineSessionRepo, regRepo, logger)

	// ── Gin router ────────────────────────────────────────────────────────────
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())

	regHandler          := handler.NewRegistrationHandler(regSvc, logger)
	authHandler         := handler.NewAuthHandler(authSvc, regRepo, logger)
	onlineSessionHandler := handler.NewOnlineSessionHandler(onlineSessionSvc, logger)
	// AdminPanelHandler with nil adminSvc — attendance page doesn't use adminSvc.
	adminPanelHandler := handler.NewAdminPanelHandler(
		eventRepo, regRepo, fieldSvc, refListRepo,
		scanSvc, trackingSvc, nil, logger,
	)

	authMW := middleware.Auth(authSvc)

	// Admin panel HTML routes (no role check in test — role check is unit-tested
	// in middleware tests; here we focus on the business logic layer).
	adminPanelHandler.RegisterRoutes(r.Group(""), authMW)

	api := r.Group("/api")
	{
		api.POST("/register",   regHandler.HandleRegister)
		api.POST("/verify-otp", regHandler.HandleVerifyOTP)
		authHandler.RegisterRoutes(api, authMW)
		onlineSessionHandler.RegisterRoutes(api, authMW)
	}

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	return &testEnv{
		Server:           srv,
		DB:               db,
		FM:               fm,
		AuthSvc:          authSvc,
		RegRepo:          regRepo,
		UserRepo:         userRepo,
		OnlineSessionSvc: onlineSessionSvc,
	}
}

// ── Migration runner ──────────────────────────────────────────────────────────

func applyMigrations(t *testing.T, db *sqlx.DB) {
	t.Helper()

	// From internal/integration/ → ../../migrations/
	migrDir := filepath.Join("..", "..", "migrations")
	entries, err := filepath.Glob(filepath.Join(migrDir, "*.up.sql"))
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("no migration files found in %s", migrDir)
	}
	sort.Strings(entries)

	for _, path := range entries {
		sql, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", path, err)
		}
		if _, err := db.ExecContext(context.Background(), string(sql)); err != nil {
			// Idempotent: ignore "already exists" / "duplicate" errors so that
			// re-running tests against the same DB schema works cleanly.
			msg := err.Error()
			if strings.Contains(msg, "already exists") ||
				strings.Contains(msg, "duplicate column") ||
				strings.Contains(msg, "duplicate key") {
				continue
			}
			t.Fatalf("apply migration %s: %v", filepath.Base(path), err)
		}
	}
}

// cleanTables truncates all transient tables between tests for isolation.
func cleanTables(t *testing.T, db *sqlx.DB) {
	t.Helper()
	tables := []string{
		"online_sessions",
		"reg_tracking",
		"reg_scan_logs",
		"reg_attendance_events",
		"reg_field_answers",
		"reg_questions",
		"reg_consents",
		"reg_registrations",
		"reg_users",
		"reg_logs",
		"reg_events",
	}
	for _, tbl := range tables {
		if _, err := db.ExecContext(context.Background(), "DELETE FROM "+tbl); err != nil {
			// Best-effort: table might not exist in all schema versions.
			_ = err
		}
	}
}

// ── Seed helpers ──────────────────────────────────────────────────────────────

// seedEvent inserts a minimal active event and returns its ID.
func seedEvent(t *testing.T, db *sqlx.DB, eventType string) int {
	t.Helper()
	var id int
	err := db.QueryRowContext(context.Background(), `
		INSERT INTO reg_events (title, description, event_date, event_time, is_active, event_type)
		VALUES ($1, '', '2027-01-01', '10:00:00', true, $2)
		RETURNING id`,
		"Integration Test Event ("+eventType+")", eventType,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedEvent: %v", err)
	}
	return id
}

// seedUser inserts a user and returns their ID.
func seedUser(t *testing.T, db *sqlx.DB, email, firstName, lastName string) int {
	t.Helper()
	var id int
	err := db.QueryRowContext(context.Background(), `
		INSERT INTO reg_users (email, first_name, last_name, organization, district, is_union_member)
		VALUES ($1, $2, $3, 'TestOrg', 'TestDistrict', false)
		RETURNING id`,
		email, firstName, lastName,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedUser(%s): %v", email, err)
	}
	return id
}

// getOTPFromDB reads the current otp_code for a user+event registration.
func getOTPFromDB(t *testing.T, db *sqlx.DB, email string, eventID int) string {
	t.Helper()
	var otp string
	err := db.QueryRowContext(context.Background(), `
		SELECT COALESCE(r.otp_code, '')
		FROM reg_registrations r
		JOIN reg_users u ON u.id = r.user_id
		WHERE u.email = $1 AND r.event_id = $2`,
		email, eventID,
	).Scan(&otp)
	if err != nil {
		t.Fatalf("getOTPFromDB(%s, %d): %v", email, eventID, err)
	}
	return otp
}

// ── HTTP helpers ──────────────────────────────────────────────────────────────

// post sends a JSON POST to path and returns the response.
func post(t *testing.T, srv *httptest.Server, path string, body interface{}) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal POST %s body: %v", path, err)
	}
	resp, err := http.Post(srv.URL+path, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

// postWithToken sends a POST with a Bearer token.
func postWithToken(t *testing.T, srv *httptest.Server, path, token string, body interface{}) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal POST %s body: %v", path, err)
	}
	req, err := http.NewRequest(http.MethodPost, srv.URL+path, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("new POST request %s: %v", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s with token: %v", path, err)
	}
	return resp
}

// getWithToken sends a GET with a Bearer token.
func getWithToken(t *testing.T, srv *httptest.Server, path, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, srv.URL+path, nil)
	if err != nil {
		t.Fatalf("new GET request %s: %v", path, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s with token: %v", path, err)
	}
	return resp
}

// decodeJSON reads the response body into dst and closes the body.
func decodeJSON(t *testing.T, resp *http.Response, dst interface{}) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("decode JSON response (status %d): %v", resp.StatusCode, err)
	}
}

// mustStatus fails the test if the response status does not match want.
func mustStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		resp.Body.Close()
		t.Fatalf("expected HTTP %d, got %d", want, resp.StatusCode)
	}
}
