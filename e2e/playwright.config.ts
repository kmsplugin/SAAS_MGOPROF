import { defineConfig, devices } from '@playwright/test'

/**
 * Playwright smoke suite for mgoprof-saas.
 *
 * Target: the Go server-rendered application (mgoprof-saas).
 * Base URL is read from BASE_URL env var (default: http://localhost:8080).
 *
 * Local run:
 *   BASE_URL=http://localhost:8080 npx playwright test
 *
 * CI run (after server is started in CI job):
 *   see .github/workflows/ci.yml e2e job
 *
 * The tests use a "test OTP mode" controlled by TEST_OTP env var:
 *   - when TEST_OTP is set, the tests read the OTP from the test API endpoint
 *     (GET /api/test/last-otp?email=...) which is only available in test mode
 *   - in production this endpoint does not exist
 */
export default defineConfig({
  testDir: './tests',
  timeout: 30_000,
  retries: 1,
  reporter: [['list'], ['html', { open: 'never', outputFolder: 'playwright-report' }]],

  use: {
    baseURL: process.env.BASE_URL ?? 'http://localhost:8080',
    headless: true,
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
})
