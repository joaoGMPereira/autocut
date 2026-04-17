'use client';

import { useEffect, useState, useCallback } from 'react';
import { useAppStore } from '@/store/appStore';
import { usePipelineStore } from '@/store/pipelineStore';
import { createLogger } from '@/lib/logger';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Slider } from '@/components/ui/slider';
import { Switch } from '@/components/ui/switch';
import { Progress } from '@/components/ui/progress';
import type { GatePayload, ThumbnailConfig, FontInfo, ThumbnailTemplate } from '@/types/pipeline';
import { DEFAULT_THUMBNAIL_CONFIG } from '@/types/pipeline';

const log = createLogger('StepThumbnailConfig');

interface StepThumbnailConfigProps {
  historical?: GatePayload;
}

export function StepThumbnailConfig({ historical }: StepThumbnailConfigProps) {
  const goUrl = useAppStore((s) => s.goUrl);
  const activeRunId = usePipelineStore((s) => s.activeRunId);
  const clips = usePipelineStore((s) => s.clips);
  const loadClips = usePipelineStore((s) => s.loadClips);
  const thumbnailProgress = usePipelineStore((s) => s.thumbnailProgress);
  const submitThumbnailConfig = usePipelineStore((s) => s.submitThumbnailConfig);

  const isHistorical = historical !== undefined;

  // Config state
  const [config, setConfig] = useState<ThumbnailConfig>({ ...DEFAULT_THUMBNAIL_CONFIG });
  const [clipTexts, setClipTexts] = useState<Record<string, string>>({});

  // UI state
  const [isGenerating, setIsGenerating] = useState(false);
  const [fonts, setFonts] = useState<FontInfo[]>([]);
  const [templates, setTemplates] = useState<ThumbnailTemplate[]>([]);
  const [templateName, setTemplateName] = useState('');
  const [showTemplateSave, setShowTemplateSave] = useState(false);
  const [regeneratingClipId, setRegeneratingClipId] = useState<number | null>(null);

  // Load clips and fonts on mount
  useEffect(() => {
    if (!activeRunId) return;
    void loadClips(goUrl, activeRunId);
    void fetchFonts();
    void fetchTemplates();
  }, [goUrl, activeRunId]); // eslint-disable-line react-hooks/exhaustive-deps

  const fetchFonts = async () => {
    try {
      const res = await fetch(`${goUrl}/api/thumbnail/fonts`);
      if (!res.ok) return;
      const data = await res.json();
      setFonts(data.fonts ?? []);
      if (data.default_font && config.font_family === 'Impact') {
        setConfig((c) => ({ ...c, font_family: data.default_font }));
      }
    } catch (err) {
      log.error('Failed to fetch fonts', { error: err instanceof Error ? err.message : 'Unknown' });
    }
  };

  const fetchTemplates = async () => {
    try {
      const res = await fetch(`${goUrl}/api/thumbnail/templates`);
      if (!res.ok) return;
      const data = await res.json();
      setTemplates(data.templates ?? []);
    } catch (err) {
      log.error('Failed to fetch templates', { error: err instanceof Error ? err.message : 'Unknown' });
    }
  };

  // Batch generate
  const handleGenerateAll = useCallback(async () => {
    if (!activeRunId) return;
    setIsGenerating(true);
    log.info('Starting batch generation', { runId: activeRunId });
    try {
      const res = await fetch(`${goUrl}/api/thumbnail/runs/${activeRunId}/generate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ config: { ...config, clip_texts: clipTexts } }),
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || `Failed: ${res.status}`);
      }
      // SSE events will update progress
    } catch (err) {
      log.error('Batch generation failed', { error: err instanceof Error ? err.message : 'Unknown' });
      setIsGenerating(false);
    }
  }, [goUrl, activeRunId, config, clipTexts]);

  // Single clip regenerate
  const handleRegenerateSingle = useCallback(async (clipId: number) => {
    if (!activeRunId) return;
    setRegeneratingClipId(clipId);
    const text = clipTexts[String(clipId)] ?? config.text;
    try {
      const res = await fetch(`${goUrl}/api/thumbnail/runs/${activeRunId}/clips/${clipId}/generate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ config: { ...config, text } }),
      });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = await res.json();
      // Update clip in store
      usePipelineStore.setState((s) => ({
        clips: s.clips.map((c) =>
          c.id === clipId ? { ...c, thumbnail_path: data.thumbnail_path } : c
        ),
      }));
    } catch (err) {
      log.error('Single regeneration failed', { clipId, error: err instanceof Error ? err.message : 'Unknown' });
    } finally {
      setRegeneratingClipId(null);
    }
  }, [goUrl, activeRunId, config, clipTexts]);

  // Continue / Skip (advance pipeline)
  const handleContinue = useCallback(async () => {
    if (!activeRunId) return;
    if (isHistorical) {
      await usePipelineStore.getState().resubmitFromBacked(goUrl, 'WAITING_THUMBNAIL_CONFIG', {
        kind: 'thumbnail',
        default_style: 'centered',
      });
    } else {
      await submitThumbnailConfig(goUrl, activeRunId, { default_style: 'centered' });
    }
  }, [goUrl, activeRunId, isHistorical, submitThumbnailConfig]);

  const handleSkip = useCallback(async () => {
    if (!activeRunId) return;
    if (isHistorical) {
      await usePipelineStore.getState().resubmitFromBacked(goUrl, 'WAITING_THUMBNAIL_CONFIG', {
        kind: 'thumbnail',
        default_style: 'skip',
      });
    } else {
      await submitThumbnailConfig(goUrl, activeRunId, { default_style: 'skip' });
    }
  }, [goUrl, activeRunId, isHistorical, submitThumbnailConfig]);

  // Save template
  const handleSaveTemplate = async () => {
    if (!templateName.trim()) return;
    try {
      const res = await fetch(`${goUrl}/api/thumbnail/templates`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: templateName, config }),
      });
      if (!res.ok) {
        const data = await res.json();
        log.error('Save template failed', { error: data.error });
        return;
      }
      setShowTemplateSave(false);
      setTemplateName('');
      void fetchTemplates();
    } catch (err) {
      log.error('Save template failed', { error: err instanceof Error ? err.message : 'Unknown' });
    }
  };

  // Load template
  const handleLoadTemplate = (tmpl: ThumbnailTemplate) => {
    setConfig({ ...tmpl.config, clip_texts: {} });
  };

  // Delete template
  const handleDeleteTemplate = async (name: string) => {
    try {
      await fetch(`${goUrl}/api/thumbnail/templates/${encodeURIComponent(name)}`, { method: 'DELETE' });
      void fetchTemplates();
    } catch (err) {
      log.error('Delete template failed', { error: err instanceof Error ? err.message : 'Unknown' });
    }
  };

  // Progress tracking
  const batchDone = thumbnailProgress?.status === 'batch_done';
  const progressPercent = thumbnailProgress
    ? (thumbnailProgress.clipIndex / thumbnailProgress.totalClips) * 100
    : 0;

  // Stop showing generating state when batch is done
  useEffect(() => {
    if (batchDone) setIsGenerating(false);
  }, [batchDone]);

  // Check if any clip already has a thumbnail (for re-entry)
  const hasExistingThumbnails = clips.some((c) => c.thumbnail_path !== '');

  return (
    <div className="space-y-4 w-full max-w-3xl">
      <h2 className="text-xl font-semibold text-foreground">Thumbnail Configuration</h2>

      {isHistorical && (
        <div className="rounded-md border border-zinc-700 bg-zinc-900 px-4 py-3 text-xs text-zinc-400">
          Reviewing previous submission. Re-submitting will restart the pipeline from this step.
        </div>
      )}

      {/* Config Panel */}
      <div className="space-y-3 rounded-lg border border-zinc-800 bg-zinc-900/50 p-4">
        <h3 className="text-sm font-medium text-zinc-300">Visual Settings</h3>

        {/* Global Text */}
        <div className="space-y-1">
          <Label className="text-xs text-zinc-400">Overlay Text (global)</Label>
          <Input
            value={config.text}
            onChange={(e) => setConfig((c) => ({ ...c, text: e.target.value }))}
            placeholder="e.g. BEST MOMENT EVER"
            maxLength={200}
            className="bg-zinc-800 border-zinc-700"
          />
          <p className="text-[10px] text-zinc-500">Text is auto-uppercased. Max 200 chars.</p>
        </div>

        {/* Font Picker */}
        <div className="space-y-1">
          <Label className="text-xs text-zinc-400">Font Family</Label>
          <select
            value={config.font_family}
            onChange={(e) => setConfig((c) => ({ ...c, font_family: e.target.value }))}
            className="w-full rounded-md border border-zinc-700 bg-zinc-800 px-3 py-2 text-sm text-foreground"
          >
            {fonts.map((f) => (
              <option key={f.family} value={f.family}>{f.family}</option>
            ))}
            {fonts.length === 0 && <option value="Impact">Impact (default)</option>}
          </select>
        </div>

        {/* Color pickers row */}
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1">
            <Label className="text-xs text-zinc-400">Text Color</Label>
            <div className="flex items-center gap-2">
              <input
                type="color"
                value={config.text_color}
                onChange={(e) => setConfig((c) => ({ ...c, text_color: e.target.value }))}
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
                onChange={(e) => setConfig((c) => ({ ...c, outline_color: e.target.value }))}
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
            onValueChange={([v]) => setConfig((c) => ({ ...c, stroke_width: v }))}
            min={0} max={20} step={1}
          />
        </div>

        {/* Blur toggle + amount */}
        <div className="flex items-center gap-3">
          <Switch
            checked={config.blur_enabled}
            onCheckedChange={(v) => setConfig((c) => ({ ...c, blur_enabled: v }))}
          />
          <Label className="text-xs text-zinc-400">Background Blur</Label>
          {config.blur_enabled && (
            <div className="flex-1">
              <Slider
                value={[config.blur_amount]}
                onValueChange={([v]) => setConfig((c) => ({ ...c, blur_amount: v }))}
                min={1} max={100} step={1}
              />
              <span className="text-[10px] text-zinc-500">{config.blur_amount}</span>
            </div>
          )}
        </div>

        {/* Darken toggle + amount */}
        <div className="flex items-center gap-3">
          <Switch
            checked={config.darken_enabled}
            onCheckedChange={(v) => setConfig((c) => ({ ...c, darken_enabled: v }))}
          />
          <Label className="text-xs text-zinc-400">Darken Background</Label>
          {config.darken_enabled && (
            <div className="flex-1">
              <Slider
                value={[Math.round(config.darken_amount * 100)]}
                onValueChange={([v]) => setConfig((c) => ({ ...c, darken_amount: v / 100 }))}
                min={0} max={100} step={1}
              />
              <span className="text-[10px] text-zinc-500">{Math.round(config.darken_amount * 100)}%</span>
            </div>
          )}
        </div>

        {/* Center Scale */}
        <div className="space-y-1">
          <Label className="text-xs text-zinc-400">Center Scale: {Math.round(config.center_scale * 100)}%</Label>
          <Slider
            value={[Math.round(config.center_scale * 100)]}
            onValueChange={([v]) => setConfig((c) => ({ ...c, center_scale: v / 100 }))}
            min={30} max={100} step={1}
          />
        </div>

        {/* Corner Radius */}
        <div className="space-y-1">
          <Label className="text-xs text-zinc-400">Corner Radius: {config.corner_radius}px</Label>
          <Slider
            value={[config.corner_radius]}
            onValueChange={([v]) => setConfig((c) => ({ ...c, corner_radius: v }))}
            min={0} max={100} step={1}
          />
        </div>
      </div>

      {/* Templates */}
      {templates.length > 0 && (
        <div className="space-y-2 rounded-lg border border-zinc-800 bg-zinc-900/50 p-4">
          <h3 className="text-sm font-medium text-zinc-300">Templates</h3>
          <div className="flex flex-wrap gap-2">
            {templates.map((t) => (
              <div key={t.name} className="flex items-center gap-1">
                <Button variant="outline" size="xs" onClick={() => handleLoadTemplate(t)}>
                  {t.name}
                </Button>
                <Button
                  variant="ghost" size="icon-xs"
                  onClick={() => handleDeleteTemplate(t.name)}
                  className="text-zinc-500 hover:text-red-400"
                >
                  &times;
                </Button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Save Template */}
      <div className="flex items-center gap-2">
        {showTemplateSave ? (
          <>
            <Input
              value={templateName}
              onChange={(e) => setTemplateName(e.target.value)}
              placeholder="Template name"
              className="bg-zinc-800 border-zinc-700 w-48"
            />
            <Button variant="outline" size="sm" onClick={handleSaveTemplate} disabled={!templateName.trim()}>
              Save
            </Button>
            <Button variant="ghost" size="sm" onClick={() => setShowTemplateSave(false)}>
              Cancel
            </Button>
          </>
        ) : (
          <Button variant="ghost" size="sm" onClick={() => setShowTemplateSave(true)}>
            Save as Template
          </Button>
        )}
      </div>

      {/* Generate Button + Progress */}
      <div className="space-y-2">
        <Button
          onClick={handleGenerateAll}
          disabled={isGenerating || clips.length === 0}
          className="w-full"
        >
          {isGenerating ? `Generating... (${thumbnailProgress?.clipIndex ?? 0}/${thumbnailProgress?.totalClips ?? clips.length})` : 'Generate All Thumbnails'}
        </Button>

        {isGenerating && (
          <Progress value={progressPercent} className="w-full" />
        )}

        {batchDone && thumbnailProgress && (
          <p className="text-xs text-zinc-400">
            Done: {thumbnailProgress.successCount ?? 0} succeeded, {thumbnailProgress.errorCount ?? 0} failed
          </p>
        )}
      </div>

      {/* Thumbnail Preview Grid */}
      {clips.length > 0 && (
        <div className="space-y-2">
          <h3 className="text-sm font-medium text-zinc-300">Clips ({clips.length})</h3>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
            {clips.map((clip) => (
              <div key={clip.id} className="space-y-1 rounded-lg border border-zinc-800 bg-zinc-900/50 p-2">
                {/* Thumbnail preview */}
                {clip.thumbnail_path ? (
                  <img
                    src={`${goUrl}/api/thumbnail/runs/${activeRunId}/clips/${clip.id}/file?t=${Date.now()}`}
                    alt={`Clip ${clip.id} thumbnail`}
                    className="w-full rounded aspect-[9/16] object-cover bg-zinc-800"
                  />
                ) : (
                  <div className="w-full rounded aspect-[9/16] bg-zinc-800 flex items-center justify-center">
                    <span className="text-xs text-zinc-500">No thumbnail</span>
                  </div>
                )}

                {/* Per-clip text override */}
                <Input
                  value={clipTexts[String(clip.id)] ?? ''}
                  onChange={(e) => setClipTexts((prev) => ({ ...prev, [String(clip.id)]: e.target.value }))}
                  placeholder={config.text || 'Per-clip text...'}
                  className="bg-zinc-800 border-zinc-700 text-xs h-7"
                />

                {/* Regenerate button */}
                <Button
                  variant="outline"
                  size="xs"
                  className="w-full"
                  disabled={regeneratingClipId === clip.id || isGenerating}
                  onClick={() => handleRegenerateSingle(clip.id)}
                >
                  {regeneratingClipId === clip.id ? 'Generating...' : 'Regenerate'}
                </Button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Continue / Skip */}
      <div className="flex gap-2 pt-2">
        <Button onClick={handleContinue} disabled={isGenerating}>
          {hasExistingThumbnails || batchDone ? 'Continue' : 'Continue Without Thumbnails'}
        </Button>
        <Button variant="ghost" onClick={handleSkip} disabled={isGenerating}>
          Skip
        </Button>
      </div>
    </div>
  );
}
