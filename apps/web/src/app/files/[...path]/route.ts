import { NextRequest, NextResponse } from 'next/server';
import fs from 'fs';
import path from 'path';

const MIME: Record<string, string> = {
  '.jpg': 'image/jpeg',
  '.jpeg': 'image/jpeg',
  '.png': 'image/png',
  '.webp': 'image/webp',
  '.gif': 'image/gif',
  '.mp4': 'video/mp4',
  '.webm': 'video/webm',
};

export async function GET(
  _req: NextRequest,
  { params }: { params: Promise<{ path: string[] }> }
) {
  const { path: segments } = await params;
  const filePath = decodeURIComponent(segments.join('/'));

  const allowed = process.env.DATA_DIR ?? path.join(process.env.HOME ?? '', '.autocut-dev');
  if (!filePath.startsWith(allowed)) {
    return new NextResponse('Forbidden', { status: 403 });
  }

  try {
    const buf = fs.readFileSync(filePath);
    const mime = MIME[path.extname(filePath).toLowerCase()] ?? 'application/octet-stream';
    return new NextResponse(buf, {
      headers: { 'Content-Type': mime },
    });
  } catch {
    return new NextResponse('Not found', { status: 404 });
  }
}
