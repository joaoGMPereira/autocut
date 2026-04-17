import { test, expect } from '@playwright/test';
import { getSetting } from '../helpers/db';
import { getSettings, putSetting } from '../helpers/api';

const MODE_KEY = 'workflow_mode_default';

test.describe('Workflow mode selection (spec 104)', () => {
  test('AI mode persists in app_settings', async () => {
    await putSetting(MODE_KEY, 'ai');
    expect(getSetting(MODE_KEY)).toBe('ai');
    const settings = await getSettings();
    expect(settings[MODE_KEY]).toBe('ai');
  });

  test('Longform mode persists in app_settings', async () => {
    await putSetting(MODE_KEY, 'longform');
    expect(getSetting(MODE_KEY)).toBe('longform');
    const settings = await getSettings();
    expect(settings[MODE_KEY]).toBe('longform');
  });

  test('mode switching overwrites previous value', async () => {
    await putSetting(MODE_KEY, 'ai');
    expect(getSetting(MODE_KEY)).toBe('ai');
    await putSetting(MODE_KEY, 'longform');
    expect(getSetting(MODE_KEY)).toBe('longform');
  });
});
