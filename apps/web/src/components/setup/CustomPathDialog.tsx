'use client';

import { useState } from 'react';
import { useAppStore } from '@/store/appStore';
import { useSetupStore } from '@/store/setupStore';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { createLogger } from '@/lib/logger';

const log = createLogger('CustomPathDialog');

interface CustomPathDialogProps {
  toolName: string;
  currentCustomPath?: string;  // shown as placeholder if set
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CustomPathDialog({ toolName, currentCustomPath, open, onOpenChange }: CustomPathDialogProps) {
  const goUrl = useAppStore((s) => s.goUrl);
  const fetchStatus = useSetupStore((s) => s.fetchStatus);
  const [path, setPath] = useState(currentCustomPath ?? '');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const handleSave = async () => {
    if (!path.trim()) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${goUrl}/api/setup/tools/${toolName}/path`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: path.trim() }),
      });
      if (res.ok) {
        log.info('custom path set', { tool: toolName, path });
        await fetchStatus(goUrl);
        onOpenChange(false);
      } else {
        const data = await res.json();
        setError(data.message ?? 'Validation failed');
        log.error('custom path rejected', { tool: toolName, code: data.code });
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Request failed');
    } finally {
      setLoading(false);
    }
  };

  const handleClear = async () => {
    setLoading(true);
    setError(null);
    try {
      await fetch(`${goUrl}/api/setup/tools/${toolName}/path`, { method: 'DELETE' });
      log.info('custom path cleared', { tool: toolName });
      await fetchStatus(goUrl);
      onOpenChange(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Request failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Custom path for {toolName}</DialogTitle>
        </DialogHeader>

        <div className="space-y-3 py-2">
          <div className="space-y-1.5">
            <Label htmlFor="custom-path">Binary path</Label>
            <Input
              id="custom-path"
              value={path}
              onChange={(e) => setPath(e.target.value)}
              placeholder="/usr/local/bin/ffmpeg"
              disabled={loading}
            />
          </div>
          {error && (
            <p className="text-xs text-destructive">{error}</p>
          )}
        </div>

        <DialogFooter className="gap-2">
          {currentCustomPath && (
            <Button variant="outline" size="sm" disabled={loading} onClick={handleClear}>
              Clear
            </Button>
          )}
          <Button size="sm" disabled={loading || !path.trim()} onClick={handleSave}>
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
