# Architecture Decision Records

This file documents technical decisions that have non-obvious trade-offs or that
could be disputed later. Each record includes the context, the decision, and the
known alternatives.

---

## ADR-001 — Hidden-tab behaviour for online_sessions heartbeat

**Date:** 2026-03-22
**Status:** Accepted
**Stakeholders:** Product, Backend, Analytics

### Context

The ticket page sends a heartbeat ping to `POST /api/session/:event_id/ping`
every 25 seconds while the participant tab is active. When the tab is hidden
(`document.visibilityState === 'hidden'`) the JavaScript timer fires but
`apiPost()` silently skips the call:

```js
if (document.visibilityState === 'hidden') return; // skip while tab hidden
```

This means a participant who navigates away from the ticket page without closing
it will stop sending pings. After 90 seconds of silence the background worker
marks the session as timed out (`end_reason = 'timeout'`).

### Decision

**Hidden-tab users are NOT counted as active participants.**

Rationale:
1. **Engagement definition.** The event owner wants to know how many people are
   *watching* the stream, not how many have an open browser window. A minimised
   tab indicates the user is doing something else.
2. **Resource protection.** Sending pings from hidden tabs wastes DB writes with
   no business value.
3. **Fairness.** Participation time reported to the user (and in admin stats)
   should reflect actual screen-time, not idle open-tab time.

### Consequence

- A participant who switches to another application during the event will have
  their session timed out after ≈90 s.
- When they return to the ticket tab and it becomes visible again, `visibilitychange`
  fires a ping immediately. If the session is still within the 90-second window it
  resumes; otherwise a new session is created on the next heartbeat cycle.
  **The participant is responsible for returning to the tab to resume tracking.**
- Reported "total participation time" will be slightly lower than wall-clock
  attendance time. This is by design and matches the engagement definition above.

### Alternatives considered

1. **Count all open sessions regardless of visibility.** Rejected — inflates numbers
   and does not reflect actual viewing.
2. **Pause timer but keep session open for longer (e.g. 10 min).** Possible future
   enhancement if event owners request it. Would require a configurable timeout
   per event type.
3. **Use Page Visibility API to send a ping on `visibilitychange → visible`.** Already
   implemented — mitigates short alt-tab periods (< 90 s) without counting
   prolonged absence.

---

## ADR-002 — TEST_MODE OTP endpoint

**Date:** 2026-03-22
**Status:** Accepted

### Context

End-to-end Playwright tests need to complete the OTP verification step without
access to the real SMTP mailbox. In CI there is no mail server.

### Decision

When the server is started with `TEST_MODE=true`, the route
`GET /api/test/last-otp?email=...` is registered. It returns the most recent
unexpired OTP code for the given email directly from the database.

Security contract:
- Route is **not registered** when `TEST_MODE` is unset or `false`.
- The server logs `WARN TEST_MODE enabled` at startup to make accidental
  production use visible in logs.
- The OTP value itself is **not logged** (log says "served", not the code).
- CI smoke job sets `TEST_MODE=true` only for the smoke suite step; the integration
  test job leaves it unset.

### Alternatives considered

1. **Shared SMTP test inbox (MailHog/Mailpit).** Clean but adds a service to CI
   topology and requires HTML parsing. Overkill for a 6-digit code.
2. **Inject OTP via env var / API seeding.** Would require additional seeding
   infrastructure. The DB-read approach is simpler and requires no extra state.

---

## ADR-003 — SQL CASE vs Go function for role-based event_link resolution

**Date:** 2026-03-22
**Status:** Accepted

### Context

The resolved stream link for a participant depends on their `participant_role`:
- Speaker/moderator roles → `event.speaker_link`
- All others → `event.viewer_link` (fallback `event.cabinet_link`)

This logic exists in two places:
1. **`model.ResolveEventLink()`** — called by `ticket_service.go` at HTTP request time.
2. **SQL CASE** in `registration_repo.go:ListByUserVerified()` — used by the cabinet
   events list endpoint to include `event_link` in the JSON response.

### Decision

Both paths are intentionally kept in sync. `TestRoleLinkEquivalence_SQLvsGo` in
`internal/integration/role_link_equiv_test.go` creates real DB registrations for
each role and asserts that both code paths produce identical results. If they
diverge the test fails with an explicit `DIVERGENCE for role=...` message.

**Single source of truth is `model.ResolveEventLink()`**; the SQL CASE is
a performance optimisation for the list query and must always be kept equivalent.

### Consequence

Any change to role logic must update both the Go function and the SQL CASE, and
the equivalence test must remain green.

---

## ADR-004 — sendBeacon disconnect uses ?token= query param

**Date:** 2026-03-22
**Status:** Accepted

### Context

`navigator.sendBeacon()` cannot set custom HTTP headers (including `Authorization`).
The disconnect call is sent via sendBeacon on `beforeunload` for best-effort
cleanup when the user closes the page.

### Decision

The disconnect URL is appended with `?token=<jwt>` when called via sendBeacon.
The auth middleware already accepts `?token=` as a fallback (used by browser
HTML pages that cannot set headers). No special handling is needed server-side.

### Consequence

The JWT appears in the URL on the `beforeunload` request. This is visible in
server access logs but acceptable because:
1. The token is already held in the browser cookie (same domain).
2. The request is fire-and-forget and does not return sensitive data.
3. The risk is limited to access log exposure on the server side.

If stricter confidentiality is required in future, the alternative is a dedicated
unauthenticated disconnect endpoint that accepts only a `session_uuid` (a random
UUID with no user identity embedded).

---

## Running E2E Locally

### Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| Go   | 1.23+   | `go install` or system package |
| Node | 22+     | `nvm install 22` |
| PostgreSQL | 16 | local or Docker |
| `psql` CLI | any | ships with PostgreSQL client |

### Step 1 — Start PostgreSQL

```bash
# Docker (quickest)
docker run -d --name mgoprof-test \
  -e POSTGRES_USER=workos -e POSTGRES_PASSWORD=workos_secret \
  -e POSTGRES_DB=workos \
  -p 5432:5432 postgres:16-alpine

export DATABASE_URL="postgres://workos:workos_secret@localhost:5432/workos?sslmode=disable"
```

### Step 2 — Apply migrations

```bash
cd mgoprof-saas
for f in $(ls migrations/*.up.sql | sort); do
  echo "Applying $f"
  psql "$DATABASE_URL" -f "$f" 2>/dev/null || true
done
```

### Step 3 — Seed a test event (required for registration tests)

```bash
psql "$DATABASE_URL" <<'SQL'
INSERT INTO reg_events (
  title, event_type, event_date, event_time,
  is_active, viewer_link, speaker_link, cabinet_link,
  created_at, updated_at
)
SELECT
  'CI Smoke Event', 'online', CURRENT_DATE, '09:00:00',
  true,
  'https://stream.example.com/watch',
  'https://stream.example.com/speak',
  'https://cabinet.example.com',
  NOW(), NOW()
WHERE NOT EXISTS (
  SELECT 1 FROM reg_events WHERE title = 'CI Smoke Event'
);
SQL
```

### Step 4 — Start the server with TEST_MODE

```bash
cd mgoprof-saas
export JWT_SECRET="dev_secret_min_32_chars_here_ok"
export TEST_MODE="true"   # exposes /api/test/last-otp — NEVER use in production
export GIN_MODE="debug"
export SITE_URL="http://localhost:8080"
export ALLOWED_ORIGIN="*"
export SMTP_HOST=""       # disable real email in development
go run ./cmd/server
```

The server logs `WARN TEST_MODE enabled — test endpoints are active` at startup.
If you see this in production logs, `TEST_MODE=true` was accidentally set — fix immediately.

### Step 5 — Install Playwright

```bash
cd e2e
npm ci
npx playwright install chromium --with-deps
```

### Step 6 — Run the smoke suite

```bash
cd e2e
BASE_URL=http://localhost:8080 TEST_MODE=true npx playwright test
```

Expected output: all tests pass. Browser tests navigate to `/api/cabinet/events/:id/ticket`
and verify the ticket HTML page renders correctly.

### Step 7 — View the report

```bash
npx playwright show-report
```

### Fail-open / Fail-safe behaviour summary

| Component | Behaviour on failure | Safe? |
|-----------|---------------------|-------|
| Heartbeat `connect()` network error | silently swallowed (`catch(function(){})`) | fail-open — ticket page stays functional |
| Heartbeat `ping` network error | silently swallowed | fail-open — session times out server-side at 90 s |
| Heartbeat `disconnect` via sendBeacon | best-effort — may arrive after page close | fail-open — session timed out by worker |
| `TestMode` route not set | route does not exist (404) | fail-safe — production never exposes test data |
| `OnlineSessionService` nil in admin handler | falls back to legacy tracking counters | fail-safe — no crash, degraded display |
| `SummaryWithUsers` DB error | logged, empty participant list | fail-safe — attendance page still loads |

### TEST_MODE security boundary

`TEST_MODE=true` **must only** be set in:
- Local development environments
- CI/CD e2e jobs (e2e-smoke job in `.github/workflows/ci.yml`)

It must **never** be set in:
- Production Kubernetes/Docker deployments
- Staging environments that share production data
- Any environment accessible to end users

Enforcement: the route simply does not exist when the flag is unset. There is no
runtime bypass. If someone sets `TEST_MODE=true` in production the startup log emits
a `WARN` level message that should trigger an alert in any monitoring stack.
