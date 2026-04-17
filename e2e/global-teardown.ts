import fs from 'node:fs';
import { stopGoServer } from './helpers/server';
import { stopNextDev } from './helpers/next';
import { TEST_DATA_DIR } from './helpers/env';

async function globalTeardown(): Promise<void> {
  console.log('[e2e] global-teardown starting');
  stopGoServer();
  stopNextDev();
  // Give subprocesses a moment to exit cleanly before removing their DB files.
  await new Promise((r) => setTimeout(r, 500));
  try {
    if (process.env.AUTOCUT_KEEP_TEST_DIR !== 'true') {
      fs.rmSync(TEST_DATA_DIR, { recursive: true, force: true });
      console.log('[e2e] removed', TEST_DATA_DIR);
    } else {
      console.log('[e2e] keeping', TEST_DATA_DIR, '(AUTOCUT_KEEP_TEST_DIR=true)');
    }
  } catch (e) {
    console.warn('[e2e] teardown rm failed:', e);
  }
  console.log('[e2e] global-teardown complete');
}

export default globalTeardown;
