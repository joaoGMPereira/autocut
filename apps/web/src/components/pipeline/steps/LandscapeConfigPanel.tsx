'use client';

import { Label } from '@/components/ui/label';
import { Slider } from '@/components/ui/slider';
import { Switch } from '@/components/ui/switch';
import type { LandscapeThumbnailConfig, FontInfo } from '@/types/pipeline';

interface LandscapeConfigPanelProps {
  config: LandscapeThumbnailConfig;
  onChange: (config: LandscapeThumbnailConfig) => void;
  fonts: FontInfo[];
}

export function LandscapeConfigPanel({
  config, onChange, fonts,
}: LandscapeConfigPanelProps) {
  const set = <K extends keyof LandscapeThumbnailConfig>(
    key: K, value: LandscapeThumbnailConfig[K]
  ) => onChange({ ...config, [key]: value });

  return (
    <div className="space-y-3 rounded-lg border border-zinc-800 bg-zinc-900/50 p-4">
      <h3 className="text-sm font-medium text-zinc-300">Landscape Settings</h3>

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
