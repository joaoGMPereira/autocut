'use client';

import { Anton } from 'next/font/google';
import { useEffect, useState } from 'react';
import { useAppStore } from '@/store/appStore';
import { useUploadStore } from '@/store/uploadStore';
import { useChannelStore } from '@/store/channelStore';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Button } from '@/components/ui/button';

const anton = Anton({ weight: '400', subsets: ['latin'], display: 'swap' });

function StatusBadge({ status }: { status: string }) {
  switch (status) {
    case 'active':
    case 'running':
      return (
        <span className="bg-blue-500/10 text-blue-400 text-xs px-2 py-0.5 rounded-md">
          {status}
        </span>
      );
    case 'done':
      return (
        <span className="bg-emerald-500/10 text-emerald-400 text-xs px-2 py-0.5 rounded-md">
          done
        </span>
      );
    case 'error':
      return (
        <span className="bg-destructive/10 text-destructive text-xs px-2 py-0.5 rounded-md">
          error
        </span>
      );
    default:
      return (
        <span className="bg-[#1A1A26] text-[#5C5C80] text-xs px-2 py-0.5 rounded-md">
          {status || 'pending'}
        </span>
      );
  }
}

export default function QueuePage() {
  const { goUrl } = useAppStore();
  const { uploads, uploadJobs, activeJobId, fetchUploads, startUpload } = useUploadStore();
  const { channels, fetchChannels } = useChannelStore();

  const [videoPath, setVideoPath] = useState('');
  const [channelId, setChannelId] = useState<number | ''>('');
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    void fetchUploads(goUrl);
    void fetchChannels(goUrl);
  }, [goUrl, fetchUploads, fetchChannels]);

  const handleUpload = async () => {
    if (!channelId || !videoPath.trim()) return;
    setSubmitting(true);
    setUploadError(null);
    try {
      await startUpload(goUrl, {
        video_path: videoPath.trim(),
        channel_id: Number(channelId),
        title: title.trim(),
        description: description.trim(),
      });
    } catch (err) {
      setUploadError(err instanceof Error ? err.message : 'Upload failed');
    } finally {
      setSubmitting(false);
    }
  };

  const activeJob = activeJobId ? uploadJobs[activeJobId] : null;

  return (
    <div className="px-10 py-8 flex flex-col gap-7">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className={`${anton.className} text-[28px] tracking-[1.5px] text-[#F0F0F8]`}>
          QUEUE
        </h1>
        <Button variant="outline" size="sm" onClick={() => void fetchUploads(goUrl)}>
          Refresh
        </Button>
      </div>

      {/* Upload form */}
      <div className="rounded-xl bg-card border border-border p-5 flex flex-col gap-4">
        <h2 className="text-sm font-semibold">New Upload</h2>

        <div className="grid grid-cols-2 gap-4">
          <div className="col-span-2">
            <label className="text-xs text-[#5C5C80] mb-1 block">Video Path</label>
            <input
              type="text"
              value={videoPath}
              onChange={(e) => setVideoPath(e.target.value)}
              placeholder="/path/to/video.mp4"
              className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
            />
          </div>

          <div>
            <label className="text-xs text-[#5C5C80] mb-1 block">Channel</label>
            <select
              value={channelId}
              onChange={(e) =>
                setChannelId(e.target.value === '' ? '' : Number(e.target.value))
              }
              className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60 appearance-none"
            >
              <option value="">Select a channel…</option>
              {channels.map((ch) => (
                <option key={ch.ID} value={ch.ID}>
                  {ch.Name}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="text-xs text-[#5C5C80] mb-1 block">Title</label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Video title"
              className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
            />
          </div>

          <div className="col-span-2">
            <label className="text-xs text-[#5C5C80] mb-1 block">Description</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Video description"
              rows={3}
              className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm min-h-[80px] resize-none focus:outline-none focus:border-[#00D4FF]/60"
            />
          </div>
        </div>

        {uploadError && <p className="text-xs text-destructive">{uploadError}</p>}

        <Button
          onClick={() => void handleUpload()}
          disabled={submitting || !videoPath.trim() || !channelId}
          className="w-fit"
        >
          {submitting ? 'Uploading…' : 'Upload'}
        </Button>
      </div>

      {/* Active job SSE log viewer */}
      {activeJob && (
        <div className="rounded-xl bg-card border border-border p-5 flex flex-col gap-3">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold">Upload Progress</h2>
            <StatusBadge status={activeJob.status} />
          </div>
          <ScrollArea className="h-40 rounded-lg bg-[#0D0D14] border border-border px-3 py-2">
            <div className="font-mono text-xs space-y-0.5">
              {activeJob.logs.length === 0 ? (
                <span className="text-[#5C5C80]">Waiting for logs…</span>
              ) : (
                activeJob.logs.map((line, i) => (
                  <div key={i} className="text-[#A0A0C0] leading-5">
                    {line}
                  </div>
                ))
              )}
            </div>
          </ScrollArea>
        </div>
      )}

      {/* Upload history list */}
      <div className="flex flex-col gap-3">
        <h2 className="text-[18px] font-semibold">History</h2>
        {uploads.length === 0 ? (
          <div className="rounded-xl bg-card border border-border p-5">
            <p className="text-[#5C5C80] text-sm text-center py-6">No uploads in queue</p>
          </div>
        ) : (
          <div className="rounded-xl bg-card border border-border divide-y divide-border">
            {uploads.map((upload) => (
              <div
                key={upload.ID}
                className="flex items-center justify-between px-5 py-3 gap-4"
              >
                <span className="font-mono text-xs text-[#A0A0C0] truncate flex-1 min-w-0">
                  {upload.VideoPath}
                </span>
                <div className="flex items-center gap-3 shrink-0">
                  <StatusBadge status={upload.Status} />
                  <span className="text-xs text-[#5C5C80]">
                    {new Date(upload.CreatedAt).toLocaleDateString()}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
