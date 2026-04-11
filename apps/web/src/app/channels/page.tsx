'use client';

import { Anton } from 'next/font/google';
import { useEffect, useState } from 'react';
import { MessageSquare, Settings2 } from 'lucide-react';
import { useAppStore } from '@/store/appStore';
import { useChannelStore } from '@/store/channelStore';
import type { Channel } from '@/store/channelStore';
import { Button } from '@/components/ui/button';
import { ChannelConfigSheet } from '@/components/channels/ChannelConfigSheet';
import { CommentSyncSheet } from '@/components/channels/CommentSyncSheet';

const anton = Anton({ weight: '400', subsets: ['latin'], display: 'swap' });

export default function ChannelsPage() {
  const { goUrl } = useAppStore();
  const { channels, loading, error, fetchChannels, addChannel, removeChannel } = useChannelStore();

  const [showAddForm, setShowAddForm] = useState(false);
  const [newName, setNewName] = useState('');
  const [newChannelId, setNewChannelId] = useState('');
  const [addError, setAddError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [configChannel, setConfigChannel] = useState<Channel | null>(null);
  const [commentsChannel, setCommentsChannel] = useState<Channel | null>(null);

  useEffect(() => {
    void fetchChannels(goUrl);
  }, [goUrl, fetchChannels]);

  const handleAdd = async () => {
    if (!newName.trim() || !newChannelId.trim()) return;
    setSubmitting(true);
    setAddError(null);
    try {
      await addChannel(goUrl, { name: newName.trim(), youtube_channel_id: newChannelId.trim() });
      setNewName('');
      setNewChannelId('');
      setShowAddForm(false);
    } catch (err) {
      setAddError(err instanceof Error ? err.message : 'Failed to add channel');
    } finally {
      setSubmitting(false);
    }
  };

  const handleRemove = async (id: number) => {
    try {
      await removeChannel(goUrl, id);
    } catch {
      /* list will not update on error */
    }
  };

  return (
    <div className="px-10 py-8 flex flex-col gap-7">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className={`${anton.className} text-[28px] tracking-[1.5px] text-[#F0F0F8]`}>
          CHANNELS
        </h1>
        {!showAddForm && (
          <Button onClick={() => setShowAddForm(true)}>+ Add Channel</Button>
        )}
      </div>

      {/* Add channel form */}
      {showAddForm && (
        <div className="rounded-xl bg-card border border-border p-5 flex flex-col gap-4">
          <h2 className="text-sm font-semibold">New Channel</h2>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-xs text-[#5C5C80] mb-1 block">Channel Name</label>
              <input
                type="text"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder="My Channel"
                className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
              />
            </div>
            <div>
              <label className="text-xs text-[#5C5C80] mb-1 block">YouTube Channel ID</label>
              <input
                type="text"
                value={newChannelId}
                onChange={(e) => setNewChannelId(e.target.value)}
                placeholder="UCxxxxxxxxxxxxxxxxxx"
                className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
              />
            </div>
          </div>
          {addError && (
            <p className="text-xs text-destructive">{addError}</p>
          )}
          <div className="flex items-center gap-2">
            <Button
              onClick={() => void handleAdd()}
              disabled={submitting || !newName.trim() || !newChannelId.trim()}
            >
              {submitting ? 'Adding…' : 'Add'}
            </Button>
            <Button
              variant="ghost"
              onClick={() => {
                setShowAddForm(false);
                setNewName('');
                setNewChannelId('');
                setAddError(null);
              }}
            >
              Cancel
            </Button>
          </div>
        </div>
      )}

      {/* Error state */}
      {error && (
        <div className="rounded-xl bg-destructive/10 border border-destructive/20 p-4">
          <p className="text-sm text-destructive">{error}</p>
        </div>
      )}

      {/* Channel list */}
      {loading && channels.length === 0 ? (
        <div className="rounded-xl bg-card border border-border p-5">
          <p className="text-[#5C5C80] text-sm text-center py-6">Loading channels…</p>
        </div>
      ) : channels.length === 0 ? (
        <div className="rounded-xl bg-card border border-border p-5">
          <p className="text-[#5C5C80] text-sm text-center py-6">No channels configured yet</p>
        </div>
      ) : (
        <div className="rounded-xl bg-card border border-border divide-y divide-border">
          {channels.map((ch) => (
            <div key={ch.ID} className="flex items-center justify-between px-5 py-3">
              <span className="text-sm font-medium">{ch.Name}</span>
              <div className="flex items-center gap-3">
                <span className="bg-[#1A1A26] text-[#5C5C80] text-xs px-2 py-0.5 rounded-md">
                  {ch.YouTubeChannelID}
                </span>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setCommentsChannel(ch)}
                  aria-label={`Comments for ${ch.Name}`}
                >
                  <MessageSquare className="size-4" />
                  Comments
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setConfigChannel(ch)}
                  aria-label={`Configure ${ch.Name}`}
                >
                  <Settings2 className="size-4" />
                  Configure
                </Button>
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => void handleRemove(ch.ID)}
                >
                  Delete
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Channel config sheet */}
      {configChannel && (
        <ChannelConfigSheet
          channel={configChannel}
          open={configChannel !== null}
          onClose={(open) => {
            if (!open) setConfigChannel(null);
          }}
        />
      )}

      {/* Comment sync sheet */}
      {commentsChannel && (
        <CommentSyncSheet
          channel={commentsChannel}
          open={commentsChannel !== null}
          onClose={(open) => {
            if (!open) setCommentsChannel(null);
          }}
        />
      )}
    </div>
  );
}
