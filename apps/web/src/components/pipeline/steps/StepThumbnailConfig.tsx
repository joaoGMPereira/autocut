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
import type { GatePayload, ThumbnailConfig, LandscapeThumbnailConfig, FontInfo, ThumbnailTemplate } from '@/types/pipeline';
import { DEFAULT_THUMBNAIL_CONFIG, DEFAULT_LANDSCAPE_CONFIG } from '@/types/pipeline';
import { LandscapeConfigPanel } from './LandscapeConfigPanel';

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

  const run = usePipelineStore((s) => s.run);
  const videoInfo = usePipelineStore((s) => s.videoInfo);
  const isLandscapeMode = run?.mode === 'ai' || run?.mode === 'longform';

  const isHistorical = historical !== undefined;

  // Config state
  const [config, setConfig] = useState<ThumbnailConfig>({ ...DEFAULT_THUMBNAIL_CONFIG });
  const [landConfig, setLandConfig] = useState<LandscapeThumbnailConfig>({ ...DEFAULT_LANDSCAPE_CONFIG });
  const [baseImagePath, setBaseImagePath] = useState('');
  const [clipTexts, setClipTexts] = useState<Record<string, string>>({});
  const [clipBaseImages, setClipBaseImages] = useState<Record<string, string>>({});

  // UI state
  const [isGenerating, setIsGenerating] = useState(false);
  const [uploadingClipId, setUploadingClipId] = useState<number | null>(null);
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
    log.info('Starting batch generation', { runId: activeRunId, landscape: isLandscapeMode });
    try {
      const body: Record<string, unknown> = {
        config: { ...config, clip_texts: clipTexts },
      };
      if (isLandscapeMode) {
        body.landscape_config = {
          ...landConfig,
          clip_texts: clipTexts,
          base_image_path: baseImagePath,
          clip_base_images: clipBaseImages,
        };
        if (videoInfo?.thumbnailUrl) {
          body.thumbnail_url = videoInfo.thumbnailUrl;
        }
      }
      const res = await fetch(`${goUrl}/api/thumbnail/runs/${activeRunId}/generate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
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
  }, [goUrl, activeRunId, config, clipTexts, isLandscapeMode, landConfig, baseImagePath, clipBaseImages, videoInfo]);

  // Upload per-clip base image
  const handleUploadClipBaseImage = useCallback(async (clipId: number, file: File) => {
    if (!activeRunId) return;
    setUploadingClipId(clipId);
    try {
      const formData = new FormData();
      formData.append('file', file);
      const res = await fetch(`${goUrl}/api/thumbnail/runs/${activeRunId}/base-image`, {
        method: 'POST',
        body: formData,
      });
      if (!res.ok) throw new Error(`Upload failed: ${res.status}`);
      const data = await res.json();
      setClipBaseImages((prev) => ({ ...prev, [String(clipId)]: data.path }));
    } catch (err) {
      log.error('Clip base image upload failed', { clipId, error: err instanceof Error ? err.message : 'Unknown' });
    } finally {
      setUploadingClipId(null);
    }
  }, [goUrl, activeRunId]);

  // Single clip regenerate
  const handleRegenerateSingle = useCallback(async (clipId: number) => {
    if (!activeRunId) return;
    setRegeneratingClipId(clipId);
    const text = clipTexts[String(clipId)] ?? config.text;
    try {
      const body: Record<string, unknown> = {
        config: { ...config, text },
      };
      if (isLandscapeMode) {
        const clipImg = clipBaseImages[String(clipId)];
        body.landscape_config = {
          ...landConfig,
          clip_texts: { [String(clipId)]: text },
          base_image_path: baseImagePath,
          ...(clipImg ? { clip_base_images: { [String(clipId)]: clipImg } } : {}),
        };
        if (videoInfo?.thumbnailUrl) {
          body.thumbnail_url = videoInfo.thumbnailUrl;
        }
      }
      const res = await fetch(`${goUrl}/api/thumbnail/runs/${activeRunId}/clips/${clipId}/generate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
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
  }, [goUrl, activeRunId, config, clipTexts, isLandscapeMode, landConfig, baseImagePath, videoInfo]);

  // Continue / Skip (advance pipeline)
  const handleContinue = useCallback(async () => {
    if (!activeRunId) return;
    const styleValue = isLandscapeMode ? 'landscape' : 'centered';
    if (isHistorical) {
      await usePipelineStore.getState().resubmitFromBacked(goUrl, 'WAITING_THUMBNAIL_CONFIG', {
        kind: 'thumbnail',
        default_style: styleValue,
      });
    } else {
      await submitThumbnailConfig(goUrl, activeRunId, { default_style: styleValue });
    }
  }, [goUrl, activeRunId, isHistorical, isLandscapeMode, submitThumbnailConfig]);

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
        body: JSON.stringify({
          name: templateName,
          config,
          ...(isLandscapeMode && { landscape_config: landConfig }),
          strategy_type: isLandscapeMode ? 'landscape' : 'shorts',
        }),
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
    if (tmpl.landscape_config) {
      setLandConfig({ ...tmpl.landscape_config, clip_texts: {} });
    }
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

  // Filter templates by strategy type (T4.4)
  const filteredTemplates = templates.filter((t) => {
    if (isLandscapeMode) return t.strategy_type === 'landscape';
    return !t.strategy_type || t.strategy_type === 'shorts';
  });

  const aspectClass = isLandscapeMode ? 'aspect-[16/9]' : 'aspect-[9/16]';
  const gridClass = isLandscapeMode
    ? 'grid grid-cols-1 gap-3 sm:grid-cols-2'
    : 'grid grid-cols-1 gap-3 max-w-sm mx-auto';

  return (
    <div data-testid="step-thumbnail-config" className="space-y-4 w-full max-w-3xl">
      <h2 className="text-xl font-semibold text-foreground">Thumbnail Configuration</h2>

      {isHistorical && (
        <div className="rounded-md border border-zinc-700 bg-zinc-900 px-4 py-3 text-xs text-zinc-400">
          Reviewing previous submission. Re-submitting will restart the pipeline from this step.
        </div>
      )}

      {/* Config Panel — branch by mode */}
      {isLandscapeMode ? (
        <LandscapeConfigPanel
          config={landConfig}
          onChange={setLandConfig}
          fonts={fonts}
          goUrl={goUrl}
          activeRunId={activeRunId}
          baseImagePath={baseImagePath}
          onBaseImagePathChange={setBaseImagePath}
          videoThumbnailUrl={videoInfo?.thumbnailUrl}
        />
      ) : (
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
      )}

      {/* Global text input (shown in landscape mode since the panel above doesn't have it) */}
      {isLandscapeMode && (
        <div className="space-y-1 rounded-lg border border-zinc-800 bg-zinc-900/50 p-4">
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
      )}

      {/* Templates */}
      {filteredTemplates.length > 0 && (
        <div className="space-y-2 rounded-lg border border-zinc-800 bg-zinc-900/50 p-4">
          <h3 className="text-sm font-medium text-zinc-300">Templates</h3>
          <div className="flex flex-wrap gap-2">
            {filteredTemplates.map((t) => (
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
          <div className={gridClass}>
            {clips.map((clip, i) => (
              <div key={clip.id} className="space-y-1 rounded-lg border border-zinc-800 bg-zinc-900/50 p-2">
                {/* Thumbnail preview */}
                {clip.thumbnail_path ? (
                  <img
                    src={`${goUrl}/api/thumbnail/runs/${activeRunId}/clips/${clip.id}/file?t=${Date.now()}`}
                    alt={`Clip ${clip.id} thumbnail`}
                    className={`w-full rounded ${aspectClass} object-cover bg-zinc-800`}
                  />
                ) : videoInfo?.thumbnailUrl && isLandscapeMode ? (
                  <img
                    src={videoInfo.thumbnailUrl}
                    alt="YouTube thumbnail preview"
                    className={`w-full rounded ${aspectClass} object-cover bg-zinc-800 opacity-60`}
                  />
                ) : (
                  <div className={`w-full rounded ${aspectClass} bg-zinc-800 flex items-center justify-center`}>
                    <span className="text-xs text-zinc-500">No thumbnail</span>
                  </div>
                )}

                {/* Per-clip text override */}
                <Input
                  value={clipTexts[String(clip.id)] ?? ''}
                  onChange={(e) => setClipTexts((prev) => ({ ...prev, [String(clip.id)]: e.target.value }))}
                  placeholder={isLandscapeMode ? `PT${i + 1}` : (config.text || 'Per-clip text...')}
                  className="bg-zinc-800 border-zinc-700 text-xs h-7"
                />

                {/* Per-clip base image (landscape mode) */}
                {isLandscapeMode && (
                  <div className="space-y-1">
                    <Input
                      value={clipBaseImages[String(clip.id)] ?? ''}
                      onChange={(e) => setClipBaseImages((prev) => ({ ...prev, [String(clip.id)]: e.target.value }))}
                      placeholder="Image URL or path..."
                      className="bg-zinc-800 border-zinc-700 text-xs h-7"
                    />
                    <div className="flex items-center gap-2">
                      <label className="cursor-pointer text-[10px] text-blue-400 hover:text-blue-300">
                        <input
                          type="file"
                          accept="image/*"
                          className="hidden"
                          onChange={(e) => {
                            const file = e.target.files?.[0];
                            if (file) handleUploadClipBaseImage(clip.id, file);
                          }}
                        />
                        {uploadingClipId === clip.id ? 'Uploading...' : 'Upload image'}
                      </label>
                      {clipBaseImages[String(clip.id)] && (
                        <button
                          onClick={() => setClipBaseImages((prev) => {
                            const next = { ...prev };
                            delete next[String(clip.id)];
                            return next;
                          })}
                          className="text-[10px] text-zinc-500 hover:text-zinc-400"
                        >
                          Clear
                        </button>
                      )}
                    </div>
                  </div>
                )}

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
