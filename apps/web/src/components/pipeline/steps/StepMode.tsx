'use client';

import { useEffect, useState } from 'react';
import { useAppStore } from '@/store/appStore';
import { usePipelineStore, type PreviewStatus } from '@/store/pipelineStore';
import { useChannelStore } from '@/store/channelStore';
import { AI_DEFAULTS, LONGFORM_DEFAULTS } from '@/types/pipeline';
import type { AntiDupEffects, AntiDuplicateConfig, BackgroundMusicConfig, BrandingConfig, CaptionsConfig, GatePayload, ModeConfig, PriorClipsResponse, TextOverlayItem, TextOverlaysConfig, VideoOverlayConfig, WorkflowMode } from '@/types/pipeline';
import { PositionGrid } from '@/components/ui/PositionGrid';

interface StepModeProps {
  historical?: GatePayload;
}

export function StepMode({ historical }: StepModeProps) {
  const goUrl = useAppStore((s) => s.goUrl);
  const activeRunId = usePipelineStore((s) => s.activeRunId);
  const advance = usePipelineStore((s) => s.advance);
  const run = usePipelineStore((s) => s.run);
  const videoInfo = usePipelineStore((s) => s.videoInfo);
  const generatePreview = usePipelineStore((s) => s.generatePreview);
  const previewStatus = usePipelineStore((s) => s.previewStatus);
  const previewPercent = usePipelineStore((s) => s.previewPercent);
  const previewUrl = usePipelineStore((s) => s.previewUrl);
  const previewError = usePipelineStore((s) => s.previewError);
  const channels = useChannelStore((s) => s.channels);
  const fetchChannels = useChannelStore((s) => s.fetchChannels);

  const isHistorical = historical !== undefined;

  // ── Resolve channel logo for branding preview ─────────────────────────────
  const activeChannel = channels.find((c) => c.ID === run?.channel_id) ?? channels.find((c) => c.IsFavorite) ?? channels[0] ?? null;

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
  const [antiDupEffects, setAntiDupEffects] = useState<AntiDupEffects>({
    speed_boost: true, crop: true, color_grading: true,
    noise: true, noise_strength: 3, zoom: true, transitions: true,
  });

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

  // ── Background Music state ─────────────────────────────────────────────────
  const [musicEnabled, setMusicEnabled] = useState<boolean>(false);
  const [musicMode, setMusicMode] = useState<BackgroundMusicConfig['mode']>('random');
  const [musicCustomPath, setMusicCustomPath] = useState<string>('');
  const [musicSelectedTrack, setMusicSelectedTrack] = useState<string>('');
  const [musicVolumePct, setMusicVolumePct] = useState<number>(7);
  const [musicLibraryTracks, setMusicLibraryTracks] = useState<string[]>([]);

  // ── Branding state ─────────────────────────────────────────────────────────
  const [brandingLogoEnabled, setBrandingLogoEnabled] = useState<boolean>(false);
  const [brandingLogoPath, setBrandingLogoPath] = useState<string>('');
  const [brandingLogoIsCustom, setBrandingLogoIsCustom] = useState<boolean>(false);
  const [brandingLogoPosition, setBrandingLogoPosition] = useState<string>('top_right');
  const [brandingLogoOpacity, setBrandingLogoOpacity] = useState<number>(80);
  const [brandingLogoScale, setBrandingLogoScale] = useState<number>(10);
  const [brandingLogoPulse, setBrandingLogoPulse] = useState<boolean>(false);
  const [brandingIntroEnabled, setBrandingIntroEnabled] = useState<boolean>(false);
  const [brandingOutroEnabled, setBrandingOutroEnabled] = useState<boolean>(false);

  // ── Video Overlay state ────────────────────────────────────────────────────
  const [overlayEnabled, setOverlayEnabled] = useState<boolean>(false);
  const [overlayVideoPath, setOverlayVideoPath] = useState<string>('');
  const [overlayScalePct, setOverlayScalePct] = useState<number>(100);
  const [overlayPosition, setOverlayPosition] = useState<string>('mid_center');
  const [overlayAppearances, setOverlayAppearances] = useState<number>(1);
  const [overlayEndOffsetSec, setOverlayEndOffsetSec] = useState<number>(0);
  const [overlayChromaKey, setOverlayChromaKey] = useState<VideoOverlayConfig['chroma_key']>('none');
  const [overlayLibraryFiles, setOverlayLibraryFiles] = useState<string[]>([]);

  // ── Text Overlays state ────────────────────────────────────────────────────
  const [textOverlaysEnabled, setTextOverlaysEnabled] = useState<boolean>(false);
  const [textOverlayItems, setTextOverlayItems] = useState<TextOverlayItem[]>([]);

  const [presetName, setPresetName] = useState<string>('');
  const [presets, setPresets] = useState<Array<{ name: string; config: ModeConfig; updatedAt: number }>>([]);
  const [activePresetName, setActivePresetName] = useState<string | null>(null);

  // ── Derived validation (computed — not useState) ───────────────────────────
  const isMinDurationInvalid = selectedMode === 'ai' && minDurationSecs >= clipDurationSecs;

  // ── Loading / submit state ─────────────────────────────────────────────────
  const [isLoadingPreference, setIsLoadingPreference] = useState<boolean>(true);
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

  // ── Apply a ModeConfig to all state (used by presets + historical) ─────────
  const applyModeConfig = (cfg: ModeConfig) => {
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
    if (cfg.background_music) {
      setMusicEnabled(cfg.background_music.enabled);
      setMusicMode(cfg.background_music.mode);
      if (cfg.background_music.custom_path) setMusicCustomPath(cfg.background_music.custom_path);
      if (cfg.background_music.selected_track) setMusicSelectedTrack(cfg.background_music.selected_track);
      setMusicVolumePct(cfg.background_music.volume_pct);
    }
    if (cfg.branding) {
      setBrandingLogoEnabled(cfg.branding.logo_enabled);
      if (cfg.branding.logo_path != null) {
        setBrandingLogoPath(cfg.branding.logo_path);
        setBrandingLogoIsCustom(true);
      }
      setBrandingLogoPosition(cfg.branding.logo_position);
      setBrandingLogoOpacity(cfg.branding.logo_opacity);
      setBrandingLogoScale(cfg.branding.logo_scale);
      setBrandingLogoPulse(cfg.branding.logo_pulse);
      setBrandingIntroEnabled(cfg.branding.intro_enabled);
      setBrandingOutroEnabled(cfg.branding.outro_enabled);
    }
    if (cfg.video_overlay) {
      setOverlayEnabled(cfg.video_overlay.enabled);
      if (cfg.video_overlay.video_path) setOverlayVideoPath(cfg.video_overlay.video_path);
      setOverlayScalePct(cfg.video_overlay.scale_pct);
      setOverlayPosition(cfg.video_overlay.position);
      setOverlayAppearances(cfg.video_overlay.appearances);
      setOverlayEndOffsetSec(cfg.video_overlay.end_offset_sec);
      if (cfg.video_overlay.chroma_key) setOverlayChromaKey(cfg.video_overlay.chroma_key);
    }
    if (cfg.text_overlays) {
      setTextOverlaysEnabled(cfg.text_overlays.enabled);
      setTextOverlayItems(cfg.text_overlays.items);
    }
  };

  // ── Fetch channels if not loaded (needed for logo preview) ────────────────
  useEffect(() => {
    if (channels.length === 0) void fetchChannels(goUrl);
  }, [goUrl]); // eslint-disable-line react-hooks/exhaustive-deps

  // ── Mount effect: historical pre-fill or fetch preference ──────────────────
  useEffect(() => {
    console.log('[StepMode] mount — isHistorical:', isHistorical);

    // Priority 1: historical back-navigation pre-fill
    if (historical?.kind === 'mode' && historical.mode_config) {
      console.log('[StepMode] pre-filling from historical mode_config');
      applyModeConfig(historical.mode_config);
      setIsLoadingPreference(false);
      return;
    }

    // Priority 2: fetch persisted preference from settings API
    const fetchPreference = async () => {
      try {
        console.log('[StepMode] fetching settings from', goUrl);
        const res = await fetch(`${goUrl}/api/settings`);
        const data = (await res.json()) as Array<{ key: string; value: string }>;

        // Extract saved presets (keys starting with "preset_"), sort newest first
        const loadedPresets: Array<{ name: string; config: ModeConfig; updatedAt: number }> = [];
        for (const s of data) {
          if (s.key.startsWith('preset_')) {
            try {
              const config = JSON.parse(s.value) as ModeConfig;
              loadedPresets.push({ name: s.key.slice('preset_'.length), config, updatedAt: (s as { key: string; value: string; updated_at?: number }).updated_at ?? 0 });
            } catch { /* skip malformed */ }
          }
        }
        loadedPresets.sort((a, b) => b.updatedAt - a.updatedAt);
        setPresets(loadedPresets);

        // Auto-apply the most recently updated preset (first in sorted list)
        if (loadedPresets.length > 0) {
          const latest = loadedPresets[0];
          console.log('[StepMode] auto-applying most recent preset:', latest.name);
          applyModeConfig(latest.config);
          setActivePresetName(latest.name);
          setPresetName(latest.name);
          setIsLoadingPreference(false);
          // Still check for prior clips
          if (run?.url) {
            try {
              const prRes = await fetch(`${goUrl}/api/pipeline/runs/prior-clips?url=${encodeURIComponent(run.url)}`);
              if (prRes.ok) {
                const prData = (await prRes.json()) as PriorClipsResponse;
                if (prData.exists && prData.run_id != null) setPriorRunId(prData.run_id);
              }
            } catch { /* non-fatal */ }
          }
          return;
        }

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
          speed_boost: parseBoolKey('anti_dup_effect_speed_boost', true),
          crop: parseBoolKey('anti_dup_effect_crop', true),
          color_grading: parseBoolKey('anti_dup_effect_color_grading', true),
          noise: parseBoolKey('anti_dup_effect_noise', true),
          noise_strength: parseIntKey('anti_dup_effect_noise_strength', 3),
          blur: parseBoolKey('anti_dup_effect_blur', false),
          blur_edge_pct: parseIntKey('anti_dup_effect_blur_edge_pct', 10),
          zoom: parseBoolKey('anti_dup_effect_zoom', true),
          transitions: parseBoolKey('anti_dup_effect_transitions', true),
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
        setMusicEnabled(parseBoolKey('music_enabled', false));
        const musicModeVal = data.find((s) => s.key === 'music_mode')?.value;
        setMusicMode(musicModeVal === 'library' ? 'library' : musicModeVal === 'custom' ? 'custom' : 'random');
        setMusicCustomPath(data.find((s) => s.key === 'music_custom_path')?.value ?? '');
        setMusicSelectedTrack(data.find((s) => s.key === 'music_selected_track')?.value ?? '');
        setMusicVolumePct(parseIntKey('music_volume_pct', 7));
        setBrandingLogoEnabled(parseBoolKey('branding_logo_enabled', false));
        const savedLogoPath = data.find((s) => s.key === 'branding_logo_path')?.value ?? '';
        setBrandingLogoPath(savedLogoPath);
        setBrandingLogoIsCustom(savedLogoPath !== '');
        setBrandingLogoPosition(data.find((s) => s.key === 'branding_logo_position')?.value ?? 'top_right');
        setBrandingLogoOpacity(parseIntKey('branding_logo_opacity', 80));
        setBrandingLogoScale(parseIntKey('branding_logo_scale', 10));
        setBrandingLogoPulse(parseBoolKey('branding_logo_pulse', false));
        setBrandingIntroEnabled(parseBoolKey('branding_intro_enabled', false));
        setBrandingOutroEnabled(parseBoolKey('branding_outro_enabled', false));
        setOverlayEnabled(parseBoolKey('overlay_enabled', false));
        setOverlayVideoPath(data.find((s) => s.key === 'overlay_video_path')?.value ?? '');
        setOverlayScalePct(parseIntKey('overlay_scale_pct', 100));
        setOverlayPosition(data.find((s) => s.key === 'overlay_position')?.value ?? 'mid_center');
        setOverlayAppearances(parseIntKey('overlay_appearances', 1));
        setOverlayEndOffsetSec(parseIntKey('overlay_end_offset_sec', 0));
        const overlayChromaVal = data.find((s) => s.key === 'overlay_chroma_key')?.value;
        const validChromaKeys = ['none','green','black','white','blue'] as const;
        type ChromaKey = typeof validChromaKeys[number];
        setOverlayChromaKey(overlayChromaVal && (validChromaKeys as readonly string[]).includes(overlayChromaVal) ? overlayChromaVal as ChromaKey : 'none');
        setTextOverlaysEnabled(parseBoolKey('text_overlays_enabled', false));
        const textOverlaysRaw = data.find((s) => s.key === 'text_overlays_items')?.value;
        if (textOverlaysRaw) {
          try { setTextOverlayItems(JSON.parse(textOverlaysRaw) as TextOverlayItem[]); } catch { /* ignore */ }
        }
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

  useEffect(() => {
    if (!musicEnabled || musicMode !== 'library') return;
    void fetch(`${goUrl}/api/music-library`)
      .then((r) => r.json())
      .then((tracks: string[]) => setMusicLibraryTracks(tracks))
      .catch(() => setMusicLibraryTracks([]));
  }, [goUrl, musicEnabled, musicMode]);

  useEffect(() => {
    if (!overlayEnabled) return;
    void fetch(`${goUrl}/api/overlay-library`)
      .then((r) => r.json())
      .then((files: string[]) => setOverlayLibraryFiles(files))
      .catch(() => setOverlayLibraryFiles([]));
  }, [goUrl, overlayEnabled]);

  // ── setEffect helper ───────────────────────────────────────────────────────
  const setEffect = (key: keyof AntiDupEffects, value: boolean | number) => {
    setAntiDupEffects((prev) => ({ ...prev, [key]: value }));
  };

  // ── Text Overlay item helpers ──────────────────────────────────────────────
  const addTextOverlayItem = () => {
    setTextOverlayItems((prev) => [
      ...prev,
      { text: '', apply_full: true, start_sec: 0, end_sec: 60, position: 'bottom_center' },
    ]);
  };

  const removeTextOverlayItem = (idx: number) => {
    setTextOverlayItems((prev) => prev.filter((_, i) => i !== idx));
  };

  const updateTextOverlayItem = (idx: number, patch: Partial<TextOverlayItem>) => {
    setTextOverlayItems((prev) => prev.map((item, i) => (i === idx ? { ...item, ...patch } : item)));
  };

  const handleSavePreset = async () => {
    if (!presetName.trim()) return;
    const name = presetName.trim();
    const cfg = buildModeConfig();
    try {
      await fetch(`${goUrl}/api/settings`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ key: `preset_${name}`, value: JSON.stringify(cfg) }),
      });
      // Update local state — put updated preset first (most recent)
      const now = Date.now();
      setPresets((prev) => {
        const filtered = prev.filter((p) => p.name !== name);
        return [{ name, config: cfg, updatedAt: now }, ...filtered];
      });
      setActivePresetName(name);
      setPresetName('');
    } catch {
      // non-fatal
    }
  };

  const handleSelectPreset = (name: string) => {
    const preset = presets.find((p) => p.name === name);
    if (!preset) return;
    applyModeConfig(preset.config);
    setActivePresetName(name);
    setPresetName(name);
  };

  const handleDeletePreset = async (name: string) => {
    try {
      await fetch(`${goUrl}/api/settings?key=${encodeURIComponent(`preset_${name}`)}`, {
        method: 'DELETE',
      });
      setPresets((prev) => prev.filter((p) => p.name !== name));
      if (activePresetName === name) setActivePresetName(null);
    } catch {
      // non-fatal
    }
  };

  // ── buildModeConfig ────────────────────────────────────────────────────────
  function buildModeConfig(): ModeConfig {
    const brandingCfg: BrandingConfig | undefined = (brandingLogoEnabled || brandingIntroEnabled || brandingOutroEnabled) ? {
      logo_enabled: brandingLogoEnabled,
      logo_path: brandingLogoIsCustom && brandingLogoPath ? brandingLogoPath : undefined,
      logo_position: brandingLogoPosition,
      logo_opacity: brandingLogoOpacity,
      logo_scale: brandingLogoScale,
      logo_pulse: brandingLogoPulse,
      intro_enabled: brandingIntroEnabled,
      outro_enabled: brandingOutroEnabled,
    } : undefined;
    const overlayCfg: VideoOverlayConfig | undefined = overlayEnabled ? {
      enabled: true,
      video_path: overlayVideoPath || undefined,
      scale_pct: overlayScalePct,
      position: overlayPosition,
      appearances: overlayAppearances,
      end_offset_sec: overlayEndOffsetSec,
      chroma_key: overlayChromaKey,
    } : undefined;
    const textOverlaysCfg: TextOverlaysConfig | undefined = textOverlaysEnabled ? {
      enabled: true,
      items: textOverlayItems,
    } : undefined;
    const musicCfg: BackgroundMusicConfig | undefined = musicEnabled ? {
      enabled: true,
      mode: musicMode,
      custom_path: musicCustomPath || undefined,
      selected_track: musicSelectedTrack || undefined,
      volume_pct: musicVolumePct,
    } : undefined;
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
        background_music: musicCfg,
        branding: brandingCfg,
        video_overlay: overlayCfg,
        text_overlays: textOverlaysCfg,
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
      background_music: musicCfg,
      branding: brandingCfg,
      video_overlay: overlayCfg,
      text_overlays: textOverlaysCfg,
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
        { key: 'music_enabled', value: String(musicEnabled) },
        { key: 'music_mode', value: musicMode },
        { key: 'music_custom_path', value: musicCustomPath },
        { key: 'music_selected_track', value: musicSelectedTrack },
        { key: 'music_volume_pct', value: String(musicVolumePct) },
        { key: 'branding_logo_enabled', value: String(brandingLogoEnabled) },
        { key: 'branding_logo_path', value: brandingLogoPath },
        { key: 'branding_logo_position', value: brandingLogoPosition },
        { key: 'branding_logo_opacity', value: String(brandingLogoOpacity) },
        { key: 'branding_logo_scale', value: String(brandingLogoScale) },
        { key: 'branding_logo_pulse', value: String(brandingLogoPulse) },
        { key: 'branding_intro_enabled', value: String(brandingIntroEnabled) },
        { key: 'branding_outro_enabled', value: String(brandingOutroEnabled) },
        { key: 'overlay_enabled', value: String(overlayEnabled) },
        { key: 'overlay_video_path', value: overlayVideoPath },
        { key: 'overlay_scale_pct', value: String(overlayScalePct) },
        { key: 'overlay_position', value: overlayPosition },
        { key: 'overlay_appearances', value: String(overlayAppearances) },
        { key: 'overlay_end_offset_sec', value: String(overlayEndOffsetSec) },
        { key: 'overlay_chroma_key', value: overlayChromaKey ?? 'none' },
        { key: 'text_overlays_enabled', value: String(textOverlaysEnabled) },
        { key: 'text_overlays_items', value: JSON.stringify(textOverlayItems) },
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
    try {
      const cfg = buildModeConfig();
      await advance(goUrl, activeRunId, { mode_config: cfg });
    } finally {
      setIsSubmitting(false);
    }
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

      {/* Preset Carousel */}
      {presets.length > 0 && (
        <div className="space-y-2">
          <p className="text-xs font-medium text-zinc-400">Presets</p>
          <div className="flex gap-2 overflow-x-auto pb-1 scrollbar-none">
            {presets.map((p) => (
              <div
                key={p.name}
                className="flex-shrink-0 group"
              >
                <button
                  onClick={() => handleSelectPreset(p.name)}
                  className={[
                    'relative rounded-lg border px-3 py-2 text-xs font-medium transition-colors',
                    activePresetName === p.name
                      ? 'border-zinc-400 bg-zinc-800 text-zinc-100'
                      : 'border-zinc-700 bg-zinc-900 text-zinc-400 hover:border-zinc-500 hover:text-zinc-200',
                  ].join(' ')}
                >
                  <span className="pr-4">{p.name}</span>
                  <span
                    onClick={(e) => { e.stopPropagation(); void handleDeletePreset(p.name); }}
                    className="absolute right-1.5 top-1/2 -translate-y-1/2 text-zinc-600 hover:text-red-400 opacity-0 group-hover:opacity-100 transition-opacity cursor-pointer"
                  >
                    &times;
                  </span>
                </button>
              </div>
            ))}
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

          {/* Target Clip Duration — dynamic based on video length, same logic as Long Form */}
          <div className="space-y-1.5">
            <label className="text-xs text-zinc-400">Target Clip Duration</label>
            <div className="flex gap-2 flex-wrap">
              {(() => {
                const MIN_PART = 480; // 8 min
                const totalSec = videoInfo?.durationSec ?? 0;
                const suggestions: [number, string][] = [];
                for (const parts of [3, 2, 1]) {
                  if (totalSec === 0) break;
                  const partSec = Math.round(totalSec / parts / 60) * 60;
                  if (partSec >= MIN_PART) {
                    const mins = Math.round(partSec / 60);
                    suggestions.push([partSec, `${mins} min`]);
                  }
                }
                const options: [number, string][] = suggestions.length > 0
                  ? suggestions
                  : [[1020, '17 min'], [1560, '26 min'], [3120, '52 min']];
                return options.map(([secs, label]) => (
                  <button
                    key={secs}
                    onClick={() => setClipDurationSecs(secs)}
                    className={[
                      'rounded px-2 py-1.5 text-xs transition-colors',
                      clipDurationSecs === secs
                        ? 'bg-zinc-300 text-zinc-900 font-medium'
                        : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700',
                    ].join(' ')}
                  >
                    {label}
                  </button>
                ));
              })()}
            </div>
          </div>

          {/* Minimum Clip Duration */}
          <div className="space-y-1.5">
            <label className="text-xs text-zinc-400">Minimum Clip Duration</label>
            <div className="flex gap-2 flex-wrap">
              {([60, 120, 300, 480, 600, 900] as const).map((secs) => (
                <button
                  key={secs}
                  onClick={() => setMinDurationSecs(secs)}
                  className={[
                    'rounded px-2 py-1.5 text-xs transition-colors',
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

          {/* Segment duration presets — derived from video duration */}
          <div className="space-y-1.5">
            <label className="text-xs text-zinc-400">Segment Duration</label>
            <div className="flex gap-2 flex-wrap">
              {(() => {
                const MIN_PART = 480; // 8 min
                const totalSec = videoInfo?.durationSec ?? 0;
                const suggestions: [number, string][] = [];
                for (const parts of [3, 2, 1]) {
                  if (totalSec === 0) break;
                  const partSec = Math.round(totalSec / parts / 60) * 60;
                  if (partSec >= MIN_PART) {
                    const mins = Math.round(partSec / 60);
                    suggestions.push([partSec, `${mins} min`]);
                  }
                }
                const options: [number, string][] = suggestions.length > 0
                  ? suggestions
                  : [[1020, '17 min'], [1560, '26 min'], [3120, '52 min']];
                return options.map(([secs, label]) => (
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
                ));
              })()}
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

      {/* Text Overlays */}
      <div className="space-y-4 rounded-lg border border-zinc-700 bg-zinc-900 p-4">
        <div className="flex items-center justify-between">
          <p className="text-xs font-medium text-zinc-300 uppercase tracking-wider">Overlays de Texto</p>
          <button
            onClick={() => setTextOverlaysEnabled((v) => !v)}
            className={[
              'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
              textOverlaysEnabled ? 'bg-zinc-300' : 'bg-zinc-700',
            ].join(' ')}
            role="switch"
            aria-checked={textOverlaysEnabled}
          >
            <span
              className={[
                'inline-block h-3.5 w-3.5 transform rounded-full bg-zinc-900 transition-transform',
                textOverlaysEnabled ? 'translate-x-4' : 'translate-x-1',
              ].join(' ')}
            />
          </button>
        </div>

        {textOverlaysEnabled && (
          <div className="space-y-4">
            {/* Item list */}
            {textOverlayItems.map((item, idx) => (
              <div key={idx} className="space-y-3 rounded-md border border-zinc-700 p-3">
                <div className="flex items-center justify-between">
                  <span className="text-xs text-zinc-400">Overlay #{idx + 1}</span>
                  <button
                    onClick={() => removeTextOverlayItem(idx)}
                    className="text-xs text-zinc-600 hover:text-zinc-400 transition-colors"
                  >
                    Remover
                  </button>
                </div>

                {/* Text */}
                <textarea
                  value={item.text}
                  onChange={(e) => updateTextOverlayItem(idx, { text: e.target.value })}
                  placeholder="Texto do overlay…"
                  rows={2}
                  className="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-xs text-foreground placeholder-zinc-600 focus:outline-none focus:ring-1 focus:ring-zinc-500 resize-none"
                />

                {/* Apply full video */}
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={item.apply_full}
                    onChange={(e) => updateTextOverlayItem(idx, { apply_full: e.target.checked })}
                    className="rounded border-zinc-600 bg-zinc-800 text-zinc-300"
                  />
                  <span className="text-xs text-zinc-300">Aplicar no vídeo todo</span>
                </label>

                {/* Time range — only when not apply_full */}
                {!item.apply_full && (
                  <div className="flex items-center gap-3">
                    <div className="flex-1 space-y-1">
                      <label className="text-xs text-zinc-400">Início (s)</label>
                      <input
                        type="number"
                        min={0}
                        value={item.start_sec}
                        onChange={(e) => updateTextOverlayItem(idx, { start_sec: Number(e.target.value) })}
                        className="w-full rounded border border-zinc-700 bg-zinc-800 px-2 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-zinc-500"
                      />
                    </div>
                    <div className="flex-1 space-y-1">
                      <label className="text-xs text-zinc-400">Fim (s)</label>
                      <input
                        type="number"
                        min={0}
                        value={item.end_sec}
                        onChange={(e) => updateTextOverlayItem(idx, { end_sec: Number(e.target.value) })}
                        className="w-full rounded border border-zinc-700 bg-zinc-800 px-2 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-zinc-500"
                      />
                    </div>
                  </div>
                )}

                {/* Position grid */}
                <div className="space-y-1">
                  <label className="text-xs text-zinc-400">Posição</label>
                  <PositionGrid
                    value={item.position}
                    onChange={(pos) => updateTextOverlayItem(idx, { position: pos })}
                  />
                </div>
              </div>
            ))}

            {/* Add button */}
            <button
              onClick={addTextOverlayItem}
              className="w-full rounded border border-dashed border-zinc-600 px-3 py-2 text-xs text-zinc-500 hover:border-zinc-400 hover:text-zinc-300 transition-colors"
            >
              + Adicionar overlay de texto
            </button>
          </div>
        )}
      </div>

      {/* Video Overlay */}
      <div className="space-y-4 rounded-lg border border-zinc-700 bg-zinc-900 p-4">
        <div className="flex items-center justify-between">
          <p className="text-xs font-medium text-zinc-300 uppercase tracking-wider">Overlay de Vídeo</p>
          <button
            onClick={() => setOverlayEnabled((v) => !v)}
            className={[
              'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
              overlayEnabled ? 'bg-zinc-300' : 'bg-zinc-700',
            ].join(' ')}
            role="switch"
            aria-checked={overlayEnabled}
          >
            <span
              className={[
                'inline-block h-3.5 w-3.5 transform rounded-full bg-zinc-900 transition-transform',
                overlayEnabled ? 'translate-x-4' : 'translate-x-1',
              ].join(' ')}
            />
          </button>
        </div>

        {overlayEnabled && (
          <div className="space-y-4">
            {/* File selector */}
            <div className="space-y-1.5">
              <label className="text-xs text-zinc-400">Arquivo de Overlay</label>
              {overlayLibraryFiles.length > 0 ? (
                <select
                  value={overlayVideoPath}
                  onChange={(e) => setOverlayVideoPath(e.target.value)}
                  className="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-zinc-500"
                >
                  <option value="">— escolha um arquivo —</option>
                  {overlayLibraryFiles.map((f) => (
                    <option key={f} value={f}>{f}</option>
                  ))}
                </select>
              ) : (
                <input
                  type="text"
                  value={overlayVideoPath}
                  onChange={(e) => setOverlayVideoPath(e.target.value)}
                  placeholder="/caminho/para/overlay.mp4"
                  className="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-xs text-foreground placeholder-zinc-600 focus:outline-none focus:ring-1 focus:ring-zinc-500"
                />
              )}
            </div>

            {/* Scale slider */}
            <div className="flex items-center gap-3">
              <label className="text-xs text-zinc-400 w-20 flex-shrink-0">Escala</label>
              <input
                type="range"
                min={10}
                max={200}
                value={overlayScalePct}
                onChange={(e) => setOverlayScalePct(Number(e.target.value))}
                className="flex-1 accent-zinc-300"
              />
              <span className="text-xs text-zinc-400 w-10 text-right">{overlayScalePct}%</span>
            </div>

            {/* Position grid */}
            <div className="space-y-1.5">
              <label className="text-xs text-zinc-400">Posição</label>
              <PositionGrid value={overlayPosition} onChange={setOverlayPosition} />
            </div>

            {/* Appearances */}
            <div className="space-y-1.5">
              <label className="text-xs text-zinc-400">Nº de Aparições</label>
              <div className="flex gap-2">
                {([1, 2, 3, 4, 5] as const).map((n) => (
                  <button
                    key={n}
                    onClick={() => setOverlayAppearances(n)}
                    className={[
                      'flex-1 rounded px-2 py-1.5 text-xs transition-colors',
                      overlayAppearances === n
                        ? 'bg-zinc-300 text-zinc-900 font-medium'
                        : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700',
                    ].join(' ')}
                  >
                    {n}×
                  </button>
                ))}
              </div>
            </div>

            {/* End offset */}
            <div className="flex items-center gap-3">
              <label className="text-xs text-zinc-400 w-20 flex-shrink-0">Offset Final</label>
              <input
                type="range"
                min={0}
                max={10}
                value={overlayEndOffsetSec}
                onChange={(e) => setOverlayEndOffsetSec(Number(e.target.value))}
                className="flex-1 accent-zinc-300"
              />
              <span className="text-xs text-zinc-400 w-6 text-right">{overlayEndOffsetSec}s</span>
            </div>

            {/* Chroma key */}
            <div className="space-y-1.5">
              <label className="text-xs text-zinc-400">Chroma Key</label>
              <div className="flex flex-wrap gap-2">
                {(['none', 'green', 'black', 'white', 'blue'] as const).map((key) => (
                  <button
                    key={key}
                    onClick={() => setOverlayChromaKey(key)}
                    className={[
                      'rounded px-3 py-1.5 text-xs capitalize transition-colors',
                      overlayChromaKey === key
                        ? 'bg-zinc-300 text-zinc-900 font-medium'
                        : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700',
                    ].join(' ')}
                  >
                    {key === 'none' ? 'Nenhum' : key}
                  </button>
                ))}
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Branding */}
      <div className="space-y-4 rounded-lg border border-zinc-700 bg-zinc-900 p-4">
        <p className="text-xs font-medium text-zinc-300 uppercase tracking-wider">Branding</p>

        {/* Watermark (Logo) */}
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <p className="text-xs text-zinc-300">Watermark (Logo)</p>
            <button
              onClick={() => setBrandingLogoEnabled((v) => !v)}
              className={[
                'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
                brandingLogoEnabled ? 'bg-zinc-300' : 'bg-zinc-700',
              ].join(' ')}
              role="switch"
              aria-checked={brandingLogoEnabled}
            >
              <span
                className={[
                  'inline-block h-3.5 w-3.5 transform rounded-full bg-zinc-900 transition-transform',
                  brandingLogoEnabled ? 'translate-x-4' : 'translate-x-1',
                ].join(' ')}
              />
            </button>
          </div>

          {brandingLogoEnabled && (
            <div className="space-y-3 pl-1">
              {/* Logo source */}
              <div className="space-y-2">
                <label className="text-xs text-zinc-400">Logo para watermark</label>

                {/* Auto — channel logo preview row */}
                {!brandingLogoIsCustom && (
                  <div className="flex items-center gap-3 rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-2">
                    {activeChannel ? (
                      <img
                        src={`${goUrl}/api/channels/${activeChannel.ID}/avatar`}
                        alt=""
                        className="h-9 w-9 rounded-full object-cover flex-shrink-0"
                        onError={(e) => { (e.currentTarget as HTMLImageElement).style.display = 'none'; (e.currentTarget.nextElementSibling as HTMLElement | null)?.style.setProperty('display', 'flex'); }}
                      />
                    ) : null}
                    <div
                      className="h-9 w-9 rounded-full flex-shrink-0 items-center justify-center text-xs font-bold text-white"
                      style={{ display: activeChannel ? 'none' : 'flex', background: activeChannel ? `hsl(${(activeChannel.ID * 67) % 360}, 55%, 40%)` : '#52525b' }}
                    >
                      {activeChannel ? (activeChannel.ChannelTitle || activeChannel.Name).charAt(0).toUpperCase() : '?'}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-xs font-medium text-zinc-200 truncate">
                        {activeChannel?.ChannelTitle || activeChannel?.Name || 'Logo do canal (automático)'}
                      </p>
                      <p className="text-xs text-zinc-500">Baixado do YouTube</p>
                    </div>
                    <button
                      onClick={() => setBrandingLogoIsCustom(true)}
                      className="text-xs text-zinc-400 hover:text-zinc-200 flex-shrink-0 transition-colors"
                    >
                      Trocar
                    </button>
                  </div>
                )}

                {/* Custom path input */}
                {brandingLogoIsCustom && (
                  <div className="space-y-2">
                    <input
                      type="text"
                      value={brandingLogoPath}
                      onChange={(e) => setBrandingLogoPath(e.target.value)}
                      placeholder="/caminho/para/logo.png"
                      className="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-xs text-foreground placeholder-zinc-600 focus:outline-none focus:ring-1 focus:ring-zinc-500"
                    />
                    <button
                      onClick={() => { setBrandingLogoIsCustom(false); setBrandingLogoPath(''); }}
                      className="text-xs text-zinc-500 hover:text-zinc-300 transition-colors"
                    >
                      ← Usar logo do canal
                    </button>
                  </div>
                )}
              </div>

              {/* Position grid */}
              <div className="space-y-1.5">
                <label className="text-xs text-zinc-400">Posição</label>
                <PositionGrid value={brandingLogoPosition} onChange={setBrandingLogoPosition} />
              </div>

              {/* Opacity slider */}
              <div className="flex items-center gap-3">
                <label className="text-xs text-zinc-400 w-20 flex-shrink-0">Opacidade</label>
                <input
                  type="range"
                  min={0}
                  max={100}
                  value={brandingLogoOpacity}
                  onChange={(e) => setBrandingLogoOpacity(Number(e.target.value))}
                  className="flex-1 accent-zinc-300"
                />
                <span className="text-xs text-zinc-400 w-8 text-right">{brandingLogoOpacity}%</span>
              </div>

              {/* Scale slider */}
              <div className="flex items-center gap-3">
                <label className="text-xs text-zinc-400 w-20 flex-shrink-0">Tamanho</label>
                <input
                  type="range"
                  min={5}
                  max={30}
                  value={brandingLogoScale}
                  onChange={(e) => setBrandingLogoScale(Number(e.target.value))}
                  className="flex-1 accent-zinc-300"
                />
                <span className="text-xs text-zinc-400 w-8 text-right">{brandingLogoScale}%</span>
              </div>

              {/* Pulse toggle */}
              <div className="flex items-center justify-between">
                <p className="text-xs text-zinc-300">Watermark pulsante</p>
                <button
                  onClick={() => setBrandingLogoPulse((v) => !v)}
                  className={[
                    'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
                    brandingLogoPulse ? 'bg-zinc-300' : 'bg-zinc-700',
                  ].join(' ')}
                  role="switch"
                  aria-checked={brandingLogoPulse}
                >
                  <span
                    className={[
                      'inline-block h-3.5 w-3.5 transform rounded-full bg-zinc-900 transition-transform',
                      brandingLogoPulse ? 'translate-x-4' : 'translate-x-1',
                    ].join(' ')}
                  />
                </button>
              </div>
            </div>
          )}
        </div>

        {/* Intro toggle */}
        <div className="flex items-center justify-between">
          <div>
            <p className="text-xs text-zinc-300">Intro animado</p>
            <p className="text-xs text-zinc-600">3s a partir do logo do canal</p>
          </div>
          <button
            onClick={() => setBrandingIntroEnabled((v) => !v)}
            className={[
              'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
              brandingIntroEnabled ? 'bg-zinc-300' : 'bg-zinc-700',
            ].join(' ')}
            role="switch"
            aria-checked={brandingIntroEnabled}
          >
            <span
              className={[
                'inline-block h-3.5 w-3.5 transform rounded-full bg-zinc-900 transition-transform',
                brandingIntroEnabled ? 'translate-x-4' : 'translate-x-1',
              ].join(' ')}
            />
          </button>
        </div>

        {/* Outro toggle */}
        <div className="flex items-center justify-between">
          <p className="text-xs text-zinc-300">Outro (tela final)</p>
          <button
            onClick={() => setBrandingOutroEnabled((v) => !v)}
            className={[
              'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
              brandingOutroEnabled ? 'bg-zinc-300' : 'bg-zinc-700',
            ].join(' ')}
            role="switch"
            aria-checked={brandingOutroEnabled}
          >
            <span
              className={[
                'inline-block h-3.5 w-3.5 transform rounded-full bg-zinc-900 transition-transform',
                brandingOutroEnabled ? 'translate-x-4' : 'translate-x-1',
              ].join(' ')}
            />
          </button>
        </div>
      </div>

      {/* Música de Fundo */}
      <div className="space-y-4 rounded-lg border border-zinc-700 bg-zinc-900 p-4">
        <div className="flex items-center justify-between">
          <p className="text-xs font-medium text-zinc-300 uppercase tracking-wider">Música de Fundo</p>
          <button
            onClick={() => setMusicEnabled((v) => !v)}
            className={[
              'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
              musicEnabled ? 'bg-zinc-300' : 'bg-zinc-700',
            ].join(' ')}
            role="switch"
            aria-checked={musicEnabled}
          >
            <span
              className={[
                'inline-block h-3.5 w-3.5 transform rounded-full bg-zinc-900 transition-transform',
                musicEnabled ? 'translate-x-4' : 'translate-x-1',
              ].join(' ')}
            />
          </button>
        </div>

        {musicEnabled && (
          <div className="space-y-4">
            {/* Mode selector */}
            <div className="space-y-1.5">
              <label className="text-xs text-zinc-400">Seleção</label>
              <div className="flex gap-2">
                {([
                  ['random', 'Aleatório'],
                  ['library', 'Biblioteca'],
                  ['custom', 'Arquivo'],
                ] as [BackgroundMusicConfig['mode'], string][]).map(([m, label]) => (
                  <button
                    key={m}
                    onClick={() => setMusicMode(m)}
                    className={[
                      'flex-1 rounded px-2 py-1.5 text-xs transition-colors',
                      musicMode === m
                        ? 'bg-zinc-300 text-zinc-900 font-medium'
                        : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700',
                    ].join(' ')}
                  >
                    {label}
                  </button>
                ))}
              </div>
            </div>

            {/* Library track list */}
            {musicMode === 'library' && (
              <div className="space-y-1.5">
                <label className="text-xs text-zinc-400">Faixa</label>
                {musicLibraryTracks.length === 0 ? (
                  <p className="text-xs text-zinc-600">Nenhuma faixa encontrada. Configure o caminho da biblioteca nas configurações.</p>
                ) : (
                  <select
                    value={musicSelectedTrack}
                    onChange={(e) => setMusicSelectedTrack(e.target.value)}
                    className="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-zinc-500"
                  >
                    <option value="">— escolha uma faixa —</option>
                    {musicLibraryTracks.map((t) => (
                      <option key={t} value={t}>{t}</option>
                    ))}
                  </select>
                )}
              </div>
            )}

            {/* Custom path input */}
            {musicMode === 'custom' && (
              <div className="space-y-1.5">
                <label className="text-xs text-zinc-400">Caminho do arquivo</label>
                <input
                  type="text"
                  value={musicCustomPath}
                  onChange={(e) => setMusicCustomPath(e.target.value)}
                  placeholder="/caminho/para/musica.mp3"
                  className="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-xs text-foreground placeholder-zinc-600 focus:outline-none focus:ring-1 focus:ring-zinc-500"
                />
              </div>
            )}

            {/* Volume slider */}
            <div className="flex items-center gap-3">
              <label className="text-xs text-zinc-400 w-16 flex-shrink-0">Volume</label>
              <input
                type="range"
                min={0}
                max={30}
                value={musicVolumePct}
                onChange={(e) => setMusicVolumePct(Number(e.target.value))}
                className="flex-1 accent-zinc-300"
              />
              <span className="text-xs text-zinc-400 w-8 text-right">{musicVolumePct}%</span>
            </div>
            <p className="text-xs text-zinc-600">Recomendado: 5–10%</p>
          </div>
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

      {/* Save / Update Preset */}
      {(() => {
        const nameExists = presets.some((p) => p.name === presetName.trim());
        return (
          <div className="flex gap-2">
            <input
              type="text"
              value={presetName}
              onChange={(e) => {
                setPresetName(e.target.value);
                setActivePresetName(presets.find((p) => p.name === e.target.value.trim())?.name ?? null);
              }}
              onKeyDown={(e) => { if (e.key === 'Enter' && presetName.trim()) void handleSavePreset(); }}
              placeholder="Nome do preset…"
              className="flex-1 rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-xs text-foreground placeholder-zinc-600 focus:outline-none focus:ring-1 focus:ring-zinc-500"
            />
            <button
              onClick={() => void handleSavePreset()}
              disabled={!presetName.trim()}
              className="rounded border border-zinc-600 px-3 py-1.5 text-xs text-zinc-400 hover:border-zinc-400 hover:text-zinc-200 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {nameExists ? 'Atualizar' : 'Criar'}
            </button>
          </div>
        );
      })()}

      {/* Preview section */}
      {!isLoadingPreference && run?.video_path && (
        <div className="space-y-3 rounded-lg border border-zinc-700 bg-zinc-900 p-4">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-medium text-zinc-300">Preview</h3>
            <button
              onClick={() => {
                if (!activeRunId) return;
                const cfg = buildModeConfig();
                void generatePreview(goUrl, activeRunId, cfg);
              }}
              disabled={previewStatus === 'generating'}
              className="rounded-md bg-zinc-700 px-3 py-1.5 text-xs font-medium text-zinc-200 transition-colors hover:bg-zinc-600 disabled:cursor-not-allowed disabled:opacity-40"
            >
              {previewStatus === 'generating' ? 'Generating…' : previewStatus === 'ready' ? 'Regenerate Preview' : 'Generate Preview'}
            </button>
          </div>

          {/* Progress bar */}
          {previewStatus === 'generating' && (
            <div className="space-y-1">
              <div className="h-2 w-full overflow-hidden rounded-full bg-zinc-800">
                <div
                  className="h-full rounded-full bg-blue-500 transition-all duration-300"
                  style={{ width: `${Math.min(100, previewPercent)}%` }}
                />
              </div>
              <p className="text-xs text-zinc-500">{Math.round(previewPercent)}%</p>
            </div>
          )}

          {/* Error */}
          {previewStatus === 'error' && previewError && (
            <p className="text-xs text-red-400">{previewError}</p>
          )}

          {/* Video player */}
          {previewStatus === 'ready' && previewUrl && (
            <video
              src={`${goUrl}${previewUrl}`}
              controls
              preload="metadata"
              className="w-full rounded-md"
            />
          )}
        </div>
      )}

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
