'use client';

import { useEffect, useState } from 'react';
import { Loader2, CheckCircle2 } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { FormField } from '@/components/ui/form-field';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { useAppStore } from '@/store/appStore';
import { useSettingsStore, SETTINGS_KEYS } from '@/store/settingsStore';
import { createLogger } from '@/lib/logger';

const log = createLogger('AppSettingsSection');

const QUALITY_OPTIONS = [
  { value: '480p', label: '480p' },
  { value: '720p', label: '720p' },
  { value: '1080p', label: '1080p' },
  { value: '1080p60', label: '1080p60' },
  { value: 'source', label: 'Source' },
];

const LANGUAGE_OPTIONS = [
  { value: 'pt', label: 'PT — Português' },
  { value: 'en', label: 'EN — English' },
  { value: 'es', label: 'ES — Español' },
];

export function AppSettingsSection() {
  const goUrl = useAppStore((s) => s.goUrl);
  const { settings, loading, fetchSettings, saveAllSettings, saveSetting, saving, saveError, saveSuccess } =
    useSettingsStore();

  const [videosDir, setVideosDir] = useState('');
  const [cacheDir, setCacheDir] = useState('');
  const [quality, setQuality] = useState('1080p');
  const [language, setLanguage] = useState('pt');
  const [uploadStrategy, setUploadStrategy] = useState<'sequential' | 'parallel'>('sequential');
  const [uploadParallelCount, setUploadParallelCount] = useState('2');
  const [dirty, setDirty] = useState(false);

  // Fetch settings on mount
  useEffect(() => {
    void fetchSettings(goUrl);
  }, [goUrl, fetchSettings]);

  // Sync form state when settings load
  useEffect(() => {
    if (Object.keys(settings).length > 0) {
      setVideosDir(settings[SETTINGS_KEYS.VIDEOS_DIR] ?? '');
      setCacheDir(settings[SETTINGS_KEYS.CACHE_DIR] ?? '');
      setQuality(settings[SETTINGS_KEYS.DEFAULT_QUALITY] ?? '1080p');
      setLanguage(settings[SETTINGS_KEYS.DEFAULT_LANGUAGE] ?? 'pt');
      setUploadStrategy((settings['upload_strategy'] as 'sequential' | 'parallel') ?? 'sequential');
      setUploadParallelCount(settings['upload_parallel_count'] ?? '2');
      setDirty(false);
    }
  }, [settings]);

  const handleSave = async () => {
    log.info('saving app settings');
    await saveAllSettings(goUrl, {
      videos_dir: videosDir,
      cache_dir: cacheDir,
      default_quality: quality,
      default_language: language,
    });
    await saveSetting(goUrl, 'upload_strategy', uploadStrategy);
    await saveSetting(goUrl, 'upload_parallel_count', uploadParallelCount);
    setDirty(false);
  };

  const markDirty = () => setDirty(true);

  return (
    <section className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-foreground">App Settings</h2>
        {saveSuccess && (
          <Badge variant="success" className="gap-1.5 text-[11px] font-semibold">
            <CheckCircle2 className="h-3.5 w-3.5" />
            Saved
          </Badge>
        )}
      </div>

      <div className="rounded-xl border border-border bg-card p-5 space-y-5">
        {loading ? (
          <div className="flex items-center gap-2 text-sm text-subtle py-4">
            <Loader2 className="h-4 w-4 animate-spin" />
            Loading settings…
          </div>
        ) : (
          <>
            {/* Videos directory */}
            <FormField label="Videos directory">
              <Input
                value={videosDir}
                onChange={(e) => { setVideosDir(e.target.value); markDirty(); }}
                placeholder="/Users/me/Videos"
                className="font-mono text-sm"
              />
            </FormField>

            {/* Cache directory */}
            <FormField label="Cache directory">
              <Input
                value={cacheDir}
                onChange={(e) => { setCacheDir(e.target.value); markDirty(); }}
                placeholder="/tmp/autocut-cache"
                className="font-mono text-sm"
              />
            </FormField>

            {/* Default quality */}
            <div className="flex flex-col gap-1.5">
              <Label className="text-xs font-medium text-subtle">
                Default quality
              </Label>
              <select
                value={quality}
                onChange={(e) => { setQuality(e.target.value); markDirty(); }}
                className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
              >
                {QUALITY_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>

            {/* Default language */}
            <div className="flex flex-col gap-1.5">
              <Label className="text-xs font-medium text-subtle">
                Default language
              </Label>
              <select
                value={language}
                onChange={(e) => { setLanguage(e.target.value); markDirty(); }}
                className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
              >
                {LANGUAGE_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>

            {/* Upload Strategy */}
            <div className="flex flex-col gap-3">
              <h3 className="text-sm font-medium text-caption">Upload Strategy</h3>

              {/* Strategy toggle */}
              <div className="flex items-center justify-between">
                <Label>Parallel Uploads</Label>
                <Switch
                  checked={uploadStrategy === 'parallel'}
                  onCheckedChange={(checked) => {
                    setUploadStrategy(checked ? 'parallel' : 'sequential');
                    markDirty();
                  }}
                />
              </div>

              {/* Parallel count slider — only visible when parallel is enabled */}
              {uploadStrategy === 'parallel' && (
                <div className="flex flex-col gap-2">
                  <div className="flex justify-between">
                    <Label>Concurrent uploads</Label>
                    <span className="text-sm text-caption">{uploadParallelCount}</span>
                  </div>
                  <input
                    type="range"
                    min={1}
                    max={5}
                    value={parseInt(uploadParallelCount, 10)}
                    onChange={(e) => { setUploadParallelCount(e.target.value); markDirty(); }}
                    className="w-full"
                  />
                  <div className="flex justify-between text-xs text-subtle">
                    <span>1 (sequential)</span>
                    <span>5 (max)</span>
                  </div>
                  <p className="text-xs text-subtle">
                    Reduce to 1-2 when YouTube quota usage exceeds 70%
                  </p>
                </div>
              )}
            </div>

            {saveError && (
              <p className="text-xs text-destructive">{saveError}</p>
            )}

            <div className="flex justify-end pt-1">
              <Button
                onClick={() => void handleSave()}
                disabled={saving || !dirty}
                size="sm"
                className="gap-1.5"
              >
                {saving && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
                {saving ? 'Saving…' : 'Save'}
              </Button>
            </div>
          </>
        )}
      </div>
    </section>
  );
}
