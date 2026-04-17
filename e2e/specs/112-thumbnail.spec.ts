import { test, expect } from '@playwright/test';
import { putSetting } from '../helpers/api';
import { getSetting } from '../helpers/db';

test.describe('Thumbnail strategy (spec 112)', () => {
  test('thumbnail template persists in app_settings', async () => {
    const key = 'thumbnail_template_default';
    const value = JSON.stringify({ style: 'bold-text', font: 'Impact' });
    await putSetting(key, value);
    expect(getSetting(key)).toBe(value);
  });

});
