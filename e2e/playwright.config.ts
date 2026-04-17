import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './specs',
  // Full E2E with real subprocesses — generous timeouts.
  timeout: 120_000,
  expect: { timeout: 15_000 },
  // Serial: a single Go server + SQLite is shared across tests.
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [['list']],
  globalSetup: require.resolve('./global-setup'),
  globalTeardown: require.resolve('./global-teardown'),
  use: {
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
});
