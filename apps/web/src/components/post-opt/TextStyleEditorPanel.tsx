'use client';

import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { Slider } from '@/components/ui/slider';
import { Switch } from '@/components/ui/switch';
import { ColorInputField } from './ColorInputField';
import type { TextStyleConfig } from '@/types/text-overlay';

interface Props {
  config: TextStyleConfig;
  onConfigChange: (c: TextStyleConfig) => void;
  showBackgroundOptions?: boolean;
}

function SettingSwitch({
  title,
  description,
  checked,
  onCheckedChange,
  ariaLabel,
}: {
  title: string;
  description?: string;
  checked: boolean;
  onCheckedChange: (v: boolean) => void;
  ariaLabel?: string;
}) {
  return (
    <div className="flex items-center justify-between gap-4">
      <div className="flex flex-col gap-0.5 min-w-0">
        <span className="text-sm text-foreground">{title}</span>
        {description && <span className="text-xs text-subtle">{description}</span>}
      </div>
      <Switch
        checked={checked}
        onCheckedChange={onCheckedChange}
        aria-label={ariaLabel ?? title}
      />
    </div>
  );
}

export function TextStyleEditorPanel({
  config,
  onConfigChange,
  showBackgroundOptions = true,
}: Props) {
  const update = (patch: Partial<TextStyleConfig>) =>
    onConfigChange({ ...config, ...patch });

  const hasBorder = config.borderWidth > 0 && config.borderColor !== null;
  const hasShadow = config.shadowOffset > 0 && config.shadowColor !== null;
  const hasBackground = config.backgroundColor !== null;

  return (
    <div className="flex flex-col gap-4">
      {/* ── Tipografia ── */}
      <div className="rounded-lg border border-border bg-card/60 p-4 flex flex-col gap-3">
        <span className="text-xs font-medium text-subtle uppercase tracking-wide">Tipografia</span>

        <div className="flex flex-col gap-1">
          <Label className="text-xs font-medium text-subtle">Família da Fonte</Label>
          <Input
            value={config.fontFamily}
            onChange={(e) => update({ fontFamily: e.target.value })}
            placeholder="Arial"
            className="h-9"
          />
        </div>

        <div className="grid grid-cols-2 gap-3">
          <SettingSwitch
            title="Negrito"
            checked={config.isBold}
            onCheckedChange={(v) => update({ isBold: v })}
          />
          <SettingSwitch
            title="Itálico"
            checked={config.isItalic}
            onCheckedChange={(v) => update({ isItalic: v })}
          />
        </div>

        <SettingSwitch
          title="CAIXA ALTA"
          description="Forçar texto em maiúsculas"
          checked={config.allCaps}
          onCheckedChange={(v) => update({ allCaps: v })}
        />

        <div className="flex flex-col gap-2">
          <Label className="text-xs font-medium text-subtle">Tamanho: {config.fontSize}px</Label>
          <Slider
            min={12}
            max={200}
            step={1}
            value={[config.fontSize]}
            onValueChange={([v]) => update({ fontSize: v })}
          />
        </div>
      </div>

      {/* ── Cores e Bordas ── */}
      <div className="rounded-lg border border-border bg-card/60 p-4 flex flex-col gap-4">
        <span className="text-xs font-medium text-subtle uppercase tracking-wide">Cores e Bordas</span>

        <ColorInputField
          label="Cor do Texto"
          value={config.textColor}
          onValueChange={(v) => update({ textColor: v })}
        />

        {showBackgroundOptions && (
          <>
            <SettingSwitch
              title="Cor de Fundo"
              description="Adicionar fundo ao texto"
              checked={hasBackground}
              onCheckedChange={(enabled) =>
                update({ backgroundColor: enabled ? '#000000AA' : null })
              }
            />
            {hasBackground && (
              <div className="flex flex-col gap-3 pl-2 border-l border-border">
                <ColorInputField
                  label="Cor do Fundo"
                  value={config.backgroundColor ?? '#000000AA'}
                  onValueChange={(v) => update({ backgroundColor: v })}
                />
                <div className="flex flex-col gap-1.5">
                  <Label className="text-xs font-medium text-subtle">Padding: {config.padding}px</Label>
                  <Slider
                    min={0} max={100} step={1}
                    value={[config.padding]}
                    onValueChange={([v]) => update({ padding: v })}
                  />
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label className="text-xs font-medium text-subtle">Arredondamento: {config.cornerRadius}px</Label>
                  <Slider
                    min={0} max={100} step={1}
                    value={[config.cornerRadius]}
                    onValueChange={([v]) => update({ cornerRadius: v })}
                  />
                </div>
              </div>
            )}
          </>
        )}

        <SettingSwitch
          title="Borda (Outline)"
          description="Adicionar contorno ao texto"
          checked={hasBorder}
          onCheckedChange={(enabled) =>
            update(enabled
              ? { borderWidth: 2, borderColor: '#000000' }
              : { borderWidth: 0 })
          }
        />
        {hasBorder && (
          <div className="flex flex-col gap-3 pl-2 border-l border-border">
            <ColorInputField
              label="Cor da Borda"
              value={config.borderColor ?? '#000000'}
              onValueChange={(v) => update({ borderColor: v })}
            />
            <div className="flex flex-col gap-1.5">
              <Label className="text-xs font-medium text-subtle">Espessura: {config.borderWidth}px</Label>
              <Slider
                min={1} max={20} step={1}
                value={[config.borderWidth]}
                onValueChange={([v]) => update({ borderWidth: v })}
              />
            </div>
          </div>
        )}

        <SettingSwitch
          title="Sombra"
          description="Adicionar sombra projetada"
          checked={hasShadow}
          onCheckedChange={(enabled) =>
            update(enabled
              ? { shadowOffset: 2, shadowColor: '#000000' }
              : { shadowOffset: 0 })
          }
        />
        {hasShadow && (
          <div className="flex flex-col gap-3 pl-2 border-l border-border">
            <ColorInputField
              label="Cor da Sombra"
              value={config.shadowColor ?? '#000000'}
              onValueChange={(v) => update({ shadowColor: v })}
            />
            <div className="flex flex-col gap-1.5">
              <Label className="text-xs font-medium text-subtle">Distância: {config.shadowOffset}px</Label>
              <Slider
                min={1} max={50} step={1}
                value={[config.shadowOffset]}
                onValueChange={([v]) => update({ shadowOffset: v })}
              />
            </div>
          </div>
        )}
      </div>

      {/* ── Ajuste Fino de Posição ── */}
      <div className="rounded-lg border border-border bg-card/60 p-4 flex flex-col gap-3">
        <span className="text-xs font-medium text-subtle uppercase tracking-wide">Ajuste Fino de Posição</span>
        <div className="flex items-center justify-between gap-3">
          <Label className="text-sm text-prose">Offset Vertical (Y):</Label>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => update({ verticalOffset: Math.max(-1000, config.verticalOffset - 1) })}
              className="h-8 w-8 rounded bg-surface hover:bg-surface/80 text-prose text-lg font-bold flex items-center justify-center"
            >
              −
            </button>
            <input
              type="number"
              min={-1000}
              max={1000}
              value={config.verticalOffset}
              onChange={(e) => update({ verticalOffset: parseInt(e.target.value, 10) || 0 })}
              className="w-16 h-8 rounded border border-border bg-background text-center text-sm text-foreground [appearance:textfield]"
            />
            <button
              type="button"
              onClick={() => update({ verticalOffset: Math.min(1000, config.verticalOffset + 1) })}
              className="h-8 w-8 rounded bg-surface hover:bg-surface/80 text-prose text-lg font-bold flex items-center justify-center"
            >
              +
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
