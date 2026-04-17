import { test, expect } from '@playwright/test';
import { launchApp, closeApp } from '../helpers/electron';
import { WEB_BASE } from '../helpers/env';

test.describe('Smoke: app renders without setup gate (bypass on)', () => {
  test('A1: / loads and pipeline is reachable', async () => {
    const { app, page } = await launchApp();
    try {
      await page.goto(`${WEB_BASE}/pipeline`);
      await expect(page.getByTestId('active-step')).toBeVisible({ timeout: 30_000 });
      await expect(page.getByTestId('step-url-input')).toBeVisible();
    } finally {
      await closeApp(app);
    }
  });

  test('A2: ?setup=force renders the real SetupGate', async () => {
    const { app, page } = await launchApp();
    try {
      await page.goto(`${WEB_BASE}/pipeline?setup=force`);
      await expect(page.getByTestId('setup-gate')).toBeVisible({ timeout: 30_000 });
      // At least one required tool row visible (tools list populated)
      const toolRows = page.locator('[data-testid^="tool-row-"]');
      await expect(toolRows.first()).toBeVisible({ timeout: 15_000 });
    } finally {
      await closeApp(app);
    }
  });
});
