import path from 'path';
import type { NextConfig } from 'next';

const output = process.env.NEXT_OUTPUT === 'standalone' ? 'standalone' : undefined;

const nextConfig: NextConfig = {
  ...(output ? { output } : {}),
  // CRÍTICO: fixar raiz de tracing no monorepo — garante que o tracer resolva
  // symlinks pnpm (apps/web/node_modules/react → ../../../.pnpm/...) e inclua
  // react/react-dom no standalone output automaticamente.
  outputFileTracingRoot: path.resolve(__dirname, '../..'),
  transpilePackages: ['@autocut/shared'],
  images: { unoptimized: true },
  devIndicators: false,
  allowedDevOrigins: ['127.0.0.1', 'localhost'],
};

export default nextConfig;
