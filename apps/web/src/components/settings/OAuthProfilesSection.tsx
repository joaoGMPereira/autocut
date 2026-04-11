'use client';

import { useEffect, useRef, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Separator } from '@/components/ui/separator';
import { useAppStore } from '@/store/appStore';
import { useOAuthStore } from '@/store/oauthStore';
import type { OAuthProfile } from '@/store/oauthStore';
import { createLogger } from '@/lib/logger';

const log = createLogger('OAuthProfilesSection');

function DefaultBadge() {
  return (
    <span className="inline-flex items-center rounded-full px-2.5 py-0.5 text-[11px] font-semibold bg-primary/20 text-primary">
      Default
    </span>
  );
}

export function OAuthProfilesSection() {
  const goUrl = useAppStore((s) => s.goUrl);
  const { profiles, loading, fetchProfiles, uploadProfile, deleteProfile, setDefault } =
    useOAuthStore();

  const [uploadName, setUploadName] = useState('');
  const [uploadFile, setUploadFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<number | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    void fetchProfiles(goUrl);
  }, [goUrl, fetchProfiles]);

  const handleUpload = async () => {
    if (!uploadName.trim() || !uploadFile) {
      setUploadError('Name and file are required.');
      return;
    }
    setUploading(true);
    setUploadError(null);
    try {
      await uploadProfile(goUrl, uploadName.trim(), uploadFile);
      log.info('profile uploaded successfully');
      setUploadName('');
      setUploadFile(null);
      if (fileRef.current) fileRef.current.value = '';
      await fetchProfiles(goUrl);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Upload failed';
      log.error('upload profile failed', { err: message });
      setUploadError(message);
    } finally {
      setUploading(false);
    }
  };

  const handleDelete = async (profile: OAuthProfile) => {
    setActionLoading(profile.id);
    try {
      await deleteProfile(goUrl, profile.id);
      await fetchProfiles(goUrl);
    } catch (err) {
      log.error('delete profile failed', { id: profile.id, err });
    } finally {
      setActionLoading(null);
    }
  };

  const handleSetDefault = async (profile: OAuthProfile) => {
    setActionLoading(profile.id);
    try {
      await setDefault(goUrl, profile.id);
      await fetchProfiles(goUrl);
    } catch (err) {
      log.error('set default failed', { id: profile.id, err });
    } finally {
      setActionLoading(null);
    }
  };

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-foreground">OAuth Profiles</h2>

      <div className="rounded-xl border border-border bg-card">
        {loading && profiles.length === 0 ? (
          <div className="px-5 py-8 text-center text-sm text-muted-foreground">
            Loading profiles…
          </div>
        ) : profiles.length === 0 ? (
          <div className="px-5 py-8 text-center text-sm text-muted-foreground">
            No OAuth profiles configured. Upload a client_secret.json file below.
          </div>
        ) : (
          profiles.map((profile, i) => (
            <div key={profile.id}>
              {i > 0 && <Separator />}
              <div className="flex items-center gap-3 px-5 py-3.5">
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-semibold text-foreground truncate">{profile.name}</p>
                  <p className="text-[11px] font-mono text-muted-foreground truncate">
                    {profile.project_id || '—'}
                  </p>
                </div>
                {profile.is_default && <DefaultBadge />}
                {!profile.is_default && (
                  <Button
                    variant="outline"
                    size="sm"
                    className="h-7 text-xs shrink-0"
                    disabled={actionLoading === profile.id}
                    onClick={() => void handleSetDefault(profile)}
                  >
                    Set Default
                  </Button>
                )}
                <Button
                  variant="destructive"
                  size="sm"
                  className="h-7 text-xs shrink-0"
                  disabled={actionLoading === profile.id}
                  onClick={() => void handleDelete(profile)}
                >
                  Delete
                </Button>
              </div>
            </div>
          ))
        )}
      </div>

      <div className="rounded-xl border border-border bg-card p-4 space-y-3">
        <p className="text-sm font-medium text-foreground">Upload New Profile</p>
        <div className="flex gap-2">
          <Input
            placeholder="Profile name"
            value={uploadName}
            onChange={(e) => setUploadName(e.target.value)}
            className="h-8 text-sm"
          />
          <Input
            ref={fileRef}
            type="file"
            accept=".json"
            className="h-8 text-sm"
            onChange={(e) => setUploadFile(e.target.files?.[0] ?? null)}
          />
          <Button
            size="sm"
            className="h-8 text-xs shrink-0"
            disabled={uploading || !uploadName.trim() || !uploadFile}
            onClick={() => void handleUpload()}
          >
            {uploading ? 'Uploading…' : 'Upload'}
          </Button>
        </div>
        {uploadError && (
          <p className="text-xs text-destructive">{uploadError}</p>
        )}
      </div>
    </section>
  );
}
