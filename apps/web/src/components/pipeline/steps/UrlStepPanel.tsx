'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Separator } from '@/components/ui/separator';
import { MetadataPreview, type VideoMetadata } from '@/components/pipeline/MetadataPreview';
import { useWizardStore } from '@/store/wizardStore';
import { useAppStore } from '@/store/appStore';

export function UrlStepPanel() {
  const [url, setUrl] = useState('');
  const [metadata, setMetadata] = useState<VideoMetadata | null>(null);
  const [isVerifying, setIsVerifying] = useState(false);
  const [verifyError, setVerifyError] = useState<string | null>(null);
  const [localFilePath, setLocalFilePath] = useState<string | null>(null);

  const fileInputRef = useRef<HTMLInputElement>(null);

  const goUrl = useAppStore((s) => s.goUrl);
  const setCanAdvance = useWizardStore((s) => s.setCanAdvance);
  const setPendingUrl = useWizardStore((s) => s.setPendingUrl);

  useEffect(() => {
    setCanAdvance(!!metadata || !!localFilePath);
  }, [metadata, localFilePath, setCanAdvance]);

  const handleVerify = useCallback(async () => {
    if (!url.trim()) return;
    setIsVerifying(true);
    setVerifyError(null);
    setMetadata(null);
    try {
      const res = await fetch(`${goUrl}/api/metadata?url=${encodeURIComponent(url)}`);
      if (!res.ok) throw new Error(`Erro ao verificar URL (${res.status})`);
      const data = (await res.json()) as VideoMetadata;
      setMetadata(data);
      setPendingUrl(url);
    } catch (err) {
      setVerifyError(err instanceof Error ? err.message : 'Erro desconhecido');
    } finally {
      setIsVerifying(false);
    }
  }, [url, goUrl, setPendingUrl]);

  const handleSelectFile = useCallback(() => {
    const electronAPI = (window as unknown as Record<string, unknown>).electronAPI as
      | { selectFile?: () => Promise<string | null> }
      | undefined;
    if (electronAPI?.selectFile) {
      electronAPI.selectFile().then((path) => {
        if (path) {
          setLocalFilePath(path);
          setMetadata(null);
          setVerifyError(null);
        }
      });
      return;
    }
    fileInputRef.current?.click();
  }, []);

  const handleFileChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      setLocalFilePath(file.name);
      setMetadata(null);
      setVerifyError(null);
    }
  }, []);

  return (
    <div className="flex flex-col gap-6">
      {/* URL Input */}
      <div className="flex flex-col gap-2">
        <label className="text-sm font-medium text-[#F0F0F8]">URL do vídeo</label>
        <div className="flex gap-2">
          <Input
            type="text"
            placeholder="https://www.youtube.com/watch?v=..."
            value={url}
            onChange={(e) => {
              setUrl(e.target.value);
              setVerifyError(null);
            }}
            className="bg-[#1A1A26]"
          />
          <Button onClick={handleVerify} disabled={isVerifying || !url.trim()}>
            {isVerifying ? 'Verificando...' : 'Verificar'}
          </Button>
        </div>
        {verifyError && <p className="text-sm text-red-400">{verifyError}</p>}
      </div>

      {/* Metadata Preview */}
      {metadata && <MetadataPreview metadata={metadata} />}

      <Separator />

      {/* Vídeo Local */}
      <div className="flex flex-col gap-3">
        <span className="text-sm text-[#A0A0B8]">Vídeo Local</span>
        <div className="flex items-center gap-3">
          <Button variant="outline" onClick={handleSelectFile}>
            Selecionar arquivo
          </Button>
          {localFilePath && (
            <span className="truncate text-sm text-[#F0F0F8]">{localFilePath}</span>
          )}
        </div>
        <input
          ref={fileInputRef}
          type="file"
          accept="video/*"
          className="hidden"
          onChange={handleFileChange}
        />
      </div>
    </div>
  );
}
