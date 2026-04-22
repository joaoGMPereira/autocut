'use client';

import { useEffect, useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { FormField } from '@/components/ui/form-field';
import { Input } from '@/components/ui/input';
import { Calendar, CheckCircle2, XCircle, Tag } from 'lucide-react';
import type { QueueItem } from '@/store/queueStore';
import { useChannelConfigStore, type ChannelConfig } from '@/store/channelConfigStore';

function splitTags(raw?: string): string[] {
  if (!raw) return [];
  return raw.split(',').map((t) => t.trim()).filter(Boolean);
}

// ── YouTube category ID → name ────────────────────────────────────────────────
const YT_CATEGORIES: Record<number, string> = {
  1: 'Film & Animation',
  2: 'Autos & Vehicles',
  10: 'Music',
  15: 'Pets & Animals',
  17: 'Sports',
  19: 'Travel & Events',
  20: 'Gaming',
  22: 'People & Blogs',
  23: 'Comedy',
  24: 'Entertainment',
  25: 'News & Politics',
  26: 'Howto & Style',
  27: 'Education',
  28: 'Science & Technology',
  29: 'Nonprofits & Activism',
};

function categoryName(id: number): string {
  return YT_CATEGORIES[id] ?? `Category ${id}`;
}

// ── Config chips ──────────────────────────────────────────────────────────────
function Chip({
  label,
  active,
  value,
}: {
  label: string;
  active?: boolean;
  value?: string;
}) {
  return (
    <Badge variant={active ? 'success' : 'secondary'}>
      {active ? <CheckCircle2 className="size-2.5" /> : <XCircle className="size-2.5" />}
      {value ?? label}
    </Badge>
  );
}

// ── Channel config summary ────────────────────────────────────────────────────
function ChannelConfigSummary({
  config,
  videoType,
}: {
  config: ChannelConfig | undefined;
  videoType: string;
}) {
  if (!config) {
    return <p className="text-xs text-subtle italic">Loading channel config…</p>;
  }

  const playlist =
    videoType === 'short'
      ? config.default_playlist_id_shorts
      : config.default_playlist_id_cortes;

  return (
    <div className="flex flex-wrap gap-1.5 mt-1.5">
      <Chip label="Made for Kids" active={config.made_for_kids} />
      <Badge variant="secondary">
        {categoryName(config.default_category_id)}
      </Badge>
      {playlist ? (
        <span className="inline-flex items-center rounded-full border border-violet-500/30 bg-violet-500/10 px-2 py-0.5 text-[11px] text-violet-400">
          Playlist: {playlist}
        </span>
      ) : (
        <Badge variant="secondary" className="italic">
          No playlist
        </Badge>
      )}
      {config.speed_enabled && (
        <Chip label="Speed" active value={`${config.speed_factor ?? 1}x`} />
      )}
      {config.audio_music_enabled && <Chip label="Music" active />}
      {config.branding_logo_enabled && <Chip label="Logo" active />}
      {config.branding_intro_enabled && <Chip label="Intro" active />}
      {config.branding_outro_enabled && <Chip label="Outro" active />}
      {config.preview_captions_enabled && (
        <Chip label="Captions" active value={`Captions: ${config.preview_caption_style ?? 'on'}`} />
      )}
      {config.anti_duplicate_enabled && <Chip label="Anti-dup" active />}
      {config.visual_color_grading_enabled && <Chip label="Color grade" active />}
    </div>
  );
}

// ── Single item row ───────────────────────────────────────────────────────────
function ReviewItemRow({
  item,
  scheduledAt,
  channelName,
  config,
  goUrl,
}: {
  item: QueueItem;
  scheduledAt: string;
  channelName?: string;
  config?: ChannelConfig;
  goUrl: string;
}) {
  const tags = splitTags(item.tags);

  return (
    <div className="rounded-xl border border-border bg-card/50 p-3 space-y-3">
      {/* top: thumbnail + header */}
      <div className="flex gap-3">
        {/* thumbnail — fixed 16:9 dimensions to avoid flex stretching */}
        <div
          className="shrink-0 rounded-lg overflow-hidden bg-muted"
          style={{ width: 96, height: 54 }}
        >
          <img
            src={`${goUrl}/api/queue/${item.id}/thumbnail`}
            alt=""
            className="w-full h-full object-cover"
            onError={(e) => {
              (e.currentTarget as HTMLImageElement).style.display = 'none';
            }}
          />
        </div>

        {/* info */}
        <div className="flex-1 min-w-0 space-y-1">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-[11px] font-mono text-subtle">{item.video_type}</span>
            {channelName && (
              <Badge variant="outline" className="text-[10px]">
                {channelName}
              </Badge>
            )}
          </div>

          {/* title from pipeline_clips */}
          <p className="text-sm font-semibold text-heading leading-tight">
            {item.title || 'Untitled'}
          </p>

          {/* scheduled time */}
          <div className="flex items-center gap-1 text-[11px] text-violet-400">
            <Calendar className="size-3" />
            {new Date(scheduledAt).toLocaleString()}
          </div>
        </div>
      </div>

      {/* description from pipeline_clips */}
      {item.description && (
        <p className="text-[11px] text-caption line-clamp-3 leading-relaxed border-l-2 border-border pl-2">
          {item.description}
        </p>
      )}

      {/* tags from pipeline_clips */}
      {tags.length > 0 && (
        <div className="flex items-center gap-1.5 flex-wrap">
          <Tag className="size-3 text-subtle shrink-0" />
          {tags.slice(0, 8).map((tag) => (
            <Badge key={tag} variant="secondary" className="text-[10px]">
              {tag}
            </Badge>
          ))}
          {tags.length > 8 && (
            <span className="text-[10px] text-subtle">+{tags.length - 8} more</span>
          )}
        </div>
      )}

      {/* channel config chips */}
      <ChannelConfigSummary config={config} videoType={item.video_type} />
    </div>
  );
}

// ── Individual mode ───────────────────────────────────────────────────────────
interface IndividualProps {
  item: QueueItem;
  channelName?: string;
  goUrl: string;
  onConfirm: (publishAt: string) => Promise<void>;
  onCancel: () => void;
}

function IndividualReview({ item, channelName, goUrl, onConfirm, onCancel }: IndividualProps) {
  const { configs, fetchConfig } = useChannelConfigStore();
  const [scheduleDraft, setScheduleDraft] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!configs[item.channel_id]) void fetchConfig(item.channel_id);
  }, [item.channel_id]); // eslint-disable-line react-hooks/exhaustive-deps

  const config = configs[item.channel_id];

  const handleConfirm = async () => {
    if (!scheduleDraft) return;
    setLoading(true);
    try {
      await onConfirm(new Date(scheduleDraft).toISOString());
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <DialogHeader>
        <DialogTitle>Review Schedule</DialogTitle>
      </DialogHeader>

      <div className="space-y-4 py-2">
        {/* datetime picker */}
        <FormField label="Publish at">
          <Input
            type="datetime-local"
            value={scheduleDraft}
            onChange={(e) => setScheduleDraft(e.target.value)}
          />
        </FormField>

        {/* review card */}
        <div>
          <p className="text-xs text-subtle font-medium mb-2 uppercase tracking-wider">
            Video &amp; Config
          </p>
          <ReviewItemRow
            item={item}
            scheduledAt={
              scheduleDraft ? new Date(scheduleDraft).toISOString() : new Date().toISOString()
            }
            channelName={channelName}
            config={config}
            goUrl={goUrl}
          />
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" onClick={onCancel} disabled={loading}>
          Cancel
        </Button>
        <Button onClick={() => void handleConfirm()} disabled={!scheduleDraft || loading}>
          {loading ? 'Scheduling…' : 'Confirm Schedule'}
        </Button>
      </DialogFooter>
    </>
  );
}

// ── Bulk mode ─────────────────────────────────────────────────────────────────
interface BulkProps {
  items: QueueItem[];
  startAt: string;
  intervalMinutes: number;
  channels: Array<{ id: number; name: string }>;
  goUrl: string;
  onConfirm: () => Promise<void>;
  onCancel: () => void;
}

function BulkReview({
  items,
  startAt,
  intervalMinutes,
  channels,
  goUrl,
  onConfirm,
  onCancel,
}: BulkProps) {
  const { configs, fetchConfig } = useChannelConfigStore();
  const [loading, setLoading] = useState(false);

  // queued items only, in queue_order
  const queued = items.filter((i) => i.status === 'queued');

  // fetch configs for all unique channel IDs
  useEffect(() => {
    const ids = [...new Set(queued.map((i) => i.channel_id))];
    ids.forEach((id) => {
      if (!configs[id]) void fetchConfig(id);
    });
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const baseTime = new Date(startAt).getTime();

  const handleConfirm = async () => {
    setLoading(true);
    try {
      await onConfirm();
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <DialogHeader>
        <DialogTitle>Review Bulk Schedule — {queued.length} videos</DialogTitle>
      </DialogHeader>

      <div className="py-2 space-y-2 max-h-[60vh] overflow-y-auto pr-1">
        {queued.length === 0 ? (
          <p className="text-sm text-subtle text-center py-8">No queued items to schedule.</p>
        ) : (
          queued.map((item, i) => {
            const scheduledAt = new Date(baseTime + i * intervalMinutes * 60 * 1000).toISOString();
            const channelName = channels.find((c) => c.id === item.channel_id)?.name;
            const config = configs[item.channel_id];
            return (
              <ReviewItemRow
                key={item.id}
                item={item}
                scheduledAt={scheduledAt}
                channelName={channelName}
                config={config}
                goUrl={goUrl}
              />
            );
          })
        )}
      </div>

      <DialogFooter>
        <Button variant="outline" onClick={onCancel} disabled={loading}>
          Cancel
        </Button>
        <Button
          onClick={() => void handleConfirm()}
          disabled={queued.length === 0 || loading}
        >
          {loading ? 'Scheduling…' : `Schedule ${queued.length} videos`}
        </Button>
      </DialogFooter>
    </>
  );
}

// ── Public component ──────────────────────────────────────────────────────────

interface ReviewScheduleModalProps {
  open: boolean;
  mode: 'individual' | 'bulk';
  // individual
  item?: QueueItem;
  channelName?: string;
  // bulk
  items?: QueueItem[];
  startAt?: string;
  intervalMinutes?: number;
  channels?: Array<{ id: number; name: string }>;
  // common
  goUrl: string;
  onConfirm: (publishAt?: string) => Promise<void>;
  onCancel: () => void;
}

export function ReviewScheduleModal({
  open,
  mode,
  item,
  channelName,
  items,
  startAt,
  intervalMinutes,
  channels,
  goUrl,
  onConfirm,
  onCancel,
}: ReviewScheduleModalProps) {
  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onCancel(); }}>
      <DialogContent className="max-w-2xl w-full">
        {mode === 'individual' && item ? (
          <IndividualReview
            item={item}
            channelName={channelName}
            goUrl={goUrl}
            onConfirm={async (publishAt) => onConfirm(publishAt)}
            onCancel={onCancel}
          />
        ) : mode === 'bulk' && items && startAt ? (
          <BulkReview
            items={items}
            startAt={startAt}
            intervalMinutes={intervalMinutes ?? 1440}
            channels={channels ?? []}
            goUrl={goUrl}
            onConfirm={async () => onConfirm()}
            onCancel={onCancel}
          />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
