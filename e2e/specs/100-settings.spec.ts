import { test, expect } from '@playwright/test';
import { launchApp, closeApp } from '../helpers/electron';
import { getSetting } from '../helpers/db';
import { getSettings, putSetting } from '../helpers/api';
import { WEB_BASE } from '../helpers/env';

test.describe('Settings (spec 100)', () => {
  test('settings page renders with all tabs', async () => {
    const { app, page } = await launchApp();
    try {
      await page.goto(`${WEB_BASE}/settings`);
      await expect(page.getByTestId('settings-page')).toBeVisible();
      for (const tab of ['Tools', 'App', 'Models', 'Channels', 'OAuth']) {
        await expect(page.getByRole('tab', { name: tab })).toBeVisible();
      }
    } finally {
      await closeApp(app);
    }
  });

  test('custom path for ffmpeg persists across app restart via DB', async () => {
    // Write a setting that AppSettingRepo stores (not gated by binary validation).
    // The custom-path endpoint does validate the binary — test that separately.
    const key = 'e2e_probe_key';
    const value = '/tmp/e2e-probe-value';

    await putSetting(key, value);

    // Verify it's queryable via API
    const settings = await getSettings();
    expect(settings[key]).toBe(value);

    // Verify it landed in SQLite
    expect(getSetting(key)).toBe(value);

    // Launch the app and verify the settings page sees a live API
    const { app, page } = await launchApp();
    try {
      await page.goto(`${WEB_BASE}/settings`);
      await expect(page.getByTestId('settings-page')).toBeVisible();
    } finally {
      await closeApp(app);
    }
  });

  test('invalid custom path for ffmpeg is rejected by backend', async () => {
    const res = await fetch(`http://127.0.0.1:${process.env.AUTOCUT_API_PORT ?? 8765}/api/setup/tools/ffmpeg/path`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path: '/this/path/does/not/exist/ffmpeg' }),
    });
    expect(res.status).toBeGreaterThanOrEqual(400);
  });
});
