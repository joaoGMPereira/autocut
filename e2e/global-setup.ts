import { compileGoServer, startGoServer } from './helpers/server';
import { startNextDev } from './helpers/next';
import { TEST_DATA_DIR, TEST_API_PORT, TEST_WEB_PORT } from './helpers/env';

async function globalSetup(): Promise<void> {
  console.log('[e2e] global-setup starting');
  console.log('[e2e] data dir:', TEST_DATA_DIR);
  console.log('[e2e] api port:', TEST_API_PORT, 'web port:', TEST_WEB_PORT);
  compileGoServer();
  await startGoServer();
  if (process.env.AUTOCUT_E2E_NO_NEXT === 'true') {
    console.log('[e2e] AUTOCUT_E2E_NO_NEXT=true — skipping Next.js dev spawn (UI tests will fail)');
  } else {
    await startNextDev();
  }
  console.log('[e2e] global-setup complete');
}

export default globalSetup;
