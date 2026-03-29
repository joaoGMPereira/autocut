#!/usr/bin/env node
import { readFileSync, writeFileSync } from 'fs';
import { execSync } from 'child_process';

const level = process.argv[2]; // patch | minor | major
if (!['patch', 'minor', 'major'].includes(level)) {
  console.error('Usage: node bump-version.mjs patch|minor|major');
  process.exit(1);
}

const files = [
  'apps/desktop/package.json',
  'apps/web/package.json',
  'packages/shared/package.json',
];

const root = new URL('..', import.meta.url).pathname;
const desktopPkg = JSON.parse(readFileSync(`${root}apps/desktop/package.json`, 'utf8'));
const [major, minor, patch] = desktopPkg.version.split('.').map(Number);

let nextVersion;
if (level === 'major') nextVersion = `${major + 1}.0.0`;
else if (level === 'minor') nextVersion = `${major}.${minor + 1}.0`;
else nextVersion = `${major}.${minor}.${patch + 1}`;

for (const file of files) {
  const pkg = JSON.parse(readFileSync(`${root}${file}`, 'utf8'));
  pkg.version = nextVersion;
  writeFileSync(`${root}${file}`, JSON.stringify(pkg, null, 2) + '\n');
}

console.log(`Bumped to ${nextVersion}`);
