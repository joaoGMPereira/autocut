import os from 'node:os';
import path from 'node:path';

// A single shared test-run identity. `globalSetup` computes it, then exports
// TEST_DATA_DIR via process.env so every worker process resolves the same path.
const RUN_ID_FROM_ENV = process.env.AUTOCUT_TEST_RUN_ID;
export const TEST_RUN_ID = RUN_ID_FROM_ENV ?? `autocut-e2e-${Date.now()}-${process.pid}`;
if (!RUN_ID_FROM_ENV) {
  process.env.AUTOCUT_TEST_RUN_ID = TEST_RUN_ID;
}

export const TEST_API_PORT = Number(process.env.AUTOCUT_API_PORT ?? 8765);
export const TEST_WEB_PORT = Number(process.env.AUTOCUT_WEB_PORT ?? 3299);
export const TEST_DATA_DIR =
  process.env.AUTOCUT_TEST_DATA_DIR ?? path.join(os.tmpdir(), TEST_RUN_ID);
if (!process.env.AUTOCUT_TEST_DATA_DIR) {
  process.env.AUTOCUT_TEST_DATA_DIR = TEST_DATA_DIR;
}
export const TEST_DB_PATH = path.join(TEST_DATA_DIR, 'autocut.db');
export const TEST_SERVER_BIN = path.join(os.tmpdir(), 'autocut-test-server');
export const API_BASE = `http://127.0.0.1:${TEST_API_PORT}`;
export const WEB_BASE = `http://127.0.0.1:${TEST_WEB_PORT}`;
