'use client';

import { useEffect, useRef, useState } from 'react';
import { AlertCircle, Upload } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
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
    <Badge variant="brand" className="text-[11px] font-semibold">
      Default
    </Badge>
  );
}

export function OAuthProfilesSection() {
  const goUrl = useAppStore((s) => s.goUrl);
  const { profiles, loading, fetchProfiles, uploadProfile, patchProfile, deleteProfile, setDefault } =
    useOAuthStore();

  const [uploadName, setUploadName] = useState('');
  const [uploadFile, setUploadFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<number | null>(null);
  const [patchingId, setPatchingId] = useState<number | null>(null);
  const [patchError, setPatchError] = useState<Record<number, string>>({});
  const fileRef = useRef<HTMLInputElement>(null);
  const patchFileRefs = useRef<Record<number, HTMLInputElement | null>>({});

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

  const handlePatch = async (profile: OAuthProfile, file: File) => {
    setPatchingId(profile.id);
    setPatchError((prev) => ({ ...prev, [profile.id]: '' }));
    try {
      await patchProfile(goUrl, profile.id, file);
      log.info('profile re-uploaded', { id: profile.id });
      await fetchProfiles(goUrl);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Re-upload failed';
      log.error('patch profile failed', { id: profile.id, err: message });
      setPatchError((prev) => ({ ...prev, [profile.id]: message }));
    } finally {
      setPatchingId(null);
      const ref = patchFileRefs.current[profile.id];
      if (ref) ref.value = '';
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
          <div className="px-5 py-8 text-center text-sm text-subtle">
            Loading profiles…
          </div>
        ) : profiles.length === 0 ? (
          <div className="px-5 py-8 text-center text-sm text-subtle">
            No OAuth profiles configured. Upload a client_secret.json file below.
          </div>
        ) : (
          profiles.map((profile, i) => (
            <div key={profile.id}>
              {i > 0 && <Separator />}
              <div className="flex flex-col px-5 py-3.5 gap-2">
                <div className="flex items-center gap-3">
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-semibold text-foreground truncate">{profile.name}</p>
                    <p className="text-[11px] font-mono text-subtle truncate">
                      {profile.project_id || '—'}
                    </p>
                  </div>
                  {!profile.has_credentials && (
                    <Badge variant="warning" className="gap-1 shrink-0">
                      <AlertCircle className="h-3 w-3" />
                      Missing credentials
                    </Badge>
                  )}
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

                {/* Re-upload row — shown when credentials are missing */}
                {!profile.has_credentials && (
                  <div className="flex flex-col gap-1.5 pl-0">
                    <p className="text-[11px] text-warning">
                      This profile was migrated without credentials. Re-upload the OAuth JSON to fix authorization.
                    </p>
                    <div className="flex items-center gap-2">
                      <input
                        ref={(el) => { patchFileRefs.current[profile.id] = el; }}
                        type="file"
                        accept=".json"
                        className="text-[11px] text-subtle file:mr-2 file:text-xs file:font-medium file:border file:border-border file:rounded file:px-2 file:py-0.5 file:bg-card file:text-foreground hover:file:bg-muted cursor-pointer"
                        onChange={(e) => {
                          const f = e.target.files?.[0];
                          if (f) void handlePatch(profile, f);
                        }}
                      />
                      {patchingId === profile.id && (
                        <span className="text-[11px] text-subtle">Updating…</span>
                      )}
                    </div>
                    {patchError[profile.id] && (
                      <p className="text-[11px] text-destructive">{patchError[profile.id]}</p>
                    )}
                  </div>
                )}
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
            <Upload className="h-3.5 w-3.5 mr-1" />
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
