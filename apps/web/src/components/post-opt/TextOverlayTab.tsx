'use client';

import { useState } from 'react';
import { ChevronDown, ChevronUp, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Checkbox } from '@/components/ui/checkbox';
import { Switch } from '@/components/ui/switch';
import { PositionGrid } from './PositionGrid';
import { TextStyleEditorPanel } from './TextStyleEditorPanel';
import {
  DEFAULT_OVERLAY,
  type TextOverlayConfig,
  type TimedTextOverlay,
} from '@/types/text-overlay';

export function TextOverlayTab() {
  const [config, setConfig] = useState<TextOverlayConfig>({
    enabled: false,
    overlays: [],
  });
  const [expandedIndex, setExpandedIndex] = useState<number | null>(null);

  const updateOverlay = (index: number, patch: Partial<TimedTextOverlay>) => {
    setConfig((prev) => ({
      ...prev,
      overlays: prev.overlays.map((o, i) => (i === index ? { ...o, ...patch } : o)),
    }));
  };

  const addOverlay = () => {
    const nextIndex = config.overlays.length;
    setConfig((prev) => ({
      ...prev,
      overlays: [...prev.overlays, { ...DEFAULT_OVERLAY, style: { ...DEFAULT_OVERLAY.style } }],
    }));
    setExpandedIndex(nextIndex);
  };

  const removeOverlay = (index: number) => {
    setConfig((prev) => ({
      ...prev,
      overlays: prev.overlays.filter((_, i) => i !== index),
    }));
    setExpandedIndex((prev) => {
      if (prev === null) return null;
      if (prev === index) return null;
      if (prev > index) return prev - 1;
      return prev;
    });
  };

  return (
    <div className="flex flex-col gap-4">
      {/* Master toggle */}
      <div className="rounded-lg border border-zinc-800 bg-zinc-900/60 p-4 flex items-center justify-between gap-4">
        <div className="flex flex-col gap-0.5">
          <span className="text-sm font-medium text-zinc-200">Adicionar Overlays de Texto</span>
          <span className="text-xs text-zinc-500">Textos permanentes ou temporários sobre o vídeo</span>
        </div>
        <Switch
          checked={config.enabled}
          onCheckedChange={(v) => setConfig((prev) => ({ ...prev, enabled: v }))}
          aria-label="Adicionar Overlays de Texto"
        />
      </div>

      {config.enabled && (
        <>
          {/* Overlay list */}
          {config.overlays.map((overlay, index) => {
            const isExpanded = expandedIndex === index;
            return (
              <div
                key={index}
                className="rounded-lg border border-zinc-800 bg-zinc-900/40 p-3 flex flex-col gap-3"
              >
                {/* Header row */}
                <div className="flex items-center gap-2">
                  <Input
                    value={overlay.text}
                    onChange={(e) => updateOverlay(index, { text: e.target.value })}
                    className="flex-1 h-9"
                    aria-label="Texto do overlay"
                  />
                  <button
                    type="button"
                    onClick={() => setExpandedIndex(isExpanded ? null : index)}
                    className="h-9 w-9 rounded border border-zinc-700 bg-zinc-800 hover:bg-zinc-700 flex items-center justify-center text-zinc-400"
                    aria-label={isExpanded ? 'Recolher overlay' : 'Expandir overlay'}
                  >
                    {isExpanded ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
                  </button>
                  <button
                    type="button"
                    onClick={() => removeOverlay(index)}
                    className="h-9 w-9 rounded border border-red-900/40 bg-red-950/30 hover:bg-red-900/50 flex items-center justify-center text-red-400"
                    aria-label="Remover overlay"
                  >
                    <Trash2 size={14} />
                  </button>
                </div>

                {/* Expanded: style editor + timing + position */}
                {isExpanded && (
                  <div className="flex flex-col gap-4 pt-1">
                    <TextStyleEditorPanel
                      config={overlay.style}
                      onConfigChange={(style) => updateOverlay(index, { style })}
                      showBackgroundOptions
                    />

                    {/* Apply to whole video */}
                    <div className="flex items-center gap-2">
                      <Checkbox
                        id={`whole-${index}`}
                        checked={overlay.applyToWholeVideo}
                        onCheckedChange={(v) =>
                          updateOverlay(index, { applyToWholeVideo: v === true })
                        }
                      />
                      <label
                        htmlFor={`whole-${index}`}
                        className="text-sm text-zinc-300 cursor-pointer select-none"
                      >
                        Aplicar no vídeo todo
                      </label>
                    </div>

                    {/* Start / End time */}
                    {!overlay.applyToWholeVideo && (
                      <div className="flex gap-3">
                        <div className="flex flex-col gap-1.5 flex-1">
                          <Label className="text-xs text-zinc-400">Início (s)</Label>
                          <Input
                            type="number"
                            min={0}
                            value={overlay.startTime}
                            onChange={(e) =>
                              updateOverlay(index, { startTime: parseFloat(e.target.value) || 0 })
                            }
                            className="h-9"
                          />
                        </div>
                        <div className="flex flex-col gap-1.5 flex-1">
                          <Label className="text-xs text-zinc-400">Fim (s)</Label>
                          <Input
                            type="number"
                            min={0}
                            placeholder="Até o fim"
                            value={overlay.endTime ?? ''}
                            onChange={(e) =>
                              updateOverlay(index, {
                                endTime: e.target.value ? parseFloat(e.target.value) : null,
                              })
                            }
                            className="h-9"
                          />
                        </div>
                      </div>
                    )}

                    {/* Position */}
                    <div className="flex flex-col gap-2">
                      <Label className="text-xs text-zinc-400">Posição:</Label>
                      <PositionGrid
                        value={overlay.position}
                        onChange={(pos) => updateOverlay(index, { position: pos as TimedTextOverlay['position'] })}
                      />
                    </div>
                  </div>
                )}
              </div>
            );
          })}

          {/* Add button */}
          <button
            type="button"
            onClick={addOverlay}
            className="w-full h-11 rounded-lg border border-zinc-700 bg-zinc-900 hover:bg-zinc-800 text-zinc-300 text-sm font-medium transition-colors"
          >
            + Adicionar Texto
          </button>

          {/* Apply — disabled until backend exists */}
          <div className="relative group self-start">
            <Button disabled className="opacity-40 cursor-not-allowed">
              Apply Text Overlay
            </Button>
            <div className="absolute bottom-full mb-1 left-1/2 -translate-x-1/2 hidden group-hover:block bg-zinc-800 text-zinc-300 text-xs rounded px-2 py-1 whitespace-nowrap pointer-events-none">
              Backend em desenvolvimento
            </div>
          </div>
        </>
      )}
    </div>
  );
}
