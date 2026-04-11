import { DownloadTestClient } from '@/components/download-test/DownloadTestClient';

export const metadata = {
  title: 'Download Test — AutoCut',
};

export default function DownloadTestPage() {
  return (
    <div className="px-10 py-8 flex flex-col gap-7">
      <div>
        <h1 className="text-2xl font-semibold text-foreground">Download Test</h1>
        <p className="text-sm text-zinc-400 mt-1">
          Test the YouTube downloader — paste a URL to extract metadata and download.
        </p>
      </div>
      <DownloadTestClient />
    </div>
  );
}
