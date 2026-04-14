'use client';

import { useEffect, useState } from 'react';
import { useAppStore } from '@/store/appStore';
import { usePipelineStore } from '@/store/pipelineStore';
import { AI_DEFAULTS, LONGFORM_DEFAULTS } from '@/types/pipeline';
import type { AntiDupEffects, AntiDuplicateConfig, CaptionsConfig, GatePayload, ModeConfig, PriorClipsResponse, WorkflowMode } from '@/types/pipeline';

interface StepModeProps {
  historical?: GatePayload;
}

export function StepMode({ historical }: StepModeProps) {
  const goUrl = useAppStore((s) => s.goUrl);
  const activeRunId = usePipelineStore((s) => s.activeRunId);
  const advance = usePipelineStore((s) => s.advance);
  const run = usePipelineStore((s) => s.run);
  const videoInfo = usePipelineStore((s) => s.videoInfo);

  const isHistorical = historical !== undefined;

  // ── Primary selection ──────────────────────────────────────────────────────
  const [selectedMode, setSelectedMode] = useState<WorkflowMode>('longform');

  // ── AI config state ────────────────────────────────────────────────────────
  const [sensitivityPct, setSensitivityPct] = useState<number>(AI_DEFAULTS.sensitivity_pct ?? 70);
  const [skipMusicMins, setSkipMusicMins] = useState<number | null>(AI_DEFAULTS.skip_music_mins ?? null);
  const [forceRegenerate, setForceRegenerate] = useState<boolean>(AI_DEFAULTS.force_regenerate ?? true);
  const [skipTranscription, setSkipTranscription] = useState<boolean>(AI_DEFAULTS.skip_transcription ?? false);

  // ── AI duration config (new) ───────────────────────────────────────────────
  const [clipDurationSecs, setClipDurationSecs] = useState<number>(AI_DEFAULTS.max_duration_secs ?? 1200);
  const [minDurationSecs, setMinDurationSecs] = useState<number>(AI_DEFAULTS.min_duration_secs ?? 480);

  // ── Long Form config state ─────────────────────────────────────────────────
  const [segmentSecs, setSegmentSecs] = useState<number>(LONGFORM_DEFAULTS.segment_secs ?? 600);
  const [minPartSecs, setMinPartSecs] = useState<number>(0);

  // ── Anti-duplicate config (new) ────────────────────────────────────────────
  const [antiDupEnabled, setAntiDupEnabled] = useState<boolean>(false);
  const [antiDupMode, setAntiDupMode] = useState<'subtle' | 'aggressive'>('subtle');
  const [antiDupEffects, setAntiDupEffects] = useState<AntiDupEffects>({});

  // ── Existing clips reuse (new) ─────────────────────────────────────────────
  const [priorRunId, setPriorRunId] = useState<number | null>(null);
  const [skipRegenerate, setSkipRegenerate] = useState<boolean>(false);

  // ── Upload options ─────────────────────────────────────────────────────────
  const [uploadPrivacy, setUploadPrivacy] = useState<'private' | 'unlisted' | 'public'>('private');
  const [uploadScheduleEnabled, setUploadScheduleEnabled] = useState<boolean>(false);
  const [uploadAutoEnabled, setUploadAutoEnabled] = useState<boolean>(false);
  const [uploadDryRun, setUploadDryRun] = useState<boolean>(false);

  // ── Captions state ─────────────────────────────────────────────────────────
  const [captionsEnabled, setCaptionsEnabled] = useState<boolean>(false);
  const [captionsPreset, setCaptionsPreset] = useState<CaptionsConfig['preset']>('simple');
  const [captionsFontFamily, setCaptionsFontFamily] = useState<string>('');
  const [captionsBold, setCaptionsBold] = useState<boolean>(false);
  const [captionsItalic, setCaptionsItalic] = useState<boolean>(false);
  const [captionsUppercase, setCaptionsUppercase] = useState<boolean>(false);
  const [captionsFontSize, setCaptionsFontSize] = useState<number>(32);
  const [captionsTextColor, setCaptionsTextColor] = useState<string>('#ffffff');
  const [captionsBgEnabled, setCaptionsBgEnabled] = useState<boolean>(false);
  const [captionsOutlineEnabled, setCaptionsOutlineEnabled] = useState<boolean>(false);
  const [captionsOutlineColor, setCaptionsOutlineColor] = useState<string>('#000000');
  const [captionsOutlineWidth, setCaptionsOutlineWidth] = useState<number>(2);
  const [captionsShadowEnabled, setCaptionsShadowEnabled] = useState<boolean>(false);
  const [captionsShadowColor, setCaptionsShadowColor] = useState<string>('#000000');
  const [captionsShadowDistance, setCaptionsShadowDistance] = useState<number>(4);
  const [captionsVerticalOffset, setCaptionsVerticalOffset] = useState<number>(0);

  // ── Derived validation (computed — not useState) ───────────────────────────
  const isMinDurationInvalid = selectedMode === 'ai' && minDurationSecs >= clipDurationSecs;

  // ── Loading / submit state ─────────────────────────────────────────────────
  const [isLoadingPreference, setIsLoadingPreference] = useState<boolean>(true);
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

  // ── Mount effect: historical pre-fill or fetch preference ──────────────────
  useEffect(() => {
    console.log('[StepMode] mount — isHistorical:', isHistorical);

    // Priority 1: historical back-navigation pre-fill
    if (historical?.kind === 'mode' && historical.mode_config) {
      const cfg = historical.mode_config;
      console.log('[StepMode] pre-filling from historical mode_config', cfg);
      setSelectedMode(cfg.mode);
      if (cfg.sensitivity_pct != null) setSensitivityPct(cfg.sensitivity_pct);
      if (cfg.skip_music_mins !== undefined) setSkipMusicMins(cfg.skip_music_mins);
      if (cfg.force_regenerate != null) setForceRegenerate(cfg.force_regenerate);
      if (cfg.skip_transcription != null) setSkipTranscription(cfg.skip_transcription);
      if (cfg.segment_secs != null) setSegmentSecs(cfg.segment_secs);
      setMinPartSecs(cfg.min_part_secs ?? 0);
      if (cfg.max_duration_secs != null && cfg.max_duration_secs > 0) setClipDurationSecs(cfg.max_duration_secs);
      if (cfg.min_duration_secs != null) setMinDurationSecs(cfg.min_duration_secs);
      if (cfg.anti_duplicate) {
        setAntiDupEnabled(cfg.anti_duplicate.enabled);
        setAntiDupMode(cfg.anti_duplicate.mode);
        if (cfg.anti_duplicate?.effects) setAntiDupEffects(cfg.anti_duplicate.effects);
      }
      if (cfg.skip_regenerate != null) setSkipRegenerate(cfg.skip_regenerate);
      if (cfg.upload_options) {
        setUploadPrivacy(cfg.upload_options.privacy);
        setUploadScheduleEnabled(cfg.upload_options.schedule_enabled);
        setUploadAutoEnabled(cfg.upload_options.auto_enabled);
        setUploadDryRun(cfg.upload_options.dry_run);
      }
      if (cfg.captions) {
        setCaptionsEnabled(cfg.captions.enabled);
        setCaptionsPreset(cfg.captions.preset);
        if (cfg.captions.font_family != null) setCaptionsFontFamily(cfg.captions.font_family);
        if (cfg.captions.bold != null) setCaptionsBold(cfg.captions.bold);
        if (cfg.captions.italic != null) setCaptionsItalic(cfg.captions.italic);
        if (cfg.captions.uppercase != null) setCaptionsUppercase(cfg.captions.uppercase);
        if (cfg.captions.font_size != null) setCaptionsFontSize(cfg.captions.font_size);
        if (cfg.captions.text_color != null) setCaptionsTextColor(cfg.captions.text_color);
        if (cfg.captions.bg_enabled != null) setCaptionsBgEnabled(cfg.captions.bg_enabled);
        if (cfg.captions.outline_enabled != null) setCaptionsOutlineEnabled(cfg.captions.outline_enabled);
        if (cfg.captions.outline_color != null) setCaptionsOutlineColor(cfg.captions.outline_color);
        if (cfg.captions.outline_width != null) setCaptionsOutlineWidth(cfg.captions.outline_width);
        if (cfg.captions.shadow_enabled != null) setCaptionsShadowEnabled(cfg.captions.shadow_enabled);
        if (cfg.captions.shadow_color != null) setCaptionsShadowColor(cfg.captions.shadow_color);
        if (cfg.captions.shadow_distance != null) setCaptionsShadowDistance(cfg.captions.shadow_distance);
        if (cfg.captions.vertical_offset != null) setCaptionsVerticalOffset(cfg.captions.vertical_offset);
      }
      setIsLoadingPreference(false);
      return;
    }

    // Priority 2: fetch persisted preference from settings API
    const fetchPreference = async () => {
      try {
        console.log('[StepMode] fetching settings from', goUrl);
        const res = await fetch(`${goUrl}/api/settings`);
        const data = (await res.json()) as Array<{ key: string; value: string }>;

        // Helper functions to parse settings values
        const parseIntKey = (key: string, fallback: number): number => {
          const v = data.find((s) => s.key === key)?.value;
          return v ? (parseInt(v, 10) || fallback) : fallback;
        };
        const parseBoolKey = (key: string, fallback: boolean): boolean => {
          const v = data.find((s) => s.key === key)?.value;
          return v !== undefined ? v === 'true' : fallback;
        };

        const entry = data.find((s) => s.key === 'workflow_mode_default');
        const mode: WorkflowMode = entry?.value === 'ai' ? 'ai' : 'longform';
        console.log('[StepMode] preference loaded:', mode);
        setSelectedMode(mode);

        setClipDurationSecs(parseIntKey('ai_clip_duration_secs', 1200));
        setMinDurationSecs(parseIntKey('ai_min_duration_secs', 480));
        setSensitivityPct(parseIntKey('ai_sensitivity_pct', 70));
        const skipMusicVal = data.find((s) => s.key === 'ai_skip_music_mins')?.value;
        setSkipMusicMins(skipMusicVal && skipMusicVal !== '' ? parseFloat(skipMusicVal) : null);
        setForceRegenerate(parseBoolKey('ai_force_regenerate', true));
        setSkipTranscription(parseBoolKey('ai_skip_transcription', false));
        setSegmentSecs(parseIntKey('longform_segment_secs', 600));
        setMinPartSecs(parseIntKey('longform_min_part_secs', 0));
        setAntiDupEnabled(parseBoolKey('anti_dup_enabled', false));
        const adModeVal = data.find((s) => s.key === 'anti_dup_mode')?.value;
        setAntiDupMode(adModeVal === 'aggressive' ? 'aggressive' : 'subtle');
        setAntiDupEffects({
          speed_boost: parseBoolKey('anti_dup_effect_speed_boost', false),
          crop: parseBoolKey('anti_dup_effect_crop', false),
          color_grading: parseBoolKey('anti_dup_effect_color_grading', false),
          noise: parseBoolKey('anti_dup_effect_noise', false),
          noise_strength: parseIntKey('anti_dup_effect_noise_strength', 3),
          blur: parseBoolKey('anti_dup_effect_blur', false),
          blur_edge_pct: parseIntKey('anti_dup_effect_blur_edge_pct', 10),
          zoom: parseBoolKey('anti_dup_effect_zoom', false),
          transitions: parseBoolKey('anti_dup_effect_transitions', false),
        });
        const privacyVal = data.find((s) => s.key === 'upload_privacy')?.value;
        setUploadPrivacy(privacyVal === 'unlisted' ? 'unlisted' : privacyVal === 'public' ? 'public' : 'private');
        setUploadScheduleEnabled(parseBoolKey('upload_schedule_enabled', false));
        setUploadAutoEnabled(parseBoolKey('upload_auto_enabled', false));
        setUploadDryRun(parseBoolKey('upload_dry_run', false));
        setCaptionsEnabled(parseBoolKey('captions_enabled', false));
        const captionsPresetVal = data.find((s) => s.key === 'captions_preset')?.value;
        setCaptionsPreset(captionsPresetVal === 'bold' ? 'bold' : captionsPresetVal === 'word_by_word' ? 'word_by_word' : 'simple');
        setCaptionsFontFamily(data.find((s) => s.key === 'captions_font_family')?.value ?? '');
        setCaptionsBold(parseBoolKey('captions_bold', false));
        setCaptionsItalic(parseBoolKey('captions_italic', false));
        setCaptionsUppercase(parseBoolKey('captions_uppercase', false));
        setCaptionsFontSize(parseIntKey('captions_font_size', 32));
        setCaptionsTextColor(data.find((s) => s.key === 'captions_text_color')?.value ?? '#ffffff');
        setCaptionsBgEnabled(parseBoolKey('captions_bg_enabled', false));
        setCaptionsOutlineEnabled(parseBoolKey('captions_outline_enabled', false));
        setCaptionsOutlineColor(data.find((s) => s.key === 'captions_outline_color')?.value ?? '#000000');
        setCaptionsOutlineWidth(parseIntKey('captions_outline_width', 2));
        setCaptionsShadowEnabled(parseBoolKey('captions_shadow_enabled', false));
        setCaptionsShadowColor(data.find((s) => s.key === 'captions_shadow_color')?.value ?? '#000000');
        setCaptionsShadowDistance(parseIntKey('captions_shadow_distance', 4));
        setCaptionsVerticalOffset(parseIntKey('captions_vertical_offset', 0));
      } catch (err) {
        console.warn('[StepMode] failed to load preference, defaulting to longform', err);
        setSelectedMode('longform');
      } finally {
        setIsLoadingPreference(false);
      }

      // After preferences are loaded, check for prior clips (non-fatal)
      if (run?.url) {
        try {
          console.log('[StepMode] checking for prior clips for url:', run.url);
          const prRes = await fetch(`${goUrl}/api/pipeline/runs/prior-clips?url=${encodeURIComponent(run.url)}`);
          if (prRes.ok) {
            const prData = (await prRes.json()) as PriorClipsResponse;
            if (prData.exists && prData.run_id != null) {
              console.log('[StepMode] prior clips found, run_id:', prData.run_id);
              setPriorRunId(prData.run_id);
            }
          }
        } catch {
          console.warn('[StepMode] prior clips check failed, continuing');
        }
      }
    };

    void fetchPreference();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // ── setEffect helper ───────────────────────────────────────────────────────
  const setEffect = (key: keyof AntiDupEffects, value: boolean | number) => {
    setAntiDupEffects((prev) => ({ ...prev, [key]: value }));
  };

  // ── buildModeConfig ────────────────────────────────────────────────────────
  function buildModeConfig(): ModeConfig {
    const captionsCfg: CaptionsConfig | undefined = captionsEnabled ? {
      enabled: true,
      preset: captionsPreset,
      font_family: captionsFontFamily || undefined,
      bold: captionsBold,
      italic: captionsItalic,
      uppercase: captionsUppercase,
      font_size: captionsFontSize,
      text_color: captionsTextColor,
      bg_enabled: captionsBgEnabled,
      outline_enabled: captionsOutlineEnabled,
      outline_color: captionsOutlineColor,
      outline_width: captionsOutlineWidth,
      shadow_enabled: captionsShadowEnabled,
      shadow_color: captionsShadowColor,
      shadow_distance: captionsShadowDistance,
      vertical_offset: captionsVerticalOffset,
    } : undefined;
    const antiDup: AntiDuplicateConfig = {
      enabled: antiDupEnabled,
      mode: antiDupMode,
      effects: antiDupEnabled ? antiDupEffects : undefined,
    };
    if (selectedMode === 'ai') {
      return {
        mode: 'ai',
        ...AI_DEFAULTS,
        min_duration_secs: minDurationSecs,
        max_duration_secs: clipDurationSecs,
        sensitivity_pct: sensitivityPct,
        skip_music_mins: skipMusicMins,
        force_regenerate: forceRegenerate,
        skip_transcription: skipTranscription,
        anti_duplicate: antiDup,
        skip_regenerate: skipRegenerate,
        upload_options: { privacy: uploadPrivacy, schedule_enabled: uploadScheduleEnabled, auto_enabled: uploadAutoEnabled, dry_run: uploadDryRun },
        captions: captionsCfg,
      };
    }
    return {
      mode: 'longform',
      ...LONGFORM_DEFAULTS,
      segment_secs: segmentSecs,
      min_part_secs: minPartSecs > 0 ? minPartSecs : undefined,
      anti_duplicate: antiDup,
      skip_regenerate: skipRegenerate,
      upload_options: { privacy: uploadPrivacy, schedule_enabled: uploadScheduleEnabled, auto_enabled: uploadAutoEnabled, dry_run: uploadDryRun },
      captions: captionsCfg,
    };
  }

  // ── Submit handler ─────────────────────────────────────────────────────────
  const handleSubmit = async () => {
    if (!activeRunId || isSubmitting) return;
    setIsSubmitting(true);
    console.log('[StepMode] submitting mode:', selectedMode);

    // Persist all preferences (non-fatal)
    try {
      const settingPairs = [
        { key: 'workflow_mode_default', value: selectedMode },
        { key: 'ai_clip_duration_secs', value: String(clipDurationSecs) },
        { key: 'ai_min_duration_secs', value: String(minDurationSecs) },
        { key: 'ai_sensitivity_pct', value: String(sensitivityPct) },
        { key: 'ai_skip_music_mins', value: skipMusicMins != null ? String(skipMusicMins) : '' },
        { key: 'ai_force_regenerate', value: String(forceRegenerate) },
        { key: 'ai_skip_transcription', value: String(skipTranscription) },
        { key: 'longform_segment_secs', value: String(segmentSecs) },
        { key: 'longform_min_part_secs', value: String(minPartSecs) },
        { key: 'anti_dup_enabled', value: String(antiDupEnabled) },
        { key: 'anti_dup_mode', value: antiDupMode },
        { key: 'anti_dup_effect_speed_boost', value: String(antiDupEffects.speed_boost ?? false) },
        { key: 'anti_dup_effect_crop', value: String(antiDupEffects.crop ?? false) },
        { key: 'anti_dup_effect_color_grading', value: String(antiDupEffects.color_grading ?? false) },
        { key: 'anti_dup_effect_noise', value: String(antiDupEffects.noise ?? false) },
        { key: 'anti_dup_effect_noise_strength', value: String(antiDupEffects.noise_strength ?? 3) },
        { key: 'anti_dup_effect_blur', value: String(antiDupEffects.blur ?? false) },
        { key: 'anti_dup_effect_blur_edge_pct', value: String(antiDupEffects.blur_edge_pct ?? 10) },
        { key: 'anti_dup_effect_zoom', value: String(antiDupEffects.zoom ?? false) },
        { key: 'anti_dup_effect_transitions', value: String(antiDupEffects.transitions ?? false) },
        { key: 'upload_privacy', value: uploadPrivacy },
        { key: 'upload_schedule_enabled', value: String(uploadScheduleEnabled) },
        { key: 'upload_auto_enabled', value: String(uploadAutoEnabled) },
        { key: 'upload_dry_run', value: String(uploadDryRun) },
        { key: 'captions_enabled', value: String(captionsEnabled) },
        { key: 'captions_preset', value: captionsPreset },
        { key: 'captions_font_family', value: captionsFontFamily },
        { key: 'captions_bold', value: String(captionsBold) },
        { key: 'captions_italic', value: String(captionsItalic) },
        { key: 'captions_uppercase', value: String(captionsUppercase) },
        { key: 'captions_font_size', value: String(captionsFontSize) },
        { key: 'captions_text_color', value: captionsTextColor },
        { key: 'captions_bg_enabled', value: String(captionsBgEnabled) },
        { key: 'captions_outline_enabled', value: String(captionsOutlineEnabled) },
        { key: 'captions_outline_color', value: captionsOutlineColor },
        { key: 'captions_outline_width', value: String(captionsOutlineWidth) },
        { key: 'captions_shadow_enabled', value: String(captionsShadowEnabled) },
        { key: 'captions_shadow_color', value: captionsShadowColor },
        { key: 'captions_shadow_distance', value: String(captionsShadowDistance) },
        { key: 'captions_vertical_offset', value: String(captionsVerticalOffset) },
      ];
      await Promise.allSettled(
        settingPairs.map(({ key, value }) =>
          fetch(`${goUrl}/api/settings`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ key, value }),
          })
        )
      );
      console.log('[StepMode] preferences persisted');
    } catch (err) {
      console.warn('[StepMode] failed to persist preferences', err);
    }

    // Advance the pipeline gate
    const cfg = buildModeConfig();
    console.log('[StepMode] calling advance with mode_config', cfg);
    await advance(goUrl, activeRunId, { mode_config: cfg });
  };

  const isSubmitDisabled = selectedMode === null || isSubmitting || isLoadingPreference || isMinDurationInvalid;

  return (
    <div className="space-y-6 max-w-lg">
      <div className="space-y-1">
        <h2 className="text-xl font-semibold text-foreground">Workflow Mode</h2>
        <p className="text-sm text-zinc-500">Choose how to process this video.</p>
      </div>

      {isHistorical && (
        <div className="rounded-md border border-zinc-700 bg-zinc-900 px-4 py-3 text-xs text-zinc-400">
          Reviewing previous submission. Re-submitting will restart the pipeline from this step.
        </div>
      )}

      {/* Video Info Card */}
      {videoInfo && (
        <div className="flex items-center gap-3 rounded-lg border border-zinc-700 bg-zinc-900 p-3">
          {videoInfo.thumbnailUrl && (
            <img
              src={videoInfo.thumbnailUrl}
              alt=""
              className="h-14 w-24 rounded object-cover flex-shrink-0"
            />
          )}
          <div className="min-w-0 flex-1">
            <p className="text-xs font-medium text-zinc-200 truncate">{videoInfo.title}</p>
            {videoInfo.channelName && (
              <p className="text-xs text-zinc-500 truncate">{videoInfo.channelName}</p>
            )}
            <p className="text-xs text-zinc-500">
              {(() => {
                const h = Math.floor(videoInfo.durationSec / 3600);
                const m = Math.floor((videoInfo.durationSec % 3600) / 60);
                const s = videoInfo.durationSec % 60;
                return h > 0
                  ? `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
                  : `${m}:${String(s).padStart(2, '0')}`;
              })()}
            </p>
          </div>
        </div>
      )}

      {/* Reuse Existing Clips — only shown when prior run exists */}
      {priorRunId !== null && (
        <div className="rounded-md border border-zinc-700 bg-zinc-900 px-4 py-3">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs font-medium text-zinc-300">Reuse Existing Clips</p>
              <p className="text-xs text-zinc-500">From previous run #{priorRunId}</p>
            </div>
            <button
              onClick={() => {
                console.log('[StepMode] skip_regenerate toggled:', !skipRegenerate);
                setSkipRegenerate((v) => !v);
              }}
              className={[
                'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
                skipRegenerate ? 'bg-zinc-300' : 'bg-zinc-700',
              ].join(' ')}
              role="switch"
              aria-checked={skipRegenerate}
            >
              <span
                className={[
                  'inline-block h-3.5 w-3.5 transform rounded-full bg-zinc-900 transition-transform',
                  skipRegenerate ? 'translate-x-4' : 'translate-x-1',
                ].join(' ')}
              />
            </button>
          </div>
        </div>
      )}

      {/* Mode cards */}
      <div className="grid grid-cols-2 gap-3">
        {/* AI Highlights card */}
        <button
          onClick={() => {
            console.log('[StepMode] mode selected: ai');
            setSelectedMode('ai');
          }}
          className={[
            'relative flex flex-col gap-2 rounded-lg border p-4 text-left transition-all',
            selectedMode === 'ai'
              ? 'border-zinc-300 bg-zinc-800 ring-1 ring-zinc-300'
              : 'border-zinc-700 bg-zinc-900 hover:border-zinc-500 hover:bg-zinc-800',
          ].join(' ')}
        >
          {selectedMode === 'ai' && (
            <span className="absolute top-3 right-3 flex h-4 w-4 items-center justify-center rounded-full bg-zinc-300 text-zinc-900 text-[10px] font-bold">✓</span>
          )}
          <span className="text-sm font-semibold text-foreground">AI Highlights</span>
          <span className="text-xs text-zinc-400">Extract the best moments from your video automatically.</span>
          <ul className="space-y-0.5 text-xs text-zinc-500">
            <li>• AI-powered highlight detection</li>
            <li>• Adjustable sensitivity threshold</li>
            <li>• Skip music intros</li>
            <li>• Force re-analysis control</li>
          </ul>
        </button>

        {/* Long Form card */}
        <button
          onClick={() => {
            console.log('[StepMode] mode selected: longform');
            setSelectedMode('longform');
          }}
          className={[
            'relative flex flex-col gap-2 rounded-lg border p-4 text-left transition-all',
            selectedMode === 'longform'
              ? 'border-zinc-300 bg-zinc-800 ring-1 ring-zinc-300'
              : 'border-zinc-700 bg-zinc-900 hover:border-zinc-500 hover:bg-zinc-800',
          ].join(' ')}
        >
          {selectedMode === 'longform' && (
            <span className="absolute top-3 right-3 flex h-4 w-4 items-center justify-center rounded-full bg-zinc-300 text-zinc-900 text-[10px] font-bold">✓</span>
          )}
          <span className="text-sm font-semibold text-foreground">Long Form</span>
          <span className="text-xs text-zinc-400">Split the full video into equal-length segments.</span>
          <ul className="space-y-0.5 text-xs text-zinc-500">
            <li>• Equal-length part splitting</li>
            <li>• Configurable segment duration</li>
            <li>• No content filtering</li>
            <li>• Preserves full video</li>
          </ul>
        </button>
      </div>

      {/* AI configuration section */}
      {selectedMode === 'ai' && (
        <div className="space-y-4 rounded-lg border border-zinc-700 bg-zinc-900 p-4">
          <p className="text-xs font-medium text-zinc-300 uppercase tracking-wider">AI Configuration</p>

          {/* Target Clip Duration */}
          <div className="space-y-1.5">
            <label className="text-xs text-zinc-400">Target Clip Duration</label>
            <div className="flex gap-2">
              {([900, 1200, 1800] as const).map((secs) => (
                <button
                  key={secs}
                  onClick={() => setClipDurationSecs(secs)}
                  className={[
                    'flex-1 rounded px-2 py-1.5 text-xs transition-colors',
                    clipDurationSecs === secs
                      ? 'bg-zinc-300 text-zinc-900 font-medium'
                      : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700',
                  ].join(' ')}
                >
                  {secs === 900 ? '15 min' : secs === 1200 ? '20 min' : '30 min'}
                </button>
              ))}
            </div>
          </div>

          {/* Minimum Clip Duration */}
          <div className="space-y-1.5">
            <label className="text-xs text-zinc-400">Minimum Clip Duration</label>
            <div className="flex gap-2">
              {([240, 360, 480, 600] as const).map((secs) => (
                <button
                  key={secs}
                  onClick={() => setMinDurationSecs(secs)}
                  className={[
                    'flex-1 rounded px-2 py-1.5 text-xs transition-colors',
                    minDurationSecs === secs
                      ? 'bg-zinc-300 text-zinc-900 font-medium'
                      : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700',
                  ].join(' ')}
                >
                  {secs / 60} min
                </button>
              ))}
            </div>
            {isMinDurationInvalid && (
              <p className="text-xs text-red-400">Min duration must be less than target duration</p>
            )}
          </div>

          {/* Sensitivity threshold presets */}
          <div className="space-y-1.5">
            <label className="text-xs text-zinc-400">Highlight Threshold</label>
            <div className="flex gap-2">
              {([50, 70, 90] as const).map((pct) => (
                <button
                  key={pct}
                  onClick={() => setSensitivityPct(pct)}
                  className={[
                    'flex-1 rounded px-2 py-1.5 text-xs transition-colors',
                    sensitivityPct === pct
                      ? 'bg-zinc-300 text-zinc-900 font-medium'
                      : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700',
                  ].join(' ')}
                >
                  {pct === 50 ? 'Liberal' : pct === 70 ? 'Balanced' : 'Selective'}
                  <span className="ml-1 opacity-60">({pct})</span>
                </button>
              ))}
            </div>
          </div>

          {/* Skip music dropdown */}
          <div className="space-y-1.5">
            <label className="text-xs text-zinc-400">Skip Music Intro</label>
            <select
              value={skipMusicMins ?? ''}
              onChange={(e) => setSkipMusicMins(e.target.value === '' ? null : Number(e.target.value))}
              className="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-zinc-500"
            >
              <option value="">None</option>
              <option value="1">1 min</option>
              <option value="2">2 min</option>
              <option value="3">3 min</option>
              <option value="5">5 min</option>
            </select>
          </div>

          {/* Force regenerate toggle */}
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs text-zinc-300">Force Re-analyze</p>
              <p className="text-xs text-zinc-600">Ignore cached transcript and analysis</p>
            </div>
            <button
              onClick={() => setForceRegenerate((v) => !v)}
              className={[
                'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
                forceRegenerate ? 'bg-zinc-300' : 'bg-zinc-700',
              ].join(' ')}
              role="switch"
              aria-checked={forceRegenerate}
            >
              <span
                className={[
                  'inline-block h-3.5 w-3.5 transform rounded-full bg-zinc-900 transition-transform',
                  forceRegenerate ? 'translate-x-4' : 'translate-x-1',
                ].join(' ')}
              />
            </button>
          </div>

          {/* Skip transcription toggle */}
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs text-zinc-300">Skip Transcription</p>
              <p className="text-xs text-zinc-600">Use existing transcript if available</p>
            </div>
            <button
              onClick={() => setSkipTranscription((v) => !v)}
              className={[
                'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
                skipTranscription ? 'bg-zinc-300' : 'bg-zinc-700',
              ].join(' ')}
              role="switch"
              aria-checked={skipTranscription}
            >
              <span
                className={[
                  'inline-block h-3.5 w-3.5 transform rounded-full bg-zinc-900 transition-transform',
                  skipTranscription ? 'translate-x-4' : 'translate-x-1',
                ].join(' ')}
              />
            </button>
          </div>
        </div>
      )}

      {/* Long Form configuration section */}
      {selectedMode === 'longform' && (
        <div className="space-y-4 rounded-lg border border-zinc-700 bg-zinc-900 p-4">
          <p className="text-xs font-medium text-zinc-300 uppercase tracking-wider">Long Form Configuration</p>

          {/* Segment duration presets */}
          <div className="space-y-1.5">
            <label className="text-xs text-zinc-400">Segment Duration</label>
            <div className="flex gap-2">
              {([
                [1020, '17 min'],
                [1560, '26 min'],
                [3120, '52 min'],
              ] as [number, string][]).map(([secs, label]) => (
                <button
                  key={secs}
                  onClick={() => setSegmentSecs(secs)}
                  className={[
                    'rounded px-2 py-1.5 text-xs transition-colors',
                    segmentSecs === secs
                      ? 'bg-zinc-300 text-zinc-900 font-medium'
                      : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700',
                  ].join(' ')}
                >
                  {label}
                </button>
              ))}
            </div>
          </div>

          {/* Min Part Duration */}
          <div className="space-y-1.5">
            <label className="text-xs text-zinc-400">
              Min Part Duration{' '}
              <span className="text-zinc-600">(ignore parts shorter than this)</span>
            </label>
            <div className="flex flex-wrap gap-2">
              {([
                [0, 'Off'],
                [60, '1 min'],
                [120, '2 min'],
                [300, '5 min'],
                [480, '8 min'],
                [600, '10 min'],
                [900, '15 min'],
              ] as [number, string][]).map(([secs, label]) => (
                <button
                  key={secs}
                  onClick={() => setMinPartSecs(secs)}
                  className={[
                    'rounded px-2 py-1.5 text-xs transition-colors',
                    minPartSecs === secs
                      ? 'bg-zinc-300 text-zinc-900 font-medium'
                      : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700',
                  ].join(' ')}
                >
                  {label}
                </button>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Anti-Duplicate Protection */}
      <div className="space-y-3 rounded-lg border border-zinc-700 bg-zinc-900 p-4">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-xs font-medium text-zinc-300 uppercase tracking-wider">Anti-Duplicate Protection</p>
            <p className="text-xs text-zinc-500">Prevent YouTube duplicate detection</p>
          </div>
          <button
            onClick={() => {
              console.log('[StepMode] anti-dup toggled:', !antiDupEnabled);
              setAntiDupEnabled((v) => !v);
            }}
            className={[
              'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
              antiDupEnabled ? 'bg-zinc-300' : 'bg-zinc-700',
            ].join(' ')}
            role="switch"
            aria-checked={antiDupEnabled}
          >
            <span
              className={[
                'inline-block h-3.5 w-3.5 transform rounded-full bg-zinc-900 transition-transform',
                antiDupEnabled ? 'translate-x-4' : 'translate-x-1',
              ].join(' ')}
            />
          </button>
        </div>

        {antiDupEnabled && (
          <>
            <div className="grid grid-cols-2 gap-2">
              {(['subtle', 'aggressive'] as const).map((m) => (
                <button
                  key={m}
                  onClick={() => {
                    console.log('[StepMode] anti-dup mode selected:', m);
                    setAntiDupMode(m);
                  }}
                  className={[
                    'flex flex-col items-start rounded-lg border p-3 text-left transition-all',
                    antiDupMode === m
                      ? 'border-zinc-300 bg-zinc-800 ring-1 ring-zinc-300'
                      : 'border-zinc-700 bg-zinc-800 hover:border-zinc-500',
                  ].join(' ')}
                >
                  <span className="text-xs font-medium text-zinc-300 capitalize">{m}</span>
                  <span className="text-xs text-zinc-500">
                    {m === 'subtle' ? '1–2% changes' : '5–10% changes'}
                  </span>
                </button>
              ))}
            </div>

            <div className="space-y-2 pt-1">
              <p className="text-xs text-zinc-500">Visual Effects</p>

              {/* Simple checkboxes */}
              {([
                ['speed_boost', 'Speed +2%'],
                ['crop', 'Crop subtle 2%'],
                ['color_grading', 'Color grading'],
                ['zoom', 'Static zoom 3%'],
                ['transitions', 'Transitions 2×'],
              ] as [keyof AntiDupEffects, string][]).map(([key, label]) => (
                <label key={key} className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={!!antiDupEffects[key]}
                    onChange={(e) => setEffect(key, e.target.checked)}
                    className="rounded border-zinc-600 bg-zinc-800 text-zinc-300 focus:ring-zinc-500"
                  />
                  <span className="text-xs text-zinc-300">{label}</span>
                </label>
              ))}

              {/* Noise with strength slider */}
              <div className="space-y-1">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={!!antiDupEffects.noise}
                    onChange={(e) => setEffect('noise', e.target.checked)}
                    className="rounded border-zinc-600 bg-zinc-800 text-zinc-300 focus:ring-zinc-500"
                  />
                  <span className="text-xs text-zinc-300">Noise/Grain</span>
                </label>
                {antiDupEffects.noise && (
                  <div className="ml-6 flex items-center gap-2">
                    <span className="text-xs text-zinc-500 w-16">Strength</span>
                    <input
                      type="range"
                      min={1}
                      max={10}
                      value={antiDupEffects.noise_strength ?? 3}
                      onChange={(e) => setEffect('noise_strength', Number(e.target.value))}
                      className="flex-1 accent-zinc-300"
                    />
                    <span className="text-xs text-zinc-400 w-4">{antiDupEffects.noise_strength ?? 3}</span>
                  </div>
                )}
              </div>

              {/* Blur with edge % slider */}
              <div className="space-y-1">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={!!antiDupEffects.blur}
                    onChange={(e) => setEffect('blur', e.target.checked)}
                    className="rounded border-zinc-600 bg-zinc-800 text-zinc-300 focus:ring-zinc-500"
                  />
                  <span className="text-xs text-zinc-300">Background Blur</span>
                </label>
                {antiDupEffects.blur && (
                  <div className="ml-6 flex items-center gap-2">
                    <span className="text-xs text-zinc-500 w-16">Edge %</span>
                    <input
                      type="range"
                      min={0}
                      max={100}
                      value={antiDupEffects.blur_edge_pct ?? 10}
                      onChange={(e) => setEffect('blur_edge_pct', Number(e.target.value))}
                      className="flex-1 accent-zinc-300"
                    />
                    <span className="text-xs text-zinc-400 w-8">{antiDupEffects.blur_edge_pct ?? 10}%</span>
                  </div>
                )}
              </div>
            </div>
          </>
        )}
      </div>

      {/* Captions */}
      <div className="space-y-4 rounded-lg border border-zinc-700 bg-zinc-900 p-4">
        <div className="flex items-center justify-between">
          <p className="text-xs font-medium text-zinc-300 uppercase tracking-wider">Legendas</p>
          <button
            onClick={() => setCaptionsEnabled((v) => !v)}
            className={[
              'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
              captionsEnabled ? 'bg-zinc-300' : 'bg-zinc-700',
            ].join(' ')}
            role="switch"
            aria-checked={captionsEnabled}
          >
            <span
              className={[
                'inline-block h-3.5 w-3.5 transform rounded-full bg-zinc-900 transition-transform',
                captionsEnabled ? 'translate-x-4' : 'translate-x-1',
              ].join(' ')}
            />
          </button>
        </div>

        {captionsEnabled && (
          <div className="space-y-4">
            {/* Preset selector */}
            <div className="space-y-1.5">
              <label className="text-xs text-zinc-400">Preset</label>
              <div className="flex gap-2">
                {([
                  ['simple', 'Simples'],
                  ['bold', 'Negrito'],
                  ['word_by_word', 'Palavra a Palavra'],
                ] as [CaptionsConfig['preset'], string][]).map(([p, label]) => (
                  <button
                    key={p}
                    onClick={() => setCaptionsPreset(p)}
                    className={[
                      'flex-1 rounded px-2 py-1.5 text-xs transition-colors',
                      captionsPreset === p
                        ? 'bg-zinc-300 text-zinc-900 font-medium'
                        : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700',
                    ].join(' ')}
                  >
                    {label}
                  </button>
                ))}
              </div>
            </div>

            {/* Font customization */}
            <div className="space-y-3">
              <p className="text-xs text-zinc-500">Personalizar</p>

              <div className="space-y-1.5">
                <label className="text-xs text-zinc-400">Fonte</label>
                <input
                  type="text"
                  value={captionsFontFamily}
                  onChange={(e) => setCaptionsFontFamily(e.target.value)}
                  placeholder="Arial, Roboto, …"
                  className="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-xs text-foreground placeholder-zinc-600 focus:outline-none focus:ring-1 focus:ring-zinc-500"
                />
              </div>

              <div className="flex gap-2">
                {([
                  ['bold', captionsBold, setCaptionsBold, 'Negrito'],
                  ['italic', captionsItalic, setCaptionsItalic, 'Itálico'],
                  ['uppercase', captionsUppercase, setCaptionsUppercase, 'Maiúsculas'],
                ] as [string, boolean, (v: boolean) => void, string][]).map(([key, val, setter, label]) => (
                  <button
                    key={key}
                    onClick={() => setter(!val)}
                    className={[
                      'flex-1 rounded px-2 py-1.5 text-xs transition-colors',
                      val ? 'bg-zinc-300 text-zinc-900 font-medium' : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700',
                    ].join(' ')}
                  >
                    {label}
                  </button>
                ))}
              </div>

              <div className="flex items-center gap-3">
                <label className="text-xs text-zinc-400 w-20 flex-shrink-0">Tamanho</label>
                <input
                  type="range"
                  min={12}
                  max={96}
                  value={captionsFontSize}
                  onChange={(e) => setCaptionsFontSize(Number(e.target.value))}
                  className="flex-1 accent-zinc-300"
                />
                <span className="text-xs text-zinc-400 w-8 text-right">{captionsFontSize}px</span>
              </div>
            </div>

            {/* Colors */}
            <div className="space-y-3">
              <p className="text-xs text-zinc-500">Cores</p>

              <div className="flex items-center gap-3">
                <label className="text-xs text-zinc-400 w-20 flex-shrink-0">Texto</label>
                <input
                  type="color"
                  value={captionsTextColor}
                  onChange={(e) => setCaptionsTextColor(e.target.value)}
                  className="h-7 w-12 cursor-pointer rounded border border-zinc-700 bg-zinc-800"
                />
                <span className="text-xs text-zinc-500">{captionsTextColor}</span>
              </div>

              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={captionsBgEnabled}
                  onChange={(e) => setCaptionsBgEnabled(e.target.checked)}
                  className="rounded border-zinc-600 bg-zinc-800 text-zinc-300"
                />
                <span className="text-xs text-zinc-300">Fundo</span>
              </label>

              {/* Outline */}
              <div className="space-y-2">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={captionsOutlineEnabled}
                    onChange={(e) => setCaptionsOutlineEnabled(e.target.checked)}
                    className="rounded border-zinc-600 bg-zinc-800 text-zinc-300"
                  />
                  <span className="text-xs text-zinc-300">Contorno</span>
                </label>
                {captionsOutlineEnabled && (
                  <div className="ml-6 space-y-2">
                    <div className="flex items-center gap-3">
                      <label className="text-xs text-zinc-400 w-16 flex-shrink-0">Cor</label>
                      <input
                        type="color"
                        value={captionsOutlineColor}
                        onChange={(e) => setCaptionsOutlineColor(e.target.value)}
                        className="h-7 w-12 cursor-pointer rounded border border-zinc-700 bg-zinc-800"
                      />
                    </div>
                    <div className="flex items-center gap-3">
                      <label className="text-xs text-zinc-400 w-16 flex-shrink-0">Espessura</label>
                      <input
                        type="range"
                        min={1}
                        max={10}
                        value={captionsOutlineWidth}
                        onChange={(e) => setCaptionsOutlineWidth(Number(e.target.value))}
                        className="flex-1 accent-zinc-300"
                      />
                      <span className="text-xs text-zinc-400 w-4">{captionsOutlineWidth}</span>
                    </div>
                  </div>
                )}
              </div>

              {/* Shadow */}
              <div className="space-y-2">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={captionsShadowEnabled}
                    onChange={(e) => setCaptionsShadowEnabled(e.target.checked)}
                    className="rounded border-zinc-600 bg-zinc-800 text-zinc-300"
                  />
                  <span className="text-xs text-zinc-300">Sombra</span>
                </label>
                {captionsShadowEnabled && (
                  <div className="ml-6 space-y-2">
                    <div className="flex items-center gap-3">
                      <label className="text-xs text-zinc-400 w-16 flex-shrink-0">Cor</label>
                      <input
                        type="color"
                        value={captionsShadowColor}
                        onChange={(e) => setCaptionsShadowColor(e.target.value)}
                        className="h-7 w-12 cursor-pointer rounded border border-zinc-700 bg-zinc-800"
                      />
                    </div>
                    <div className="flex items-center gap-3">
                      <label className="text-xs text-zinc-400 w-16 flex-shrink-0">Distância</label>
                      <input
                        type="range"
                        min={0}
                        max={20}
                        value={captionsShadowDistance}
                        onChange={(e) => setCaptionsShadowDistance(Number(e.target.value))}
                        className="flex-1 accent-zinc-300"
                      />
                      <span className="text-xs text-zinc-400 w-4">{captionsShadowDistance}</span>
                    </div>
                  </div>
                )}
              </div>
            </div>

            {/* Vertical offset */}
            <div className="flex items-center gap-3">
              <label className="text-xs text-zinc-400 w-20 flex-shrink-0">Offset Vertical</label>
              <input
                type="range"
                min={-100}
                max={100}
                value={captionsVerticalOffset}
                onChange={(e) => setCaptionsVerticalOffset(Number(e.target.value))}
                className="flex-1 accent-zinc-300"
              />
              <span className="text-xs text-zinc-400 w-8 text-right">{captionsVerticalOffset}</span>
            </div>
          </div>
        )}
      </div>

      {/* Upload Options */}
      <div className="space-y-4 rounded-lg border border-zinc-700 bg-zinc-900 p-4">
        <p className="text-xs font-medium text-zinc-300 uppercase tracking-wider">Upload Options</p>

        {/* Privacy pill selector */}
        <div className="space-y-1.5">
          <label className="text-xs text-zinc-400">Privacy</label>
          <div className="flex gap-2">
            {(['private', 'unlisted', 'public'] as const).map((p) => (
              <button
                key={p}
                onClick={() => setUploadPrivacy(p)}
                className={[
                  'flex-1 rounded-full px-3 py-1.5 text-xs font-medium capitalize transition-colors',
                  uploadPrivacy === p
                    ? 'bg-zinc-300 text-zinc-900'
                    : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700',
                ].join(' ')}
              >
                {p}
              </button>
            ))}
          </div>
        </div>

        {/* Schedule toggle */}
        <div className="flex items-center justify-between">
          <div>
            <p className="text-xs text-zinc-300">Schedule Sequentially</p>
            <p className="text-xs text-zinc-600">2 uploads/day per type</p>
          </div>
          <button
            onClick={() => setUploadScheduleEnabled((v) => !v)}
            className={[
              'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
              uploadScheduleEnabled ? 'bg-zinc-300' : 'bg-zinc-700',
            ].join(' ')}
            role="switch"
            aria-checked={uploadScheduleEnabled}
          >
            <span
              className={[
                'inline-block h-3.5 w-3.5 transform rounded-full bg-zinc-900 transition-transform',
                uploadScheduleEnabled ? 'translate-x-4' : 'translate-x-1',
              ].join(' ')}
            />
          </button>
        </div>

        {/* Auto upload toggle */}
        <div className="flex items-center justify-between">
          <div>
            <p className="text-xs text-zinc-300">Auto Upload</p>
            <p className="text-xs text-zinc-600">Upload immediately vs. manual queue</p>
          </div>
          <button
            onClick={() => setUploadAutoEnabled((v) => !v)}
            className={[
              'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
              uploadAutoEnabled ? 'bg-zinc-300' : 'bg-zinc-700',
            ].join(' ')}
            role="switch"
            aria-checked={uploadAutoEnabled}
          >
            <span
              className={[
                'inline-block h-3.5 w-3.5 transform rounded-full bg-zinc-900 transition-transform',
                uploadAutoEnabled ? 'translate-x-4' : 'translate-x-1',
              ].join(' ')}
            />
          </button>
        </div>

        {/* Dry run toggle */}
        <div className="flex items-center justify-between">
          <div>
            <p className="text-xs text-zinc-300">Dry Run</p>
            <p className="text-xs text-zinc-600">Generate thumbnails without uploading</p>
          </div>
          <button
            onClick={() => setUploadDryRun((v) => !v)}
            className={[
              'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
              uploadDryRun ? 'bg-zinc-300' : 'bg-zinc-700',
            ].join(' ')}
            role="switch"
            aria-checked={uploadDryRun}
          >
            <span
              className={[
                'inline-block h-3.5 w-3.5 transform rounded-full bg-zinc-900 transition-transform',
                uploadDryRun ? 'translate-x-4' : 'translate-x-1',
              ].join(' ')}
            />
          </button>
        </div>
      </div>

      {/* Submit button */}
      {!isLoadingPreference && (
        <button
          onClick={() => void handleSubmit()}
          disabled={isSubmitDisabled}
          className="w-full rounded-md bg-zinc-100 px-4 py-2 text-sm font-medium text-zinc-900 transition-colors hover:bg-zinc-200 disabled:cursor-not-allowed disabled:opacity-40"
        >
          {isSubmitting ? 'Starting…' : 'Next'}
        </button>
      )}

      {isLoadingPreference && (
        <div className="flex items-center gap-2 text-xs text-zinc-500">
          <span className="inline-block h-3 w-3 rounded-full border-2 border-zinc-500 border-t-transparent animate-spin" />
          <span>Loading preference…</span>
        </div>
      )}
    </div>
  );
}
