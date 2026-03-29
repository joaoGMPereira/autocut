import { execSync } from 'child_process';
import * as esbuild from 'esbuild';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const root = path.join(__dirname, '..');

let gitSha = 'unknown';
try {
  gitSha = execSync('git rev-parse --short HEAD', { cwd: root }).toString().trim();
} catch {}

const buildTs = new Date().toISOString().slice(0, 16).replace('T', '-');

const shared = {
  bundle: true,
  platform: 'node',
  target: 'node20',
  format: 'cjs',
  external: ['electron'],
  define: {
    '__GIT_SHA__': JSON.stringify(gitSha),
    '__BUILD_TS__': JSON.stringify(buildTs),
  },
  sourcemap: false,
  minify: false,
  // Resolve @autocut/shared from monorepo
  alias: {
    '@autocut/shared': path.join(root, '../../packages/shared/src/index.ts'),
  },
};

await Promise.all([
  esbuild.build({
    ...shared,
    entryPoints: [path.join(root, 'src/main.ts')],
    outfile: path.join(root, 'compiled/main.js'),
  }),
  esbuild.build({
    ...shared,
    entryPoints: [path.join(root, 'src/preload.ts')],
    outfile: path.join(root, 'compiled/preload.js'),
  }),
]);

console.log(`Built main + preload (sha=${gitSha} ts=${buildTs})`);
