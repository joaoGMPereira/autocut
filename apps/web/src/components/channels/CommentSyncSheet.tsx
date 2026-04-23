'use client';

import { useEffect, useRef, useState } from 'react';
import { createLogger } from '@/lib/logger';
import { useAppStore } from '@/store/appStore';
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Progress } from '@/components/ui/progress';
import type { Channel } from '@/store/channelStore';

const log = createLogger('CommentSyncSheet');

// ─── Types ────────────────────────────────────────────────────────────────────

interface Comment {
  id: number;
  comment_id: string;
  author: string;
  text: string;
  likes: number;
  published_at: string;
}

interface CommentsResponse {
  comments: Comment[];
  page: number;
  total: number;
}

interface SyncProgress {
  done: number;
  total: number;
}

// ─── Props ────────────────────────────────────────────────────────────────────

interface Props {
  channel: Channel;
  open: boolean;
  onClose: (open: boolean) => void;
}

// ─── Component ────────────────────────────────────────────────────────────────

export function CommentSyncSheet({ channel, open, onClose }: Props) {
  const { goUrl } = useAppStore();

  const [comments, setComments] = useState<Comment[]>([]);
  const [totalComments, setTotalComments] = useState(0);
  const [currentPage, setCurrentPage] = useState(1);
  const [search, setSearch] = useState('');
  const [loadingComments, setLoadingComments] = useState(false);
  const [commentsError, setCommentsError] = useState<string | null>(null);

  const [syncing, setSyncing] = useState(false);
  const [syncProgress, setSyncProgress] = useState<SyncProgress | null>(null);
  const [syncStatus, setSyncStatus] = useState<'idle' | 'running' | 'done' | 'error'>('idle');
  const [syncError, setSyncError] = useState<string | null>(null);

  const esRef = useRef<EventSource | null>(null);

  // ── Fetch comments ──────────────────────────────────────────────────────────

  const fetchComments = async (page: number, searchTerm: string) => {
    setLoadingComments(true);
    setCommentsError(null);
    try {
      const params = new URLSearchParams({ page: String(page) });
      if (searchTerm.trim()) params.set('search', searchTerm.trim());
      const res = await fetch(`${goUrl}/api/channels/${channel.ID}/comments?${params.toString()}`);
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as CommentsResponse;
      log.info('comments fetched', { count: data.comments.length, total: data.total });
      setComments(data.comments);
      setTotalComments(data.total);
      setCurrentPage(data.page);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      log.error('fetch comments failed', { err: message });
      setCommentsError(message);
    } finally {
      setLoadingComments(false);
    }
  };

  // Load comments when sheet opens
  useEffect(() => {
    if (open) {
      void fetchComments(1, '');
    }
    // Cleanup SSE on close
    return () => {
      if (!open) {
        esRef.current?.close();
        esRef.current = null;
      }
    };
  }, [open, channel.ID]); // eslint-disable-line react-hooks/exhaustive-deps

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      esRef.current?.close();
    };
  }, []);

  // ── Sync handler ────────────────────────────────────────────────────────────

  const handleSync = async () => {
    setSyncing(true);
    setSyncStatus('running');
    setSyncProgress(null);
    setSyncError(null);

    try {
      const res = await fetch(`${goUrl}/api/channels/${channel.ID}/comments/sync`, {
        method: 'POST',
      });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as { job_id: string };
      log.info('comment sync job started', { jobId: data.job_id });

      // Subscribe to SSE progress
      const url = `${goUrl}/api/channels/${channel.ID}/comments/sync/${data.job_id}/stream`;
      const es = new EventSource(url);
      esRef.current = es;

      es.onmessage = (event) => {
        try {
          type CommentSyncEvent =
            | { type: 'progress'; index: number; total: number; youtube_id: string; comments_saved: number }
            | { type: 'done'; total: number };

          const payload = JSON.parse(event.data as string) as CommentSyncEvent;

          if (payload.type === 'progress') {
            setSyncProgress({ done: payload.index, total: payload.total });
          } else if (payload.type === 'done') {
            log.info('comment sync done', { total: payload.total });
            setSyncStatus('done');
            setSyncing(false);
            es.close();
            esRef.current = null;
            // Reload comments after sync
            void fetchComments(1, search);
          }
        } catch (err) {
          log.error('sync SSE parse error', { err: err instanceof Error ? err.message : String(err) });
        }
      };

      es.onerror = () => {
        log.error('sync SSE connection error');
        setSyncStatus('error');
        setSyncError('Progress stream connection lost');
        setSyncing(false);
        es.close();
        esRef.current = null;
      };
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      log.error('comment sync request failed', { err: message });
      setSyncStatus('error');
      setSyncError(message);
      setSyncing(false);
    }
  };

  // ── Search handler ──────────────────────────────────────────────────────────

  const handleSearch = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value;
    setSearch(val);
    void fetchComments(1, val);
  };

  // ── Pagination ──────────────────────────────────────────────────────────────

  const totalPages = Math.ceil(totalComments / 20);

  const handlePrev = () => {
    if (currentPage > 1) void fetchComments(currentPage - 1, search);
  };

  const handleNext = () => {
    if (currentPage < totalPages) void fetchComments(currentPage + 1, search);
  };

  // ── Sync progress % ─────────────────────────────────────────────────────────

  const syncPct = syncProgress && syncProgress.total > 0
    ? Math.round((syncProgress.done / syncProgress.total) * 100)
    : 0;

  // ── Render ──────────────────────────────────────────────────────────────────

  return (
    <Sheet open={open} onOpenChange={onClose}>
      <SheetContent
        side="right"
        className="w-[520px] sm:max-w-[520px] flex flex-col gap-0 p-0"
      >
        <SheetHeader className="px-4 pt-5 pb-3 border-b border-border">
          <SheetTitle>Comments — {channel.Name}</SheetTitle>
        </SheetHeader>

        <div className="flex-1 overflow-y-auto flex flex-col gap-0">
          {/* Sync section */}
          <div className="px-4 py-4 border-b border-border flex flex-col gap-3">
            <div className="flex items-center gap-3">
              <Button
                size="sm"
                onClick={() => void handleSync()}
                disabled={syncing}
              >
                {syncing ? 'Syncing…' : 'Sync Comments'}
              </Button>
              {syncStatus === 'done' && (
                <span className="text-xs text-success">Sync complete</span>
              )}
              {syncStatus === 'error' && syncError && (
                <span className="text-xs text-destructive">{syncError}</span>
              )}
            </div>

            {syncing && syncProgress && (
              <div className="flex flex-col gap-1.5">
                <div className="flex items-center justify-between text-xs text-subtle">
                  <span>Syncing…</span>
                  <span>{syncProgress.done} / {syncProgress.total}</span>
                </div>
                <Progress value={syncPct} />
              </div>
            )}
          </div>

          {/* Search bar */}
          <div className="px-4 py-3 border-b border-border">
            <Input
              type="search"
              placeholder="Search comments…"
              value={search}
              onChange={handleSearch}
            />
          </div>

          {/* Comments list */}
          <div className="flex-1 overflow-y-auto">
            {commentsError && (
              <div className="px-4 py-3">
                <p className="text-sm text-destructive">{commentsError}</p>
              </div>
            )}

            {loadingComments ? (
              <p className="text-sm text-subtle text-center py-8">Loading comments…</p>
            ) : comments.length === 0 ? (
              <p className="text-sm text-subtle text-center py-8">
                No comments yet. Try syncing first.
              </p>
            ) : (
              <div className="divide-y divide-border">
                {comments.map((comment) => (
                  <div key={comment.id} className="px-4 py-3 flex flex-col gap-1">
                    <div className="flex items-center justify-between gap-2">
                      <span className="text-xs font-medium text-heading truncate">
                        {comment.author}
                      </span>
                      <div className="flex items-center gap-3 shrink-0">
                        {comment.likes > 0 && (
                          <span className="text-xs text-subtle">
                            {comment.likes} like{comment.likes !== 1 ? 's' : ''}
                          </span>
                        )}
                        <span className="text-xs text-subtle">
                          {comment.published_at
                            ? new Date(comment.published_at).toLocaleDateString()
                            : '—'}
                        </span>
                      </div>
                    </div>
                    <p className="text-sm text-prose leading-relaxed">
                      {comment.text.length > 200
                        ? `${comment.text.slice(0, 200)}…`
                        : comment.text}
                    </p>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Pagination footer */}
          {totalPages > 1 && (
            <div className="border-t border-border px-4 py-3 flex items-center justify-between">
              <span className="text-xs text-subtle">
                Page {currentPage} of {totalPages} ({totalComments} total)
              </span>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handlePrev}
                  disabled={currentPage <= 1 || loadingComments}
                >
                  Prev
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleNext}
                  disabled={currentPage >= totalPages || loadingComments}
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}
