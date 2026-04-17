'use client';

import { useState } from 'react';
import { Label } from '@/components/ui/label';
import { Slider } from '@/components/ui/slider';
import { Switch } from '@/components/ui/switch';
import type { LandscapeThumbnailConfig, FontInfo } from '@/types/pipeline';
import { createLogger } from '@/lib/logger';

const log = createLogger('LandscapeConfigPanel');

interface LandscapeConfigPanelProps {
  config: LandscapeThumbnailConfig;
  onChange: (config: LandscapeThumbnailConfig) => void;
  fonts: FontInfo[];
  goUrl: string;
  activeRunId: number | null;
  baseImagePath: string;
  onBaseImagePathChange: (path: string) => void;
  videoThumbnailUrl?: string;
}

export function LandscapeConfigPanel({
  config, onChange, fonts, goUrl, activeRunId,
  baseImagePath, onBaseImagePathChange, videoThumbnailUrl,
}: LandscapeConfigPanelProps) {
  const [uploading, setUploading] = useState(false);

  const set = <K extends keyof LandscapeThumbnailConfig>(
    key: K, value: LandscapeThumbnailConfig[K]
  ) => onChange({ ...config, [key]: value });

  const handleUploadBaseImage = async (file: File) => {
    if (!activeRunId) return;
    setUploading(true);
    try {
      const formData = new FormData();
      formData.append('file', file);
      const res = await fetch(`${goUrl}/api/thumbnail/runs/${activeRunId}/base-image`, {
        method: 'POST',
        body: formData,
      });
      if (!res.ok) throw new Error(`Upload failed: ${res.status}`);
      const data = await res.json();
      onBaseImagePathChange(data.path);
    } catch (err) {
      log.error('Base image upload failed', { error: err instanceof Error ? err.message : 'Unknown' });
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="space-y-3 rounded-lg border border-zinc-800 bg-zinc-900/50 p-4">
      <h3 className="text-sm font-medium text-zinc-300">Landscape Settings</h3>

      {/* Base Image Section */}
      <div className="space-y-2">
        <Label className="text-xs text-zinc-400">Base Image</Label>
        {(baseImagePath || videoThumbnailUrl) && (
          <div className="relative rounded overflow-hidden aspect-[16/9] bg-zinc-800">
            {baseImagePath ? (
              <div className="flex items-center justify-center h-full">
                <span className="text-xs text-zinc-300">Custom image uploaded</span>
              </div>
            ) : (
              <img
                src={videoThumbnailUrl}
                alt="YouTube thumbnail"
                className="w-full h-full object-cover"
              />
            )}
            {baseImagePath && (
              <span className="absolute top-1 left-1 text-[10px] bg-black/60 px-1 rounded text-zinc-300">
                Custom
              </span>
            )}
          </div>
        )}
        <label className="inline-flex items-center gap-2 cursor-pointer text-xs text-blue-400 hover:text-blue-300">
          <input
            type="file"
            accept="image/*"
            className="hidden"
            onChange={(e) => {
              const file = e.target.files?.[0];
              if (file) handleUploadBaseImage(file);
            }}
          />
          {uploading ? 'Uploading...' : 'Upload Custom Image (16:9)'}
        </label>
        {baseImagePath && (
          <button
            onClick={() => onBaseImagePathChange('')}
            className="text-[10px] text-zinc-500 hover:text-zinc-400"
          >
            Reset to YouTube thumbnail
          </button>
        )}
      </div>

      {/* Gradient Toggle + Colors */}
      <div className="flex items-center gap-3">
        <Switch
          checked={config.apply_gradient}
          onCheckedChange={(v) => set('apply_gradient', v)}
        />
        <Label className="text-xs text-zinc-400">Gradient Border</Label>
      </div>

      {config.apply_gradient && (
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1">
            <Label className="text-xs text-zinc-400">Gradient Start</Label>
            <div className="flex items-center gap-2">
              <input
                type="color"
                value={config.gradient_start}
                onChange={(e) => set('gradient_start', e.target.value)}
                className="h-8 w-8 cursor-pointer rounded border border-zinc-700"
              />
              <span className="text-xs text-zinc-400">{config.gradient_start}</span>
            </div>
          </div>
          <div className="space-y-1">
            <Label className="text-xs text-zinc-400">Gradient End</Label>
            <div className="flex items-center gap-2">
              <input
                type="color"
                value={config.gradient_end}
                onChange={(e) => set('gradient_end', e.target.value)}
                className="h-8 w-8 cursor-pointer rounded border border-zinc-700"
              />
              <span className="text-xs text-zinc-400">{config.gradient_end}</span>
            </div>
          </div>
        </div>
      )}

      {config.apply_gradient && (
        <>
          {/* Border Thickness */}
          <div className="space-y-1">
            <Label className="text-xs text-zinc-400">Border Thickness: {config.border_size}px</Label>
            <Slider
              value={[config.border_size]}
              onValueChange={([v]) => set('border_size', v)}
              min={0} max={60} step={1}
            />
          </div>

          {/* Corner Radius */}
          <div className="space-y-1">
            <Label className="text-xs text-zinc-400">Corner Radius: {config.corner_radius}px</Label>
            <Slider
              value={[config.corner_radius]}
              onValueChange={([v]) => set('corner_radius', v)}
              min={0} max={80} step={1}
            />
          </div>
        </>
      )}

      {/* Blur + Darken toggles */}
      <div className="flex items-center gap-3">
        <Switch
          checked={config.apply_blur}
          onCheckedChange={(v) => set('apply_blur', v)}
        />
        <Label className="text-xs text-zinc-400">Background Blur</Label>
      </div>
      <div className="flex items-center gap-3">
        <Switch
          checked={config.apply_darken}
          onCheckedChange={(v) => set('apply_darken', v)}
        />
        <Label className="text-xs text-zinc-400">Darken Background</Label>
      </div>

      {/* Text Position */}
      <div className="space-y-1">
        <Label className="text-xs text-zinc-400">Text Position</Label>
        <select
          value={config.text_position}
          onChange={(e) => set('text_position', e.target.value)}
          className="w-full rounded-md border border-zinc-700 bg-zinc-800 px-3 py-2 text-sm text-foreground"
        >
          <option value="top_right">Top Right</option>
          <option value="top_left">Top Left</option>
          <option value="center">Center</option>
          <option value="bottom_center">Bottom Center</option>
        </select>
      </div>

      {/* Font Picker */}
      <div className="space-y-1">
        <Label className="text-xs text-zinc-400">Font Family</Label>
        <select
          value={config.font_family}
          onChange={(e) => set('font_family', e.target.value)}
          className="w-full rounded-md border border-zinc-700 bg-zinc-800 px-3 py-2 text-sm text-foreground"
        >
          {fonts.map((f) => (
            <option key={f.family} value={f.family}>{f.family}</option>
          ))}
          {fonts.length === 0 && <option value="Impact">Impact (default)</option>}
        </select>
      </div>

      {/* Text Color + Outline */}
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1">
          <Label className="text-xs text-zinc-400">Text Color</Label>
          <div className="flex items-center gap-2">
            <input
              type="color"
              value={config.text_color}
              onChange={(e) => set('text_color', e.target.value)}
              className="h-8 w-8 cursor-pointer rounded border border-zinc-700"
            />
            <span className="text-xs text-zinc-400">{config.text_color}</span>
          </div>
        </div>
        <div className="space-y-1">
          <Label className="text-xs text-zinc-400">Outline Color</Label>
          <div className="flex items-center gap-2">
            <input
              type="color"
              value={config.outline_color}
              onChange={(e) => set('outline_color', e.target.value)}
              className="h-8 w-8 cursor-pointer rounded border border-zinc-700"
            />
            <span className="text-xs text-zinc-400">{config.outline_color}</span>
          </div>
        </div>
      </div>

      {/* Stroke Width */}
      <div className="space-y-1">
        <Label className="text-xs text-zinc-400">Stroke Width: {config.stroke_width}px</Label>
        <Slider
          value={[config.stroke_width]}
          onValueChange={([v]) => set('stroke_width', v)}
          min={0} max={20} step={1}
        />
      </div>
    </div>
  );
}
