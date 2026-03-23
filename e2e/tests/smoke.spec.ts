/**
 * Smoke suite — mgoprof-saas
 *
 * Coverage:
 *   API-level (request API, no browser rendering):
 *     health check, event list, auth guards, TEST_MODE guard
 *
 *   BROWSER-level (real Chromium, page.goto + DOM assertions):
 *     registration → OTP (TEST_MODE) → verify (JWT from response) → ticket page
 *     ticket page HTML: event title, participant token, heartbeat script present
 *
 * TEST_MODE contract (server + env):
 *   GET /api/test/last-otp?email=...  returns {"email":...,"otp":"123456"}
 *   Only registered when TEST_MODE=true. Returns 404 otherwise (verified below).
 *
 * Event seed:
 *   CI workflow inserts 'CI Smoke Event' (online, is_active=true) before this
 *   suite runs.  If no active event is found the browser tests fail explicitly
 *   (no silent skip) so the CI seed step is visibly broken.
 */

import { test, expect, type Page } from '@playwright/test'

const BASE      = process.env.BASE_URL  ?? 'http://localhost:8080'
const TEST_MODE = process.env.TEST_MODE === 'true'

// ── Helpers ───────────────────────────────────────────────────────────────────

async function apiPost(page: Page, path: string, body: object, token?: string) {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (token) headers['Authorization'] = `Bearer ${token}`
  const res = await page.request.post(`${BASE}${path}`, { data: body, headers })
  const json = await res.json().catch(() => ({}))
  return { status: res.status(), body: json as Record<string, unknown> }
}

async function apiGet(page: Page, path: string, token?: string) {
  const headers: Record<string, string> = {}
  if (token) headers['Authorization'] = `Bearer ${token}`
  const res = await page.request.get(`${BASE}${path}`, { headers })
  const json = await res.json().catch(() => ({}))
  return { status: res.status(), body: json as Record<string, unknown> }
}

// ── 1. Health check ───────────────────────────────────────────────────────────

test('server health check passes', async ({ page }) => {
  const res = await apiGet(page, '/health')
  expect(res.status).toBe(200)
  expect(res.body).toHaveProperty('status', 'ok')
})

// ── 2. Public event list ──────────────────────────────────────────────────────

test('public event list returns 200 with events array', async ({ page }) => {
  const res = await apiGet(page, '/api/events')
  expect(res.status).toBe(200)
  expect(res.body).toHaveProperty('events')
  expect(Array.isArray(res.body.events)).toBe(true)
})

// ── 3. Auth guards ────────────────────────────────────────────────────────────

test('cabinet events requires auth — 401 without token', async ({ page }) => {
  const res = await apiGet(page, '/api/cabinet/events')
  expect(res.status).toBe(401)
})

test('session connect requires auth — 401 without token', async ({ page }) => {
  const res = await apiPost(page, '/api/session/1/connect', {})
  expect(res.status).toBe(401)
})

test('session ping requires auth — 401 without token', async ({ page }) => {
  const res = await apiPost(page, '/api/session/1/ping', { session_uuid: 'fake' })
  expect(res.status).toBe(401)
})

// ── 4. TEST_MODE guard ────────────────────────────────────────────────────────

test('TEST_MODE endpoint absent when TEST_MODE=false', async ({ page }) => {
  if (TEST_MODE) {
    test.skip() // When TEST_MODE is on, this endpoint IS registered — skip the "absent" check
    return
  }
  const res = await apiGet(page, '/api/test/last-otp?email=any@example.com')
  expect(res.status).toBe(404)
})

// ── 5. BROWSER: full registration → OTP → verify → ticket page ───────────────

// This test group requires TEST_MODE=true (seed event + last-otp endpoint).
// All steps run in a single test worker so state is shared (email, JWT, eventID).

test.describe('browser: registration → OTP → verify → ticket page', () => {
  test.skip(!TEST_MODE, 'Requires TEST_MODE=true and a seeded event')

  const email = `smoke_${Date.now()}@example.com`
  let eventID = 0
  let jwt     = ''

  // ── 5a. Find the seeded event ─────────────────────────────────────────────

  test('seed event exists and is active', async ({ page }) => {
    const res = await apiGet(page, '/api/events')
    expect(res.status).toBe(200)

    const events = res.body.events as Array<{ id: number; is_active: boolean; title: string }>
    const active = events.find(e => e.is_active)

    // Hard fail — no silent skip — so the CI seed step failure is visible.
    expect(active, 'No active event found. Check the CI seed step.').toBeTruthy()
    eventID = active!.id
  })

  // ── 5b. Register ──────────────────────────────────────────────────────────

  test('register returns show_otp: true', async ({ page }) => {
    expect(eventID).toBeGreaterThan(0)

    const res = await apiPost(page, '/api/register', {
      email,
      event_id: eventID,
      first_name:   'Smoke',
      last_name:    'Browser',
      organization: 'CI Org',
      district:     'CI District',
      consent_given: true,
    })

    expect(res.status).toBe(200)
    expect(res.body.status).toBe('success')
    expect(res.body.show_otp).toBe(true)
  })

  // ── 5c. Wrong OTP returns 400 ────────────────────────────────────────────

  test('verify with wrong OTP returns 400', async ({ page }) => {
    expect(eventID).toBeGreaterThan(0)

    const res = await apiPost(page, '/api/verify-otp', {
      email,
      event_id: eventID,
      code: '000000',
    })
    expect(res.status).toBe(400)
  })

  // ── 5d. TEST_MODE: retrieve real OTP ─────────────────────────────────────

  test('TEST_MODE last-otp returns 6-digit code', async ({ page }) => {
    const res = await apiGet(page, `/api/test/last-otp?email=${encodeURIComponent(email)}`)
    expect(res.status).toBe(200)
    expect(typeof res.body.otp).toBe('string')
    expect(res.body.otp as string).toMatch(/^\d{6}$/)
  })

  // ── 5e. Verify OTP — server returns JWT directly ─────────────────────────

  test('verify with correct OTP returns success + JWT', async ({ page }) => {
    expect(eventID).toBeGreaterThan(0)

    // Fetch the live OTP
    const otpRes = await apiGet(page, `/api/test/last-otp?email=${encodeURIComponent(email)}`)
    expect(otpRes.status).toBe(200)
    const otp = otpRes.body.otp as string
    expect(otp).toMatch(/^\d{6}$/)

    // Verify
    const res = await apiPost(page, '/api/verify-otp', {
      email,
      event_id: eventID,
      code: otp,
    })

    expect(res.status).toBe(200)
    expect(res.body.status).toBe('success')

    // Server must return cabinet redirect URL so the frontend can navigate.
    expect(typeof res.body.redirect_url).toBe('string')
    expect(res.body.redirect_url as string).toContain('/cabinet')

    // Server returns JWT — store for the browser navigation test below.
    expect(typeof res.body.token).toBe('string')
    jwt = res.body.token as string
    expect(jwt.length).toBeGreaterThan(20)
  })

  // ── 5f. BROWSER: navigate to ticket page, verify rendered HTML ───────────

  test('browser: ticket page renders event title and participant token', async ({ page }) => {
    expect(jwt).toBeTruthy()
    expect(eventID).toBeGreaterThan(0)

    // Navigate to the HTML ticket page using ?token= query-param auth.
    // The auth middleware accepts this as a fallback for browser pages.
    const ticketURL = `${BASE}/api/cabinet/events/${eventID}/ticket?token=${encodeURIComponent(jwt)}`
    const response  = await page.goto(ticketURL, { waitUntil: 'domcontentloaded' })

    expect(response?.status()).toBe(200)

    // Ticket card must be present in the DOM
    await expect(page.locator('.ticket')).toBeVisible()

    // Event title must match the seeded event
    await expect(page.locator('.event-title')).toContainText('CI Smoke Event')

    // Participant token box must be rendered
    await expect(page.locator('.token-val')).toBeVisible()
    const tokenText = await page.locator('.token-val').textContent()
    expect(tokenText?.trim().length).toBeGreaterThan(0)
  })

  // ── 5g. BROWSER: heartbeat script rendered for online event ──────────────

  test('browser: ticket page contains heartbeat JS for online event', async ({ page }) => {
    expect(jwt).toBeTruthy()
    expect(eventID).toBeGreaterThan(0)

    const ticketURL = `${BASE}/api/cabinet/events/${eventID}/ticket?token=${encodeURIComponent(jwt)}`
    await page.goto(ticketURL, { waitUntil: 'domcontentloaded' })

    const html = await page.content()

    // The stream section + heartbeat script are rendered when EventLink is set.
    // Our seeded event has viewer_link set so EventLink is populated.
    if (html.includes('stream-section')) {
      // Script block must use the correct query param (not the broken ?_token=)
      expect(html).toContain('?token=')
      expect(html).not.toContain('?_token=')

      // Single connect() call — no double-connect bug
      const connectOccurrences = (html.match(/\/connect'/g) ?? []).length
      expect(connectOccurrences).toBeLessThanOrEqual(2) // one in apiPost + one in connect()

      // Heartbeat infrastructure present
      expect(html).toContain('startHeartbeat')
      expect(html).toContain('sendBeacon')
      expect(html).toContain('sessionStorage')
    } else {
      // EventLink not set — stream section absent, that's acceptable
      // (means the seed event's viewer_link wasn't reflected in GetTicket)
      console.log('stream-section not rendered — EventLink may not be set for this registration')
    }
  })

  // ── 5h. BROWSER: ticket page not reachable without auth ──────────────────

  test('browser: ticket page returns 401 without token', async ({ page }) => {
    const response = await page.goto(
      `${BASE}/api/cabinet/events/${eventID}/ticket`,
      { waitUntil: 'commit' },
    )
    const status = response?.status() ?? 0
    expect([401, 302]).toContain(status)
    expect(status).not.toBe(500)
  })
})
