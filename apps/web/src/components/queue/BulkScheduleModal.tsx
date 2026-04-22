'use client';

import { useState, useMemo } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { createLogger } from '@/lib/logger';

const log = createLogger('BulkScheduleModal');

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  itemCount: number;
  onConfirm: (startAt: string, intervalMinutes: number) => Promise<void>;
}

const INTERVAL_PRESETS: { label: string; minutes: number }[] = [
  { label: '1h', minutes: 60 },
  { label: '6h', minutes: 360 },
  { label: '12h', minutes: 720 },
  { label: '24h', minutes: 1440 },
  { label: '48h', minutes: 2880 },
];

export function BulkScheduleModal({ open, onOpenChange, itemCount, onConfirm }: Props) {
  const [startAt, setStartAt] = useState('');
  const [selectedPreset, setSelectedPreset] = useState<number | null>(1440);
  const [customMinutes, setCustomMinutes] = useState('');
  const [loading, setLoading] = useState(false);

  const intervalMinutes = useMemo(() => {
    if (selectedPreset !== null) return selectedPreset;
    const parsed = parseInt(customMinutes, 10);
    return Number.isFinite(parsed) && parsed > 0 ? parsed : null;
  }, [selectedPreset, customMinutes]);

  const previewText = useMemo(() => {
    if (!startAt || !intervalMinutes || itemCount === 0) return null;
    const start = new Date(startAt);
    if (isNaN(start.getTime())) return null;
    const lastMs = start.getTime() + (itemCount - 1) * intervalMinutes * 60 * 1000;
    const last = new Date(lastMs);
    return `${itemCount} video${itemCount !== 1 ? 's' : ''} starting ${start.toLocaleString()}, every ${intervalMinutes >= 60 ? `${intervalMinutes / 60}h` : `${intervalMinutes}m`} → last upload on ${last.toLocaleString()}`;
  }, [startAt, intervalMinutes, itemCount]);

  const handleConfirm = async () => {
    if (!startAt || !intervalMinutes) return;
    setLoading(true);
    try {
      await onConfirm(new Date(startAt).toISOString(), intervalMinutes);
      onOpenChange(false);
      setStartAt('');
      setSelectedPreset(1440);
      setCustomMinutes('');
    } catch (err) {
      log.error('bulk schedule failed', { err });
    } finally {
      setLoading(false);
    }
  };

  const isValid = Boolean(startAt && intervalMinutes && !isNaN(new Date(startAt).getTime()));

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Bulk Schedule</DialogTitle>
        </DialogHeader>

        <div className="flex flex-col gap-5 py-2">
          {/* Start date */}
          <div className="flex flex-col gap-1.5">
            <label className="text-xs font-medium text-caption">Start date &amp; time</label>
            <input
              type="datetime-local"
              value={startAt}
              onChange={(e) => setStartAt(e.target.value)}
              className="rounded-lg bg-surface border border-border px-3 py-2 text-sm text-heading focus:outline-none focus:border-brand/60"
            />
          </div>

          {/* Interval */}
          <div className="flex flex-col gap-2">
            <label className="text-xs font-medium text-caption">Interval between uploads</label>
            <div className="flex flex-wrap gap-2">
              {INTERVAL_PRESETS.map((preset) => (
                <button
                  key={preset.minutes}
                  type="button"
                  onClick={() => { setSelectedPreset(preset.minutes); setCustomMinutes(''); }}
                  className={`rounded-lg border px-3 py-1.5 text-xs font-semibold transition-colors ${
                    selectedPreset === preset.minutes
                      ? 'bg-brand/15 border-brand/50 text-brand'
                      : 'bg-surface border-border text-caption hover:border-brand/30'
                  }`}
                >
                  {preset.label}
                </button>
              ))}
              <button
                type="button"
                onClick={() => setSelectedPreset(null)}
                className={`rounded-lg border px-3 py-1.5 text-xs font-semibold transition-colors ${
                  selectedPreset === null
                    ? 'bg-brand/15 border-brand/50 text-brand'
                    : 'bg-surface border-border text-caption hover:border-brand/30'
                }`}
              >
                Custom
              </button>
            </div>

            {selectedPreset === null && (
              <div className="flex items-center gap-2">
                <input
                  type="number"
                  min="1"
                  value={customMinutes}
                  onChange={(e) => setCustomMinutes(e.target.value)}
                  placeholder="e.g. 90"
                  className="w-28 rounded-lg bg-surface border border-border px-3 py-1.5 text-sm text-heading focus:outline-none focus:border-brand/60"
                />
                <span className="text-xs text-subtle">minutes</span>
              </div>
            )}
          </div>

          {/* Preview */}
          {previewText ? (
            <div className="rounded-lg bg-surface border border-border px-3 py-2.5">
              <p className="text-xs text-caption leading-relaxed">{previewText}</p>
            </div>
          ) : (
            <div className="rounded-lg bg-surface border border-border px-3 py-2.5">
              <p className="text-xs text-subtle">
                Fill in start date and interval to see preview
              </p>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            size="sm"
            onClick={() => onOpenChange(false)}
            disabled={loading}
          >
            Cancel
          </Button>
          <Button
            size="sm"
            disabled={!isValid || loading}
            onClick={() => void handleConfirm()}
          >
            {loading ? 'Scheduling…' : 'Confirm'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
