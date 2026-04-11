'use client';

import { useEffect, useRef, useState } from 'react';
import { createLogger } from '@/lib/logger';
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Checkbox } from '@/components/ui/checkbox';
import type { Channel } from '@/store/channelStore';
import {
  useChannelConfigStore,
  type ChannelConfig,
} from '@/store/channelConfigStore';

const log = createLogger('ChannelConfigSheet');

interface Props {
  channel: Channel;
  open: boolean;
  onClose: (open: boolean) => void;
}

// ─── Branding Tab ─────────────────────────────────────────────────────────────

function BrandingTab({
  config,
  channelId,
  onChange,
}: {
  config: Partial<ChannelConfig>;
  channelId: number;
  onChange: (patch: Partial<ChannelConfig>) => void;
}) {
  const { uploadLogo } = useChannelConfigStore();
  const fileRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);

  const handleLogoUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploading(true);
    try {
      await uploadLogo(channelId, file);
    } catch (err) {
      log.error('logo upload error', { err: err instanceof Error ? err.message : String(err) });
    } finally {
      setUploading(false);
    }
  };

  const handleRemoveLogo = () => {
    onChange({ branding_logo_path: undefined });
  };

  return (
    <div className="flex flex-col gap-5 p-4">
      {/* Gradient colors */}
      <div className="grid grid-cols-2 gap-4">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="grad-start">Gradient Start</Label>
          <div className="flex items-center gap-2">
            <input
              id="grad-start"
              type="color"
              value={config.gradient_color_start ?? '#000000'}
              onChange={(e) => onChange({ gradient_color_start: e.target.value })}
              className="h-9 w-10 cursor-pointer rounded-md border border-input bg-transparent p-0.5"
            />
            <Input
              value={config.gradient_color_start ?? '#000000'}
              onChange={(e) => onChange({ gradient_color_start: e.target.value })}
              className="font-mono text-xs"
              maxLength={7}
            />
          </div>
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="grad-end">Gradient End</Label>
          <div className="flex items-center gap-2">
            <input
              id="grad-end"
              type="color"
              value={config.gradient_color_end ?? '#000000'}
              onChange={(e) => onChange({ gradient_color_end: e.target.value })}
              className="h-9 w-10 cursor-pointer rounded-md border border-input bg-transparent p-0.5"
            />
            <Input
              value={config.gradient_color_end ?? '#000000'}
              onChange={(e) => onChange({ gradient_color_end: e.target.value })}
              className="font-mono text-xs"
              maxLength={7}
            />
          </div>
        </div>
      </div>

      {/* Font family */}
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="font-family">Thumbnail Font Family</Label>
        <Input
          id="font-family"
          value={config.thumbnail_font_family ?? ''}
          onChange={(e) => onChange({ thumbnail_font_family: e.target.value })}
          placeholder="Anton, Arial, sans-serif"
        />
      </div>

      {/* Text color */}
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="text-color">Thumbnail Text Color</Label>
        <div className="flex items-center gap-2">
          <input
            id="text-color"
            type="color"
            value={config.thumbnail_text_color_1 ?? '#ffffff'}
            onChange={(e) => onChange({ thumbnail_text_color_1: e.target.value })}
            className="h-9 w-10 cursor-pointer rounded-md border border-input bg-transparent p-0.5"
          />
          <Input
            value={config.thumbnail_text_color_1 ?? '#ffffff'}
            onChange={(e) => onChange({ thumbnail_text_color_1: e.target.value })}
            className="font-mono text-xs"
            maxLength={7}
          />
        </div>
      </div>

      {/* Logo upload */}
      <div className="flex flex-col gap-1.5">
        <Label>Branding Logo</Label>
        {config.branding_logo_path ? (
          <div className="flex items-center gap-3">
            <span className="truncate text-sm text-muted-foreground">
              {config.branding_logo_path.split('/').pop()}
            </span>
            <Button
              variant="ghost"
              size="sm"
              className="h-7 text-xs shrink-0 text-destructive hover:text-destructive"
              onClick={handleRemoveLogo}
            >
              Remove
            </Button>
          </div>
        ) : (
          <>
            <input
              ref={fileRef}
              type="file"
              accept="image/*"
              className="hidden"
              onChange={(e) => void handleLogoUpload(e)}
            />
            <Button
              variant="outline"
              size="sm"
              className="w-fit"
              onClick={() => fileRef.current?.click()}
              disabled={uploading}
            >
              {uploading ? 'Uploading…' : 'Choose Image'}
            </Button>
          </>
        )}
      </div>
    </div>
  );
}

// ─── Defaults Tab ─────────────────────────────────────────────────────────────

function DefaultsTab({
  config,
  onChange,
}: {
  config: Partial<ChannelConfig>;
  onChange: (patch: Partial<ChannelConfig>) => void;
}) {
  return (
    <div className="flex flex-col gap-5 p-4">
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="default-category">Default Category ID</Label>
        <Input
          id="default-category"
          type="number"
          value={config.default_category_id ?? 22}
          onChange={(e) =>
            onChange({ default_category_id: parseInt(e.target.value, 10) || 22 })
          }
          placeholder="22"
        />
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="default-tags">Default Tags</Label>
        <textarea
          id="default-tags"
          value={config.default_tags ?? ''}
          onChange={(e) => onChange({ default_tags: e.target.value })}
          placeholder="gaming, highlights, clips"
          rows={3}
          className="w-full resize-none rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-xs outline-none transition-[color,box-shadow] placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-input/30"
        />
        <p className="text-xs text-muted-foreground">Comma-separated</p>
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="max-highlights">Max Highlights</Label>
        <Input
          id="max-highlights"
          type="number"
          value={config.max_highlights ?? ''}
          onChange={(e) => {
            const val = e.target.value;
            onChange({ max_highlights: val === '' ? undefined : parseInt(val, 10) });
          }}
          placeholder="unlimited"
        />
      </div>
    </div>
  );
}

// ─── Processing Tab ────────────────────────────────────────────────────────────

function ProcessingTab({
  config,
  onChange,
}: {
  config: Partial<ChannelConfig>;
  onChange: (patch: Partial<ChannelConfig>) => void;
}) {
  return (
    <div className="flex flex-col gap-5 p-4">
      <div className="flex items-center gap-3">
        <Checkbox
          id="anti-duplicate"
          checked={config.anti_duplicate_enabled ?? false}
          onCheckedChange={(checked) =>
            onChange({ anti_duplicate_enabled: checked === true })
          }
        />
        <div className="flex flex-col gap-0.5">
          <Label htmlFor="anti-duplicate">Anti-Duplicate</Label>
          <p className="text-xs text-muted-foreground">
            Skip clips that are too similar to existing uploads
          </p>
        </div>
      </div>

      <div className="flex items-center gap-3">
        <Checkbox
          id="branding-logo"
          checked={config.branding_logo_enabled ?? false}
          onCheckedChange={(checked) =>
            onChange({ branding_logo_enabled: checked === true })
          }
        />
        <div className="flex flex-col gap-0.5">
          <Label htmlFor="branding-logo">Logo / Watermark</Label>
          <p className="text-xs text-muted-foreground">
            Overlay channel logo on rendered clips
          </p>
        </div>
      </div>
    </div>
  );
}

// ─── Music Blacklist Tab ───────────────────────────────────────────────────────

function MusicTab({ channelId }: { channelId: number }) {
  const { blacklists, fetchBlacklist, addBlacklistPattern, removeBlacklistPattern } =
    useChannelConfigStore();
  const [newPattern, setNewPattern] = useState('');
  const [adding, setAdding] = useState(false);

  useEffect(() => {
    void fetchBlacklist(channelId);
  }, [channelId, fetchBlacklist]);

  const items = blacklists[channelId] ?? [];

  const handleAdd = async () => {
    const trimmed = newPattern.trim();
    if (!trimmed) return;
    setAdding(true);
    try {
      await addBlacklistPattern(channelId, trimmed);
      setNewPattern('');
    } catch (err) {
      log.error('add pattern failed', { err: err instanceof Error ? err.message : String(err) });
    } finally {
      setAdding(false);
    }
  };

  const handleRemove = async (patternId: number) => {
    try {
      await removeBlacklistPattern(channelId, patternId);
    } catch (err) {
      log.error('remove pattern failed', { err: err instanceof Error ? err.message : String(err) });
    }
  };

  return (
    <div className="flex flex-col gap-4 p-4">
      {/* Add new pattern */}
      <div className="flex gap-2">
        <Input
          value={newPattern}
          onChange={(e) => setNewPattern(e.target.value)}
          placeholder="artist name or pattern…"
          onKeyDown={(e) => {
            if (e.key === 'Enter') void handleAdd();
          }}
        />
        <Button
          size="sm"
          onClick={() => void handleAdd()}
          disabled={adding || !newPattern.trim()}
        >
          {adding ? 'Adding…' : 'Add'}
        </Button>
      </div>

      {/* Pattern list */}
      {items.length === 0 ? (
        <p className="text-sm text-muted-foreground text-center py-4">
          No blacklisted patterns
        </p>
      ) : (
        <div className="rounded-md border border-border divide-y divide-border">
          {items.map((item) => (
            <div
              key={item.id}
              className="flex items-center justify-between px-3 py-2"
            >
              <span className="text-sm font-mono truncate">{item.pattern}</span>
              <Button
                variant="ghost"
                size="sm"
                className="h-7 text-xs shrink-0 text-destructive hover:text-destructive"
                onClick={() => void handleRemove(item.id)}
              >
                Delete
              </Button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ─── Main Sheet ────────────────────────────────────────────────────────────────

export function ChannelConfigSheet({ channel, open, onClose }: Props) {
  const { configs, fetchConfig, saveConfig } = useChannelConfigStore();
  const [localConfig, setLocalConfig] = useState<Partial<ChannelConfig>>({});
  const [saving, setSaving] = useState(false);

  // Load config when sheet opens
  useEffect(() => {
    if (open) {
      void fetchConfig(channel.ID);
    }
  }, [open, channel.ID, fetchConfig]);

  // Sync store config into local state when it arrives
  useEffect(() => {
    const stored = configs[channel.ID];
    if (stored) {
      setLocalConfig(stored);
    }
  }, [configs, channel.ID]);

  const handleChange = (patch: Partial<ChannelConfig>) => {
    setLocalConfig((prev) => ({ ...prev, ...patch }));
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      await saveConfig(channel.ID, localConfig);
    } catch (err) {
      log.error('save config failed', { channelId: channel.ID, err: err instanceof Error ? err.message : String(err) });
    } finally {
      setSaving(false);
    }
  };

  return (
    <Sheet open={open} onOpenChange={onClose}>
      <SheetContent
        side="right"
        className="w-[480px] sm:max-w-[480px] flex flex-col gap-0 p-0"
      >
        <SheetHeader className="px-4 pt-5 pb-3 border-b border-border">
          <SheetTitle>Configure {channel.Name}</SheetTitle>
        </SheetHeader>

        <div className="flex-1 overflow-y-auto">
          <Tabs defaultValue="branding" className="h-full">
            <div className="px-4 pt-3 pb-0 border-b border-border">
              <TabsList className="grid grid-cols-4 w-full">
                <TabsTrigger value="branding">Branding</TabsTrigger>
                <TabsTrigger value="defaults">Defaults</TabsTrigger>
                <TabsTrigger value="processing">Processing</TabsTrigger>
                <TabsTrigger value="music">Music</TabsTrigger>
              </TabsList>
            </div>

            <TabsContent value="branding">
              <BrandingTab
                config={localConfig}
                channelId={channel.ID}
                onChange={handleChange}
              />
            </TabsContent>

            <TabsContent value="defaults">
              <DefaultsTab config={localConfig} onChange={handleChange} />
            </TabsContent>

            <TabsContent value="processing">
              <ProcessingTab config={localConfig} onChange={handleChange} />
            </TabsContent>

            <TabsContent value="music">
              <MusicTab channelId={channel.ID} />
            </TabsContent>
          </Tabs>
        </div>

        {/* Save footer */}
        <div className="border-t border-border px-4 py-3 flex justify-end">
          <Button onClick={() => void handleSave()} disabled={saving}>
            {saving ? 'Saving…' : 'Save Changes'}
          </Button>
        </div>
      </SheetContent>
    </Sheet>
  );
}
