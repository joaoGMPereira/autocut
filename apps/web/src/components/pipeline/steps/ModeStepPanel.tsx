'use client';

import { useEffect, useRef, useState } from 'react';
import { useWizardStore } from '@/store/wizardStore';
import { useChannelStore } from '@/store/channelStore';
import { useAppStore } from '@/store/appStore';
import { usePipelineStore } from '@/store/pipelineStore';
import type { ModeConfig } from '@/types/pipeline';
import { ModeCard } from '@/components/pipeline/ModeCard';
import { DurationChips } from '@/components/pipeline/DurationChips';
import { ToggleSection } from '@/components/pipeline/ToggleSection';
import { MetadataPreview } from '@/components/pipeline/MetadataPreview';
import type { VideoMetadata } from '@/components/pipeline/MetadataPreview';
import { VideoPreview } from '@/components/pipeline/VideoPreview';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Checkbox } from '@/components/ui/checkbox';
import { Slider } from '@/components/ui/slider';
import { Label } from '@/components/ui/label';
import { Separator } from '@/components/ui/separator';
import { cn } from '@/lib/utils';
import { useDispatcherStore } from '@/store/dispatcherStore';

type WorkflowMode = 'ai' | 'longform';
type AntiDuplicateMode = 'subtle' | 'aggressive';
type CaptionStyle = 'BOLD' | 'HIGHLIGHT' | 'SUBTITLE';
type LogoPosition = 'TOP_LEFT' | 'TOP_RIGHT' | 'BOTTOM_LEFT' | 'BOTTOM_RIGHT';

const SKIP_MUSIC_PRESETS = [0, 1, 2, 3, 5];
const MAX_DURATION_PRESETS = [17, 25, 51];
const MIN_DURATION_PRESETS = [1, 2, 5, 8, 10, 15];

function sensitivityLabel(pct: number): string {
  if (pct <= 55) return `Liberal (${pct}%)`;
  if (pct <= 75) return `Equilibrado (${pct}%)`;
  return `Muito Seletivo (${pct}%)`;
}

export function ModeStepPanel() {
  const goUrl = useAppStore((s) => s.goUrl);
  const channels = useChannelStore((s) => s.channels);
  const selectedChannelId = useChannelStore((s) => s.selectedChannelId);
  const fetchChannels = useChannelStore((s) => s.fetchChannels);
  const selectChannel = useChannelStore((s) => s.selectChannel);
  const loadChannelConfig = useChannelStore((s) => s.loadChannelConfig);
  const setCanAdvance = useWizardStore((s) => s.setCanAdvance);
  const setStepHidden = useWizardStore((s) => s.setStepHidden);
  const setPendingModeConfig = useWizardStore((s) => s.setPendingModeConfig);
  const run = usePipelineStore((s) => s.run);

  const [mode, setMode] = useState<WorkflowMode | null>(null);

  // AI config
  const [sensitivityPct, setSensitivityPct] = useState(70);
  const [skipMusicMinutes, setSkipMusicMinutes] = useState(0);
  const [forceRegen, setForceRegen] = useState(true);
  const [skipTranscription, setSkipTranscription] = useState(false);

  // Duration (shared)
  const [maxDurationMin, setMaxDurationMin] = useState(20);
  const [minDurationMin, setMinDurationMin] = useState(8);

  // Anti-duplication
  const [antiDuplicate, setAntiDuplicate] = useState(true);
  const [antiDuplicateMode, setAntiDuplicateMode] = useState<AntiDuplicateMode>('subtle');
  // Speed sub-config
  const [speedEnabled, setSpeedEnabled] = useState(true);
  const [speedFactor, setSpeedFactor] = useState(1.02);
  // Visual sub-config
  const [cropEnabled, setCropEnabled] = useState(true);
  const [cropPercent, setCropPercent] = useState(2);
  const [zoomEnabled, setZoomEnabled] = useState(false);
  const [zoomAmount, setZoomAmount] = useState(1.03);
  const [colorGrading, setColorGrading] = useState(false);
  const [brightness, setBrightness] = useState(0.03);
  const [saturation, setSaturation] = useState(1.05);
  const [contrast, setContrast] = useState(1.02);

  // Captions
  const [withCaptions, setWithCaptions] = useState(true);
  const [captionStyle, setCaptionStyle] = useState<CaptionStyle>('BOLD');
  const [captionFontSize, setCaptionFontSize] = useState(58);
  const [captionTextColor, setCaptionTextColor] = useState('#FFFFFF');
  const [captionBorderWidth, setCaptionBorderWidth] = useState(4);
  const [captionIsBold, setCaptionIsBold] = useState(true);

  // Overlays / Branding
  const [withOverlays, setWithOverlays] = useState(false);
  const [logoEnabled, setLogoEnabled] = useState(false);
  const [logoPath, setLogoPath] = useState('');
  const [logoPosition, setLogoPosition] = useState<LogoPosition>('BOTTOM_RIGHT');
  const [logoOpacity, setLogoOpacity] = useState(0.8);

  // Upload options
  const [autoUpload, setAutoUpload] = useState(false);
  const [privacy, setPrivacy] = useState<'public' | 'unlisted' | 'private'>('unlisted');

  // Preview state
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [isGeneratingPreview, setIsGeneratingPreview] = useState(false);
  const [previewError, setPreviewError] = useState<string | null>(null);

  // Background download task from dispatcher
  const downloadTask = useDispatcherStore((s) =>
    run?.id != null ? s.tasks[`download:${run.id}`] : undefined,
  );
  const autoTriggeredPreview = useRef(false);

  useEffect(() => {
    if (channels.length === 0) {
      fetchChannels(goUrl);
    }
  }, [channels.length, fetchChannels, goUrl]);

  useEffect(() => {
    setCanAdvance(mode !== null);
  }, [mode, setCanAdvance]);

  // Auto-trigger preview when background download finishes
  useEffect(() => {
    if (
      downloadTask?.status === 'done' &&
      run?.id &&
      !previewUrl &&
      !isGeneratingPreview &&
      !autoTriggeredPreview.current
    ) {
      autoTriggeredPreview.current = true;
      void handleUpdatePreview();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [downloadTask?.status, run?.id]);

  // Sync current config to wizard store so PipelineWizardLayout can PATCH the run
  useEffect(() => {
    if (!mode) return;
    const config: ModeConfig = {
      mode,
      sensitivityPct,
      skipMusicMinutes,
      forceRegen,
      skipTranscription,
      minDurationSecs: minDurationMin * 60,
      maxDurationSecs: maxDurationMin * 60,
      antiDuplicate,
      antiDuplicateMode,
      speed: { enabled: speedEnabled, factor: speedFactor },
      visual: {
        cropEnabled,
        cropPercent,
        zoomEnabled,
        zoomAmount,
        colorGrading,
        brightness,
        saturation,
        contrast,
      },
      withCaptions,
      captionStyle: {
        style: captionStyle,
        fontSize: captionFontSize,
        textColor: captionTextColor,
        borderWidth: captionBorderWidth,
        isBold: captionIsBold,
      },
      withOverlays,
      branding: {
        logoEnabled,
        logoPath,
        logoPosition,
        logoOpacity,
      },
      autoUpload,
      privacy,
      channelId: selectedChannelId ?? undefined,
    };
    setPendingModeConfig(config);
  }, [
    mode, sensitivityPct, skipMusicMinutes, forceRegen, skipTranscription,
    minDurationMin, maxDurationMin,
    antiDuplicate, antiDuplicateMode,
    speedEnabled, speedFactor,
    cropEnabled, cropPercent, zoomEnabled, zoomAmount, colorGrading, brightness, saturation, contrast,
    withCaptions, captionStyle, captionFontSize, captionTextColor, captionBorderWidth, captionIsBold,
    withOverlays, logoEnabled, logoPath, logoPosition, logoOpacity,
    autoUpload, privacy, selectedChannelId,
    setPendingModeConfig,
  ]);

  async function handleUpdatePreview() {
    if (!run?.id) return;
    setIsGeneratingPreview(true);
    setPreviewError(null);
    try {
      const res = await fetch(`${goUrl}/api/pipeline/runs/${run.id}/preview`, { method: 'POST' });
      if (!res.ok) {
        const err = await res.json() as { message?: string };
        throw new Error(err.message ?? 'Erro ao gerar preview');
      }
      const data = await res.json() as { preview_url: string };
      setPreviewUrl(`${goUrl}${data.preview_url}`);
    } catch (err) {
      setPreviewError(err instanceof Error ? err.message : 'Erro desconhecido');
    } finally {
      setIsGeneratingPreview(false);
    }
  }

  async function handleSavePreset() {
    if (!selectedChannelId) return;
    await fetch(`${goUrl}/api/channels/${selectedChannelId}/config`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        preview_captions_enabled: withCaptions,
        anti_duplicate_enabled: antiDuplicate,
        anti_duplicate_mode: antiDuplicateMode === 'aggressive' ? 'AGGRESSIVE' : 'SUBTLE',
      }),
    });
  }

  function handleModeSelect(selected: WorkflowMode) {
    setMode(selected);
    setStepHidden('highlights', selected !== 'ai');
  }

  function handleChannelSelect(value: string) {
    const id = Number(value);
    selectChannel(id);
    loadChannelConfig(goUrl, id);
  }

  // Apply preset defaults when user switches anti-duplication mode
  function handleAntiDuplicateModeChange(value: AntiDuplicateMode) {
    setAntiDuplicateMode(value);
    if (value === 'aggressive') {
      setSpeedEnabled(true); setSpeedFactor(1.05);
      setCropEnabled(true); setCropPercent(3);
      setZoomEnabled(false);
      setColorGrading(true); setBrightness(0.03); setSaturation(1.05); setContrast(1.02);
    } else {
      setSpeedEnabled(true); setSpeedFactor(1.02);
      setCropEnabled(true); setCropPercent(2);
      setZoomEnabled(false); setColorGrading(false);
    }
  }

  const metadata = run?.stepOutputs?.metadata as VideoMetadata | undefined;

  const antiDuplicateStatus = antiDuplicate
    ? antiDuplicateMode === 'aggressive'
      ? 'Ativado - Modo Agressivo'
      : 'Ativado - Modo Sutil'
    : 'Desativado';

  return (
    <div className="flex flex-col gap-6">
      {/* Video info */}
      {metadata && <MetadataPreview metadata={metadata} compact />}

      {/* Preview do Resultado */}
      <div className="flex flex-col gap-3 rounded-xl border border-border bg-[#12121C] p-4">
        <h3 className="text-sm font-semibold text-[#F0F0F8]">Preview do Resultado</h3>

        {previewUrl ? (
          <VideoPreview src={previewUrl} />
        ) : (
          <div className="flex aspect-video items-center justify-center rounded-lg bg-[#0D0D15] text-sm text-[#5C5C80]">
            {isGeneratingPreview
              ? 'Gerando preview...'
              : downloadTask?.status === 'running'
                ? '⬇ Baixando vídeo... aguarde para gerar preview'
                : run?.id
                  ? 'Clique em "Atualizar Preview" após o download'
                  : 'Preview disponível após executar o download'}
          </div>
        )}

        {previewError && <p className="text-xs text-red-400">{previewError}</p>}

        <div className="flex gap-2">
          <Button
            variant="outline"
            className="flex-1"
            onClick={() => void handleUpdatePreview()}
            disabled={isGeneratingPreview || !run?.id || downloadTask?.status === 'running'}
          >
            ↻ Atualizar Preview
          </Button>
          <Button
            variant="outline"
            className="flex-1"
            onClick={() => void handleSavePreset()}
            disabled={!selectedChannelId}
          >
            💾 Salvar Preset
          </Button>
        </div>
      </div>

      {/* Channel Selector */}
      <div className="flex flex-col gap-2">
        <Label className="text-[#F0F0F8]">Canal</Label>
        <Select
          value={selectedChannelId != null ? String(selectedChannelId) : undefined}
          onValueChange={handleChannelSelect}
        >
          <SelectTrigger className="w-full bg-[#1A1A26] text-[#F0F0F8]">
            <SelectValue placeholder="Selecione um canal" />
          </SelectTrigger>
          <SelectContent>
            {channels.map((ch) => (
              <SelectItem key={ch.ID} value={String(ch.ID)}>
                {ch.Name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <Separator />

      {/* Workflow Mode cards */}
      <div className="flex flex-col gap-3">
        <h3 className="text-sm font-semibold text-[#F0F0F8]">Modo de Trabalho</h3>
        <div className="grid grid-cols-2 gap-4">
          <ModeCard
            icon={<span>✨</span>}
            title="Com IA"
            description="Whisper + Ollama para detectar os melhores momentos automaticamente"
            selected={mode === 'ai'}
            onClick={() => handleModeSelect('ai')}
            features={[
              'Transcrição automática',
              'Detecção de tópicos',
              'Cortes inteligentes',
              'Títulos gerados por IA',
            ]}
          />
          <ModeCard
            icon={<span>▶</span>}
            title="Cortes"
            description="Divide vídeos em partes de 8+ min com thumbnails customizadas PT1, PT2..."
            selected={mode === 'longform'}
            onClick={() => handleModeSelect('longform')}
            features={[
              'Mínimo 8 min por parte',
              'Thumb com gradiente do canal',
              'Texto PT1, PT2, PT3...',
              'Títulos SEO + tags',
            ]}
          />
        </div>
      </div>

      {mode && (
        <>
          <Separator />

          {/* Duration presets (both modes) */}
          <div className="flex flex-col gap-4">
            <DurationChips
              label="⏱ Duração máxima de cada corte"
              presets={MAX_DURATION_PRESETS}
              value={maxDurationMin}
              onChange={setMaxDurationMin}
            />
            <DurationChips
              label="🔔 Duração mínima (ignora cortes menores)"
              presets={MIN_DURATION_PRESETS}
              value={minDurationMin}
              onChange={setMinDurationMin}
            />
          </div>

          {/* AI-specific config */}
          {mode === 'ai' && (
            <>
              <Separator />
              <div className="flex flex-col gap-5">
                <h3 className="text-sm font-semibold text-[#F0F0F8]">Configurações de IA</h3>

                {/* Sensitivity */}
                <div className="flex flex-col gap-3">
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-[#A0A0B8]">
                      Sensibilidade: {sensitivityPct}%
                    </span>
                  </div>
                  <div className="grid grid-cols-3 gap-2">
                    {[
                      { label: 'Liberal (50%)', value: 50 },
                      { label: 'Equilibrado (70%)', value: 70 },
                      { label: 'Muito Seletivo (90%)', value: 90 },
                    ].map(({ label, value }) => (
                      <button
                        key={value}
                        type="button"
                        onClick={() => setSensitivityPct(value)}
                        className={cn(
                          'rounded-md border px-2 py-1.5 text-xs transition-all',
                          sensitivityPct === value
                            ? 'border-[#A0A0B8] bg-[#A0A0B8]/10 text-[#F0F0F8]'
                            : 'border-border bg-[#12121C] text-[#5C5C80]',
                        )}
                      >
                        {label}
                      </button>
                    ))}
                  </div>
                  <Slider
                    min={50}
                    max={90}
                    step={1}
                    value={[sensitivityPct]}
                    onValueChange={([v]) => setSensitivityPct(v)}
                  />
                  <p className="text-xs text-[#5C5C80]">
                    💡 Maior = mais seletivo (menos cortes de melhor qualidade)
                  </p>
                </div>

                {/* Pular música inicial */}
                <div className="flex flex-col gap-2">
                  <span className="text-sm text-[#A0A0B8]">▶▶ Pular música inicial</span>
                  <div className="flex flex-wrap gap-2">
                    {SKIP_MUSIC_PRESETS.map((min) => (
                      <button
                        key={min}
                        type="button"
                        onClick={() => setSkipMusicMinutes(min)}
                        className={cn(
                          'rounded-md border px-3 py-1.5 text-sm transition-all',
                          skipMusicMinutes === min
                            ? 'border-[#F0A040] bg-[#F0A040]/10 text-[#F0A040]'
                            : 'border-border bg-[#12121C] text-[#A0A0B8] hover:border-[#A0A0B8]/50',
                        )}
                      >
                        {min === 0 ? 'Nenhum' : `${min} min`}
                      </button>
                    ))}
                  </div>
                  <p className="text-xs text-[#5C5C80]">
                    ↩↩ Útil para lives que começam com música de espera
                  </p>
                </div>

                {/* Force regen */}
                <div className="flex items-start gap-2">
                  <Checkbox
                    id="force-regen"
                    checked={forceRegen}
                    onCheckedChange={(v) => setForceRegen(v === true)}
                    className="mt-0.5"
                  />
                  <div className="flex flex-col gap-0.5">
                    <Label htmlFor="force-regen" className="text-sm text-[#F0F0F8]">
                      Forçar regeneração de análise
                    </Label>
                    <p className="text-xs text-[#5C5C80]">
                      Sempre reprocessar com Ollama (ignora cache)
                    </p>
                  </div>
                </div>

                {/* Skip transcription */}
                <div className="flex items-start gap-2">
                  <Checkbox
                    id="skip-transcription"
                    checked={skipTranscription}
                    onCheckedChange={(v) => setSkipTranscription(v === true)}
                    className="mt-0.5"
                  />
                  <div className="flex flex-col gap-0.5">
                    <Label htmlFor="skip-transcription" className="text-sm text-[#F0F0F8]">
                      Pular transcrição (usar cache)
                    </Label>
                    <p className="text-xs text-[#5C5C80]">
                      Não executar Whisper (requer transcrição prévia)
                    </p>
                  </div>
                </div>
              </div>
            </>
          )}

          <Separator />

          {/* Video options */}
          <div className="flex flex-col gap-3">
            <h3 className="text-sm font-semibold text-[#F0F0F8]">Opções de Vídeo</h3>

            {/* Anti-duplication with expanded sub-config */}
            <ToggleSection
              icon="🛡"
              title="Proteção Anti-Duplicação"
              statusText={antiDuplicateStatus}
              enabled={antiDuplicate}
              onToggle={setAntiDuplicate}
            >
              <div className="flex flex-col gap-4">
                {/* Mode selector */}
                <div className="flex flex-col gap-2">
                  <span className="text-xs text-[#A0A0B8]">Modo</span>
                  <div className="flex gap-2">
                    {(
                      [
                        { value: 'subtle', label: 'Sutil' },
                        { value: 'aggressive', label: 'Agressivo' },
                      ] as const
                    ).map(({ value, label }) => (
                      <button
                        key={value}
                        type="button"
                        onClick={() => handleAntiDuplicateModeChange(value)}
                        className={cn(
                          'rounded-md border px-3 py-1.5 text-sm transition-all',
                          antiDuplicateMode === value
                            ? 'border-[#6BCB8B] bg-[#6BCB8B]/10 text-[#6BCB8B]'
                            : 'border-border bg-[#12121C] text-[#A0A0B8]',
                        )}
                      >
                        {label}
                      </button>
                    ))}
                  </div>
                </div>

                <Separator />

                {/* Speed sub-config */}
                <div className="flex flex-col gap-2">
                  <span className="text-xs font-medium text-[#A0A0B8]">Velocidade</span>
                  <div className="flex items-start gap-2">
                    <Checkbox
                      id="speed-enabled"
                      checked={speedEnabled}
                      onCheckedChange={(v) => setSpeedEnabled(v === true)}
                      className="mt-0.5"
                    />
                    <div className="flex flex-1 flex-col gap-1.5">
                      <div className="flex items-center justify-between">
                        <Label htmlFor="speed-enabled" className="text-xs text-[#F0F0F8]">
                          Aumentar velocidade
                        </Label>
                        <span className="text-xs text-[#A0A0B8]">{speedFactor.toFixed(2)}x</span>
                      </div>
                      {speedEnabled && (
                        <Slider
                          min={100}
                          max={110}
                          step={1}
                          value={[Math.round(speedFactor * 100)]}
                          onValueChange={([v]) => setSpeedFactor(v / 100)}
                        />
                      )}
                    </div>
                  </div>
                </div>

                {/* Visual sub-config */}
                <div className="flex flex-col gap-2">
                  <span className="text-xs font-medium text-[#A0A0B8]">Visual</span>

                  {/* Crop */}
                  <div className="flex items-start gap-2">
                    <Checkbox
                      id="crop-enabled"
                      checked={cropEnabled}
                      onCheckedChange={(v) => setCropEnabled(v === true)}
                      className="mt-0.5"
                    />
                    <div className="flex flex-1 flex-col gap-1.5">
                      <div className="flex items-center justify-between">
                        <Label htmlFor="crop-enabled" className="text-xs text-[#F0F0F8]">
                          Corte de bordas
                        </Label>
                        <span className="text-xs text-[#A0A0B8]">{cropPercent}%</span>
                      </div>
                      {cropEnabled && (
                        <Slider
                          min={1}
                          max={5}
                          step={0.5}
                          value={[cropPercent]}
                          onValueChange={([v]) => setCropPercent(v)}
                        />
                      )}
                    </div>
                  </div>

                  {/* Zoom */}
                  <div className="flex items-start gap-2">
                    <Checkbox
                      id="zoom-enabled"
                      checked={zoomEnabled}
                      onCheckedChange={(v) => setZoomEnabled(v === true)}
                      className="mt-0.5"
                    />
                    <div className="flex flex-1 flex-col gap-1.5">
                      <div className="flex items-center justify-between">
                        <Label htmlFor="zoom-enabled" className="text-xs text-[#F0F0F8]">
                          Zoom suave
                        </Label>
                        <span className="text-xs text-[#A0A0B8]">{zoomAmount.toFixed(2)}x</span>
                      </div>
                      {zoomEnabled && (
                        <Slider
                          min={101}
                          max={110}
                          step={1}
                          value={[Math.round(zoomAmount * 100)]}
                          onValueChange={([v]) => setZoomAmount(v / 100)}
                        />
                      )}
                    </div>
                  </div>

                  {/* Color grading */}
                  <div className="flex items-start gap-2">
                    <Checkbox
                      id="color-grading"
                      checked={colorGrading}
                      onCheckedChange={(v) => setColorGrading(v === true)}
                      className="mt-0.5"
                    />
                    <Label htmlFor="color-grading" className="text-xs text-[#F0F0F8]">
                      Gradação de cor
                    </Label>
                  </div>
                  {colorGrading && (
                    <div className="ml-6 flex flex-col gap-2">
                      <div className="flex items-center justify-between">
                        <span className="text-xs text-[#A0A0B8]">Brilho</span>
                        <span className="text-xs text-[#F0F0F8]">
                          {brightness > 0 ? `+${brightness.toFixed(2)}` : brightness.toFixed(2)}
                        </span>
                      </div>
                      <Slider
                        min={-10}
                        max={10}
                        step={1}
                        value={[Math.round(brightness * 100)]}
                        onValueChange={([v]) => setBrightness(v / 100)}
                      />
                      <div className="flex items-center justify-between">
                        <span className="text-xs text-[#A0A0B8]">Saturação</span>
                        <span className="text-xs text-[#F0F0F8]">{saturation.toFixed(2)}x</span>
                      </div>
                      <Slider
                        min={80}
                        max={120}
                        step={1}
                        value={[Math.round(saturation * 100)]}
                        onValueChange={([v]) => setSaturation(v / 100)}
                      />
                    </div>
                  )}
                </div>
              </div>
            </ToggleSection>

            {/* Video overlay */}
            <ToggleSection
              icon="▶"
              title="Overlay de Vídeo"
              statusText={withOverlays ? 'Ativado' : 'Desativado'}
              enabled={withOverlays}
              onToggle={setWithOverlays}
            />

            {/* Captions with style sub-config */}
            <ToggleSection
              icon="T↑"
              title="Legendas / Captions"
              statusText={withCaptions ? `Ativado - ${captionStyle}` : 'Desativado'}
              enabled={withCaptions}
              onToggle={setWithCaptions}
            >
              <div className="flex flex-col gap-3">
                {/* Style picker */}
                <div className="flex flex-col gap-2">
                  <span className="text-xs text-[#A0A0B8]">Estilo</span>
                  <div className="flex gap-2">
                    {(
                      [
                        { value: 'BOLD', label: 'Negrito' },
                        { value: 'HIGHLIGHT', label: 'Destaque' },
                        { value: 'SUBTITLE', label: 'Legenda' },
                      ] as const
                    ).map(({ value, label }) => (
                      <button
                        key={value}
                        type="button"
                        onClick={() => setCaptionStyle(value)}
                        className={cn(
                          'rounded-md border px-3 py-1.5 text-sm transition-all',
                          captionStyle === value
                            ? 'border-[#00D4FF] bg-[#00D4FF]/10 text-[#00D4FF]'
                            : 'border-border bg-[#12121C] text-[#A0A0B8]',
                        )}
                      >
                        {label}
                      </button>
                    ))}
                  </div>
                </div>

                {/* Font size */}
                <div className="flex flex-col gap-1.5">
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-[#A0A0B8]">Tamanho</span>
                    <span className="text-xs text-[#F0F0F8]">{captionFontSize}px</span>
                  </div>
                  <Slider
                    min={36}
                    max={72}
                    step={2}
                    value={[captionFontSize]}
                    onValueChange={([v]) => setCaptionFontSize(v)}
                  />
                </div>

                {/* Color */}
                <div className="flex items-center gap-4">
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-[#A0A0B8]">Cor</span>
                    <input
                      type="color"
                      value={captionTextColor}
                      onChange={(e) => setCaptionTextColor(e.target.value)}
                      className="h-7 w-7 cursor-pointer rounded border border-border bg-transparent"
                    />
                    <span className="text-xs text-[#F0F0F8]">{captionTextColor}</span>
                  </div>
                  <div className="flex flex-1 flex-col gap-1">
                    <div className="flex items-center justify-between">
                      <span className="text-xs text-[#A0A0B8]">Borda</span>
                      <span className="text-xs text-[#F0F0F8]">{captionBorderWidth}px</span>
                    </div>
                    <Slider
                      min={0}
                      max={8}
                      step={1}
                      value={[captionBorderWidth]}
                      onValueChange={([v]) => setCaptionBorderWidth(v)}
                    />
                  </div>
                </div>

                {/* Bold toggle */}
                <div className="flex items-center gap-2">
                  <Checkbox
                    id="caption-bold"
                    checked={captionIsBold}
                    onCheckedChange={(v) => setCaptionIsBold(v === true)}
                  />
                  <Label htmlFor="caption-bold" className="text-xs text-[#F0F0F8]">
                    Negrito
                  </Label>
                </div>
              </div>
            </ToggleSection>

            {/* Branding / Logo */}
            <ToggleSection
              icon="🎨"
              title="Identidade Visual (Logo)"
              statusText={
                logoEnabled
                  ? `Ativado - ${logoPosition.replace('_', ' ').toLowerCase()}`
                  : 'Desativado'
              }
              enabled={logoEnabled}
              onToggle={setLogoEnabled}
            >
              <div className="flex flex-col gap-3">
                {/* Logo path */}
                <div className="flex flex-col gap-1.5">
                  <span className="text-xs text-[#A0A0B8]">Caminho do Logo</span>
                  <input
                    type="text"
                    value={logoPath}
                    onChange={(e) => setLogoPath(e.target.value)}
                    placeholder="/caminho/para/logo.png"
                    className="w-full rounded-md border border-border bg-[#1A1A26] px-3 py-1.5 text-sm text-[#F0F0F8] placeholder:text-[#5C5C80] focus:outline-none focus:ring-1 focus:ring-[#00D4FF]/50"
                  />
                </div>

                {/* Position picker */}
                <div className="flex flex-col gap-2">
                  <span className="text-xs text-[#A0A0B8]">Posição</span>
                  <div className="grid grid-cols-2 gap-1.5">
                    {(
                      [
                        { value: 'TOP_LEFT', label: '↖ Superior Esq' },
                        { value: 'TOP_RIGHT', label: '↗ Superior Dir' },
                        { value: 'BOTTOM_LEFT', label: '↙ Inferior Esq' },
                        { value: 'BOTTOM_RIGHT', label: '↘ Inferior Dir' },
                      ] as const
                    ).map(({ value, label }) => (
                      <button
                        key={value}
                        type="button"
                        onClick={() => setLogoPosition(value)}
                        className={cn(
                          'rounded-md border px-2 py-1.5 text-xs transition-all',
                          logoPosition === value
                            ? 'border-[#00D4FF] bg-[#00D4FF]/10 text-[#00D4FF]'
                            : 'border-border bg-[#12121C] text-[#A0A0B8]',
                        )}
                      >
                        {label}
                      </button>
                    ))}
                  </div>
                </div>

                {/* Opacity */}
                <div className="flex flex-col gap-1.5">
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-[#A0A0B8]">Opacidade</span>
                    <span className="text-xs text-[#F0F0F8]">{Math.round(logoOpacity * 100)}%</span>
                  </div>
                  <Slider
                    min={10}
                    max={100}
                    step={5}
                    value={[Math.round(logoOpacity * 100)]}
                    onValueChange={([v]) => setLogoOpacity(v / 100)}
                  />
                </div>
              </div>
            </ToggleSection>
          </div>

          <Separator />

          {/* Upload options */}
          <div className="flex flex-col gap-4">
            <h3 className="text-sm font-semibold text-[#F0F0F8]">Opções de Upload</h3>

            <div className="flex items-center gap-2">
              <Checkbox
                id="auto-upload"
                checked={autoUpload}
                onCheckedChange={(v) => setAutoUpload(v === true)}
              />
              <Label htmlFor="auto-upload" className="text-[#A0A0B8]">
                Upload automático
              </Label>
            </div>

            <div className="flex flex-col gap-2">
              <Label className="text-[#A0A0B8]">Privacidade</Label>
              <Select value={privacy} onValueChange={(v) => setPrivacy(v as typeof privacy)}>
                <SelectTrigger className="w-full bg-[#1A1A26] text-[#F0F0F8]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="public">Public</SelectItem>
                  <SelectItem value="unlisted">Unlisted</SelectItem>
                  <SelectItem value="private">Private</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
