'use client';

import { Anton } from 'next/font/google';
import { useEffect, useState } from 'react';
import { createLogger } from '@/lib/logger';
import { useAppStore } from '@/store/appStore';
import { useChannelStore } from '@/store/channelStore';
import { useMassUpdateStore } from '@/store/massUpdateStore';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Progress } from '@/components/ui/progress';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';

const anton = Anton({ weight: '400', subsets: ['latin'], display: 'swap' });
const log = createLogger('MassUpdatePage');

// ─── Channel Selector ─────────────────────────────────────────────────────────

function ChannelSelector() {
  const { goUrl } = useAppStore();
  const { channels, fetchChannels } = useChannelStore();
  const { channelId, setChannelId, fetchVideos } = useMassUpdateStore();

  useEffect(() => {
    void fetchChannels(goUrl);
  }, [goUrl, fetchChannels]);

  const handleChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const id = parseInt(e.target.value, 10);
    if (!isNaN(id)) {
      setChannelId(id);
      void fetchVideos(goUrl, 1);
    }
  };

  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor="channel-select">Channel</Label>
      <select
        id="channel-select"
        value={channelId ?? ''}
        onChange={handleChange}
        className="w-full max-w-xs rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
      >
        <option value="">Select a channel…</option>
        {channels.map((ch) => (
          <option key={ch.ID} value={ch.ID}>
            {ch.Name}
          </option>
        ))}
      </select>
    </div>
  );
}

// ─── Filter Bar ───────────────────────────────────────────────────────────────

function FilterBar() {
  const { goUrl } = useAppStore();
  const { filter, setFilter, fetchVideos, channelId } = useMassUpdateStore();

  const handleApply = () => {
    if (channelId === null) return;
    void fetchVideos(goUrl, 1);
  };

  return (
    <div className="rounded-xl bg-card border border-border p-5 flex flex-col gap-4">
      <h2 className="text-sm font-semibold text-[#F0F0F8]">Filters</h2>
      <div className="grid grid-cols-3 gap-4">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="date-from">Date From</Label>
          <Input
            id="date-from"
            type="date"
            value={filter.date_from ?? ''}
            onChange={(e) => setFilter({ date_from: e.target.value || undefined })}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="date-to">Date To</Label>
          <Input
            id="date-to"
            type="date"
            value={filter.date_to ?? ''}
            onChange={(e) => setFilter({ date_to: e.target.value || undefined })}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="category-filter">Category ID</Label>
          <Input
            id="category-filter"
            type="number"
            placeholder="e.g. 22"
            value={filter.category_id ?? ''}
            onChange={(e) => {
              const val = e.target.value;
              setFilter({ category_id: val === '' ? undefined : parseInt(val, 10) });
            }}
          />
        </div>
      </div>
      <Button variant="outline" size="sm" className="w-fit" onClick={handleApply}>
        Apply Filters
      </Button>
    </div>
  );
}

// ─── Video List ───────────────────────────────────────────────────────────────

function VideoList() {
  const { goUrl } = useAppStore();
  const { videos, totalVideos, currentPage, status, channelId, fetchVideos } = useMassUpdateStore();

  if (!channelId) return null;
  if (status === 'loading' && videos.length === 0) {
    return (
      <div className="rounded-xl bg-card border border-border p-5 text-center text-sm text-[#5C5C80]">
        Loading videos…
      </div>
    );
  }
  if (videos.length === 0) return null;

  const totalPages = Math.ceil(totalVideos / 20);

  return (
    <div className="rounded-xl bg-card border border-border overflow-hidden">
      <div className="px-5 py-3 border-b border-border flex items-center justify-between">
        <span className="text-sm font-medium">
          {totalVideos} video{totalVideos !== 1 ? 's' : ''}
        </span>
        <div className="flex items-center gap-2 text-xs text-[#5C5C80]">
          <span>Page {currentPage} / {totalPages}</span>
          <Button
            variant="outline"
            size="sm"
            disabled={currentPage <= 1}
            onClick={() => void fetchVideos(goUrl, currentPage - 1)}
          >
            Prev
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={currentPage >= totalPages}
            onClick={() => void fetchVideos(goUrl, currentPage + 1)}
          >
            Next
          </Button>
        </div>
      </div>
      <div className="divide-y divide-border max-h-64 overflow-y-auto">
        {videos.map((v) => (
          <div key={v.id} className="flex items-center gap-3 px-5 py-2.5">
            <span className="text-xs text-[#5C5C80] font-mono shrink-0">{v.youtube_id}</span>
            <span className="text-sm truncate">{v.title}</span>
            <span className="text-xs text-[#5C5C80] shrink-0 ml-auto">
              {v.published_at ? new Date(v.published_at).toLocaleDateString() : '—'}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

// ─── Update Form ──────────────────────────────────────────────────────────────

function UpdateForm() {
  const { updates, setUpdates } = useMassUpdateStore();

  return (
    <div className="rounded-xl bg-card border border-border p-5 flex flex-col gap-5">
      <h2 className="text-sm font-semibold text-[#F0F0F8]">Update Rules</h2>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="title-template">Title Template</Label>
        <Input
          id="title-template"
          placeholder="e.g. {original} | {channel} — {date}"
          value={updates.title_template ?? ''}
          onChange={(e) => setUpdates({ title_template: e.target.value || undefined })}
        />
        <p className="text-xs text-[#5C5C80]">
          Placeholders: <code className="font-mono">{'{original}'}</code>,{' '}
          <code className="font-mono">{'{channel}'}</code>,{' '}
          <code className="font-mono">{'{date}'}</code>
        </p>
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="description-append">Append to Description</Label>
        <textarea
          id="description-append"
          rows={3}
          placeholder="Text to append at the end of each description…"
          value={updates.description_append ?? ''}
          onChange={(e) => setUpdates({ description_append: e.target.value || undefined })}
          className="w-full resize-none rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-xs outline-none transition-[color,box-shadow] placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-input/30"
        />
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="tags-add">Tags to Add</Label>
          <Input
            id="tags-add"
            placeholder="gaming, highlights"
            value={updates.tags_add?.join(', ') ?? ''}
            onChange={(e) => {
              const val = e.target.value;
              setUpdates({ tags_add: val ? val.split(',').map((t) => t.trim()).filter(Boolean) : undefined });
            }}
          />
          <p className="text-xs text-[#5C5C80]">Comma-separated</p>
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="tags-remove">Tags to Remove</Label>
          <Input
            id="tags-remove"
            placeholder="oldtag, spam"
            value={updates.tags_remove?.join(', ') ?? ''}
            onChange={(e) => {
              const val = e.target.value;
              setUpdates({ tags_remove: val ? val.split(',').map((t) => t.trim()).filter(Boolean) : undefined });
            }}
          />
          <p className="text-xs text-[#5C5C80]">Comma-separated</p>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="hashtags">Hashtags</Label>
          <Input
            id="hashtags"
            placeholder="#gaming, #clips"
            value={updates.hashtags?.join(', ') ?? ''}
            onChange={(e) => {
              const val = e.target.value;
              setUpdates({ hashtags: val ? val.split(',').map((t) => t.trim()).filter(Boolean) : undefined });
            }}
          />
          <p className="text-xs text-[#5C5C80]">Comma-separated</p>
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="category-update">Category ID</Label>
          <Input
            id="category-update"
            type="number"
            placeholder="Leave blank to keep existing"
            value={updates.category_id ?? ''}
            onChange={(e) => {
              const val = e.target.value;
              setUpdates({ category_id: val === '' ? undefined : parseInt(val, 10) });
            }}
          />
        </div>
      </div>
    </div>
  );
}

// ─── Dry Run Preview ──────────────────────────────────────────────────────────

function DryRunPreview() {
  const { dryRunPreview, status } = useMassUpdateStore();
  if (status !== 'dry_run_done' || !dryRunPreview) return null;

  return (
    <div className="rounded-xl bg-card border border-border overflow-hidden">
      <div className="px-5 py-3 border-b border-border flex items-center gap-3">
        <h2 className="text-sm font-semibold">Dry Run Preview</h2>
        <span className="bg-[#00D4FF]/10 text-[#00D4FF] text-xs px-2 py-0.5 rounded-full font-medium">
          {dryRunPreview.length} affected
        </span>
      </div>
      <div className="overflow-x-auto max-h-72 overflow-y-auto">
        <table className="w-full text-sm">
          <thead className="sticky top-0 bg-card border-b border-border">
            <tr>
              <th className="text-left px-5 py-2 text-xs font-medium text-[#5C5C80] w-32">YouTube ID</th>
              <th className="text-left px-5 py-2 text-xs font-medium text-[#5C5C80]">Current Title</th>
              <th className="text-left px-5 py-2 text-xs font-medium text-[#5C5C80]">New Title</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {dryRunPreview.map((item) => (
              <tr key={item.youtube_id} className="hover:bg-[#1A1A26]/50">
                <td className="px-5 py-2 font-mono text-xs text-[#5C5C80] truncate max-w-[7rem]">
                  {item.youtube_id}
                </td>
                <td className="px-5 py-2 text-[#5C5C80] truncate max-w-[16rem]">
                  {item.old_title}
                </td>
                <td className="px-5 py-2 truncate max-w-[16rem]">
                  {item.new_title}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// ─── Progress Panel ───────────────────────────────────────────────────────────

function ProgressPanel() {
  const { goUrl } = useAppStore();
  const { status, progress, jobId, subscribeProgress } = useMassUpdateStore();

  useEffect(() => {
    if (status === 'running' && jobId) {
      const unsub = subscribeProgress(goUrl);
      return unsub;
    }
  }, [status, jobId, goUrl, subscribeProgress]);

  if (status !== 'running' && status !== 'done') return null;

  const pct = progress && progress.total > 0
    ? Math.round((progress.done / progress.total) * 100)
    : 0;

  return (
    <div className="rounded-xl bg-card border border-border p-5 flex flex-col gap-3">
      {status === 'running' ? (
        <>
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">Updating videos…</span>
            <span className="text-sm text-[#5C5C80]">
              {progress ? `${progress.done} / ${progress.total}` : '—'}
            </span>
          </div>
          <Progress value={pct} />
        </>
      ) : (
        <div className="flex items-center gap-2">
          <span className="h-2 w-2 rounded-full bg-green-500 shrink-0" />
          <span className="text-sm font-medium">
            Done — updated {progress?.done ?? 0} video{progress?.done !== 1 ? 's' : ''}
            {progress && progress.total > progress.done
              ? `, ${progress.total - progress.done} failed`
              : ''}
          </span>
        </div>
      )}
    </div>
  );
}

// ─── Action Buttons ───────────────────────────────────────────────────────────

function ActionButtons() {
  const { goUrl } = useAppStore();
  const { channelId, status, runDryRun, runUpdate } = useMassUpdateStore();
  const [confirmOpen, setConfirmOpen] = useState(false);

  const disabled = channelId === null || status === 'loading' || status === 'running';

  const handleDryRun = () => {
    log.info('dry run triggered');
    void runDryRun(goUrl);
  };

  const handleConfirmApply = () => {
    setConfirmOpen(false);
    log.info('mass update confirmed');
    void runUpdate(goUrl);
  };

  return (
    <>
      <div className="flex items-center gap-3">
        <Button
          variant="outline"
          onClick={handleDryRun}
          disabled={disabled}
        >
          Preview (Dry Run)
        </Button>
        <Button
          onClick={() => setConfirmOpen(true)}
          disabled={disabled}
        >
          Apply to All Videos
        </Button>
      </div>

      <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Confirm Mass Update</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-[#5C5C80]">
            This will update all matching videos on YouTube. This action cannot be undone.
            Run a Dry Run first to preview changes.
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirmOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleConfirmApply}>
              Apply Updates
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

// ─── Error Banner ─────────────────────────────────────────────────────────────

function ErrorBanner() {
  const { status, error } = useMassUpdateStore();
  if (status !== 'error' || !error) return null;
  return (
    <div className="rounded-xl bg-destructive/10 border border-destructive/20 p-4">
      <p className="text-sm text-destructive">{error}</p>
    </div>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function MassUpdatePage() {
  return (
    <div className="px-10 py-8 flex flex-col gap-7">
      <h1 className={`${anton.className} text-[28px] tracking-[1.5px] text-[#F0F0F8]`}>
        MASS UPDATE
      </h1>

      <ChannelSelector />
      <FilterBar />
      <VideoList />
      <UpdateForm />
      <ActionButtons />
      <DryRunPreview />
      <ProgressPanel />
      <ErrorBanner />
    </div>
  );
}
