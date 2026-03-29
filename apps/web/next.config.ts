import type { NextConfig } from 'next';

const output = process.env.NEXT_OUTPUT === 'standalone' ? 'standalone' : undefined;

const nextConfig: NextConfig = {
  ...(output ? { output } : {}),
  transpilePackages: ['@autocut/shared'],
  images: { unoptimized: true },
  devIndicators: false,
  allowedDevOrigins: ['127.0.0.1', 'localhost'],
};

export default nextConfig;
