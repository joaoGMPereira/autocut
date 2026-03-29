#!/usr/bin/env node
// Cross-platform port killer
import { execSync } from 'child_process';

const ports = process.argv.slice(2).map(Number).filter(Boolean);

for (const port of ports) {
  try {
    if (process.platform === 'win32') {
      const result = execSync(`netstat -ano | findstr :${port}`, { encoding: 'utf8' }).trim();
      const pids = [...new Set(result.split('\n').map(l => l.trim().split(/\s+/).pop()).filter(Boolean))];
      for (const pid of pids) execSync(`taskkill /F /PID ${pid}`, { stdio: 'ignore' });
    } else {
      execSync(`lsof -ti:${port} | xargs kill -9 2>/dev/null || true`, { shell: true });
    }
    console.log(`Killed port ${port}`);
  } catch {}
}
