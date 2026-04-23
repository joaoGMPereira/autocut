# Design System Phase 4 — StepMode Consumer Migration Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate 32 hand-rolled control sites in `StepMode.tsx` to Phase 3 primitives (`SettingRow`, `Switch`, `SegmentedControl`, `SliderRow`) across 3 conventional commits on `main`. Zero behavior change. Zero new primitives.

**Architecture:** Single-file refactor. Each commit targets one primitive family and must pass `npx vitest run` before being created. No PR, no worktree — direct to `main` per standing decision. Visual drift accepted per spec.

**Tech Stack:** TypeScript 5.8, React 19, Next.js 16, Tailwind CSS, Radix UI, Vitest.

**Spec:** `docs/superpowers/specs/2026-04-23-design-system-phase4-design.md`

---

## Pre-flight

**Files touched across all tasks:**
- Modify: `apps/web/src/components/pipeline/steps/StepMode.tsx`

**Primitive APIs:**
- `SettingRow` — `{ label: ReactNode, description?: ReactNode, className?: string, children: ReactNode }` — renders label column + right-aligned control slot.
- `Switch` — Radix Switch, props: `{ checked: boolean, onCheckedChange: (v: boolean) => void }`.
- `SegmentedControl<T extends string>` — `{ options: SegmentedOption<T>[], value: T, onChange: (v: T) => void, variant?: 'flat' | 'card', className?: string }` where `SegmentedOption<T> = { value: T, label: string, description?: string }`. Card variant supports 2/3/4 options.
- `SliderRow` — `{ label: string, min: number, max: number, value: number, onChange: (v: number) => void, step?: number, format?: (v: number) => string, className?: string }`.

**Verification gate (every commit):**
```sh
cd apps/web && npx vitest run
```

- [ ] **Step 0.1: Verify baseline state**

Run from repo root:
```sh
git rev-parse --abbrev-ref HEAD
git status --short
```
Expected: on `main`, clean working tree.

- [ ] **Step 0.2: Verify baseline tests pass**

Run:
```sh
cd apps/web && npx vitest run
```
Expected: all tests PASS. Capture the baseline green state before starting migrations. If anything fails here, STOP and report — do not proceed.

- [ ] **Step 0.3: Confirm primitive imports exist**

Run:
```sh
ls apps/web/src/components/ui/setting-row.tsx apps/web/src/components/ui/switch.tsx apps/web/src/components/ui/segmented-control.tsx apps/web/src/components/ui/slider-row.tsx
```
Expected: all four files exist.

---

## Task 1: Switch + SettingRow Migration

**Files:**
- Modify: `apps/web/src/components/pipeline/steps/StepMode.tsx`

**Scope:** 15 `role="switch"` sites → `<Switch>` (Radix). Of those, 10 sit inside `<div class="flex items-center justify-between">` wrappers that become `<SettingRow>`; 5 sit in `SectionPanel actions={…}` props and become bare `<Switch>`.

### Step 1.1: Add imports

- [ ] **Step 1.1: Add SettingRow and Switch imports**

Open `apps/web/src/components/pipeline/steps/StepMode.tsx`. The import block runs lines 1–17. Add two new lines, keeping alphabetical order within the ui/ group:

Before (lines 14–15):
```typescript
import { SectionPanel } from '@/components/ui/section-panel';
import { Textarea } from '@/components/ui/textarea';
```

After:
```typescript
import { SectionPanel } from '@/components/ui/section-panel';
import { SettingRow } from '@/components/ui/setting-row';
import { Switch } from '@/components/ui/switch';
import { Textarea } from '@/components/ui/textarea';
```

### Step 1.2: Migrate the 10 SettingRow + Switch sites

All 10 follow an identical transformation. Each current site has this shape:

**Template — BEFORE:**
```tsx
<div className="flex items-center justify-between">
  <div>
    <p className="text-xs text-prose">{LABEL}</p>
    <p className="text-xs text-caption">{DESCRIPTION}</p>  {/* optional */}
  </div>
  <button
    onClick={() => {HANDLER}}
    className={[
      'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
      {STATE} ? 'bg-brand/80' : 'bg-muted',
    ].join(' ')}
    role="switch"
    aria-checked={STATE}
  >
    <span
      className={[
        'inline-block h-3.5 w-3.5 transform rounded-full bg-background transition-transform',
        {STATE} ? 'translate-x-4' : 'translate-x-1',
      ].join(' ')}
    />
  </button>
</div>
```

**Template — AFTER:**
```tsx
<SettingRow label={LABEL} description={DESCRIPTION /* omit prop if no description */}>
  <Switch checked={STATE} onCheckedChange={HANDLER_NEW_SIG} />
</SettingRow>
```

**Handler signature translation.** The current button uses `onClick={() => setX((v) => !v)}`. `<Switch>` calls `onCheckedChange(nextChecked: boolean)`. Two cases:

- If the current handler body is `() => setX((v) => !v)`, replace with `setX` directly (it accepts `boolean` or updater — the boolean form wins here).
- If the current handler has side effects (e.g., `console.log` before setter), wrap with `(checked) => { console.log(...); setX(checked); }`.

**Per-site table:**

| # | Find anchor (label text + handler) | LABEL | DESCRIPTION | STATE | HANDLER_NEW_SIG |
|---|----------------------------------|-------|-------------|-------|-----------------|
| 1 | "Reuse Existing Clips" / `setSkipRegenerate` (inside `<InfoBanner>`) | `"Reuse Existing Clips"` | `` `From previous run #${priorRunId}` `` | `skipRegenerate` | `(checked) => { console.log('[StepMode] skip_regenerate toggled:', checked); setSkipRegenerate(checked); }` |
| 2 | "Force Re-analyze" / `setForceRegenerate` | `"Force Re-analyze"` | `"Ignore cached transcript and analysis"` | `forceRegenerate` | `setForceRegenerate` |
| 3 | "Skip Transcription" / `setSkipTranscription` | `"Skip Transcription"` | `"Use existing transcript if available"` | `skipTranscription` | `setSkipTranscription` |
| 4 | "Watermark (Logo)" / `setBrandingLogoEnabled` | `"Watermark (Logo)"` | _none_ | `brandingLogoEnabled` | `setBrandingLogoEnabled` |
| 5 | "Watermark pulsante" / `setBrandingLogoPulse` | `"Watermark pulsante"` | _none_ | `brandingLogoPulse` | `setBrandingLogoPulse` |
| 6 | "Intro animado" / `setBrandingIntroEnabled` | `"Intro animado"` | `"3s a partir do logo do canal"` | `brandingIntroEnabled` | `setBrandingIntroEnabled` |
| 7 | "Outro (tela final)" / `setBrandingOutroEnabled` | `"Outro (tela final)"` | _none_ | `brandingOutroEnabled` | `setBrandingOutroEnabled` |
| 8 | "Schedule Sequentially" / `setUploadScheduleEnabled` | `"Schedule Sequentially"` | `"2 uploads/day per type"` | `uploadScheduleEnabled` | `setUploadScheduleEnabled` |
| 9 | "Auto Upload" / `setUploadAutoEnabled` | `"Auto Upload"` | `"Upload immediately vs. manual queue"` | `uploadAutoEnabled` | `setUploadAutoEnabled` |
| 10 | "Dry Run" / `setUploadDryRun` | `"Dry Run"` | `"Generate thumbnails without uploading"` | `uploadDryRun` | `setUploadDryRun` |

**Concrete exemplar (site #6 — with description).**

- [ ] **Step 1.2: Migrate site #6 (Intro animado) as reference implementation**

BEFORE (around line 1606):
```tsx
{/* Intro toggle */}
<div className="flex items-center justify-between">
  <div>
    <p className="text-xs text-prose">Intro animado</p>
    <p className="text-xs text-caption">3s a partir do logo do canal</p>
  </div>
  <button
    onClick={() => setBrandingIntroEnabled((v) => !v)}
    className={[
      'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
      brandingIntroEnabled ? 'bg-brand/80' : 'bg-muted',
    ].join(' ')}
    role="switch"
    aria-checked={brandingIntroEnabled}
  >
    <span
      className={[
        'inline-block h-3.5 w-3.5 transform rounded-full bg-background transition-transform',
        brandingIntroEnabled ? 'translate-x-4' : 'translate-x-1',
      ].join(' ')}
    />
  </button>
</div>
```

AFTER:
```tsx
{/* Intro toggle */}
<SettingRow label="Intro animado" description="3s a partir do logo do canal">
  <Switch checked={brandingIntroEnabled} onCheckedChange={setBrandingIntroEnabled} />
</SettingRow>
```

- [ ] **Step 1.3: Migrate site #1 (Reuse Existing Clips) — includes side-effect console.log**

BEFORE (around line 767 inside `<InfoBanner>`):
```tsx
<div className="flex items-center justify-between">
  <div>
    <p className="text-xs font-medium text-prose">Reuse Existing Clips</p>
    <p className="text-xs text-subtle">From previous run #{priorRunId}</p>
  </div>
  <button
    onClick={() => {
      console.log('[StepMode] skip_regenerate toggled:', !skipRegenerate);
      setSkipRegenerate((v) => !v);
    }}
    className={[
      'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
      skipRegenerate ? 'bg-brand/80' : 'bg-muted',
    ].join(' ')}
    role="switch"
    aria-checked={skipRegenerate}
  >
    <span
      className={[
        'inline-block h-3.5 w-3.5 transform rounded-full bg-background transition-transform',
        skipRegenerate ? 'translate-x-4' : 'translate-x-1',
      ].join(' ')}
    />
  </button>
</div>
```

AFTER:
```tsx
<SettingRow label="Reuse Existing Clips" description={`From previous run #${priorRunId}`}>
  <Switch
    checked={skipRegenerate}
    onCheckedChange={(checked) => {
      console.log('[StepMode] skip_regenerate toggled:', checked);
      setSkipRegenerate(checked);
    }}
  />
</SettingRow>
```

- [ ] **Step 1.4a: Migrate site #2 (Force Re-analyze)**

BEFORE (around line 958):
```tsx
{/* Force regenerate toggle */}
<div className="flex items-center justify-between">
  <div>
    <p className="text-xs text-prose">Force Re-analyze</p>
    <p className="text-xs text-caption">Ignore cached transcript and analysis</p>
  </div>
  <button
    onClick={() => setForceRegenerate((v) => !v)}
    className={[
      'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
      forceRegenerate ? 'bg-brand/80' : 'bg-muted',
    ].join(' ')}
    role="switch"
    aria-checked={forceRegenerate}
  >
    <span
      className={[
        'inline-block h-3.5 w-3.5 transform rounded-full bg-background transition-transform',
        forceRegenerate ? 'translate-x-4' : 'translate-x-1',
      ].join(' ')}
    />
  </button>
</div>
```

AFTER:
```tsx
{/* Force regenerate toggle */}
<SettingRow label="Force Re-analyze" description="Ignore cached transcript and analysis">
  <Switch checked={forceRegenerate} onCheckedChange={setForceRegenerate} />
</SettingRow>
```

- [ ] **Step 1.4b: Migrate site #3 (Skip Transcription)**

BEFORE (around line 982):
```tsx
{/* Skip transcription toggle */}
<div className="flex items-center justify-between">
  <div>
    <p className="text-xs text-prose">Skip Transcription</p>
    <p className="text-xs text-caption">Use existing transcript if available</p>
  </div>
  <button
    onClick={() => setSkipTranscription((v) => !v)}
    className={[
      'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
      skipTranscription ? 'bg-brand/80' : 'bg-muted',
    ].join(' ')}
    role="switch"
    aria-checked={skipTranscription}
  >
    <span
      className={[
        'inline-block h-3.5 w-3.5 transform rounded-full bg-background transition-transform',
        skipTranscription ? 'translate-x-4' : 'translate-x-1',
      ].join(' ')}
    />
  </button>
</div>
```

AFTER:
```tsx
{/* Skip transcription toggle */}
<SettingRow label="Skip Transcription" description="Use existing transcript if available">
  <Switch checked={skipTranscription} onCheckedChange={setSkipTranscription} />
</SettingRow>
```

- [ ] **Step 1.4c: Migrate site #4 (Watermark (Logo) — no description)**

BEFORE (around line 1469):
```tsx
<div className="flex items-center justify-between">
  <p className="text-xs text-prose">Watermark (Logo)</p>
  <button
    onClick={() => setBrandingLogoEnabled((v) => !v)}
    className={[
      'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
      brandingLogoEnabled ? 'bg-brand/80' : 'bg-muted',
    ].join(' ')}
    role="switch"
    aria-checked={brandingLogoEnabled}
  >
    <span
      className={[
        'inline-block h-3.5 w-3.5 transform rounded-full bg-background transition-transform',
        brandingLogoEnabled ? 'translate-x-4' : 'translate-x-1',
      ].join(' ')}
    />
  </button>
</div>
```

AFTER:
```tsx
<SettingRow label="Watermark (Logo)">
  <Switch checked={brandingLogoEnabled} onCheckedChange={setBrandingLogoEnabled} />
</SettingRow>
```

- [ ] **Step 1.4d: Migrate site #5 (Watermark pulsante — no description)**

BEFORE (around line 1582):
```tsx
{/* Pulse toggle */}
<div className="flex items-center justify-between">
  <p className="text-xs text-prose">Watermark pulsante</p>
  <button
    onClick={() => setBrandingLogoPulse((v) => !v)}
    className={[
      'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
      brandingLogoPulse ? 'bg-brand/80' : 'bg-muted',
    ].join(' ')}
    role="switch"
    aria-checked={brandingLogoPulse}
  >
    <span
      className={[
        'inline-block h-3.5 w-3.5 transform rounded-full bg-background transition-transform',
        brandingLogoPulse ? 'translate-x-4' : 'translate-x-1',
      ].join(' ')}
    />
  </button>
</div>
```

AFTER:
```tsx
{/* Pulse toggle */}
<SettingRow label="Watermark pulsante">
  <Switch checked={brandingLogoPulse} onCheckedChange={setBrandingLogoPulse} />
</SettingRow>
```

- [ ] **Step 1.4e: Migrate site #7 (Outro (tela final) — no description)**

BEFORE (around line 1630):
```tsx
{/* Outro toggle */}
<div className="flex items-center justify-between">
  <p className="text-xs text-prose">Outro (tela final)</p>
  <button
    onClick={() => setBrandingOutroEnabled((v) => !v)}
    className={[
      'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
      brandingOutroEnabled ? 'bg-brand/80' : 'bg-muted',
    ].join(' ')}
    role="switch"
    aria-checked={brandingOutroEnabled}
  >
    <span
      className={[
        'inline-block h-3.5 w-3.5 transform rounded-full bg-background transition-transform',
        brandingOutroEnabled ? 'translate-x-4' : 'translate-x-1',
      ].join(' ')}
    />
  </button>
</div>
```

AFTER:
```tsx
{/* Outro toggle */}
<SettingRow label="Outro (tela final)">
  <Switch checked={brandingOutroEnabled} onCheckedChange={setBrandingOutroEnabled} />
</SettingRow>
```

- [ ] **Step 1.4f: Migrate site #8 (Schedule Sequentially)**

BEFORE (around line 1993):
```tsx
{/* Schedule toggle */}
<div className="flex items-center justify-between">
  <div>
    <p className="text-xs text-prose">Schedule Sequentially</p>
    <p className="text-xs text-caption">2 uploads/day per type</p>
  </div>
  <button
    onClick={() => setUploadScheduleEnabled((v) => !v)}
    className={[
      'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
      uploadScheduleEnabled ? 'bg-brand/80' : 'bg-muted',
    ].join(' ')}
    role="switch"
    aria-checked={uploadScheduleEnabled}
  >
    <span
      className={[
        'inline-block h-3.5 w-3.5 transform rounded-full bg-background transition-transform',
        uploadScheduleEnabled ? 'translate-x-4' : 'translate-x-1',
      ].join(' ')}
    />
  </button>
</div>
```

AFTER:
```tsx
{/* Schedule toggle */}
<SettingRow label="Schedule Sequentially" description="2 uploads/day per type">
  <Switch checked={uploadScheduleEnabled} onCheckedChange={setUploadScheduleEnabled} />
</SettingRow>
```

- [ ] **Step 1.4g: Migrate site #9 (Auto Upload)**

BEFORE (around line 2017):
```tsx
{/* Auto upload toggle */}
<div className="flex items-center justify-between">
  <div>
    <p className="text-xs text-prose">Auto Upload</p>
    <p className="text-xs text-caption">Upload immediately vs. manual queue</p>
  </div>
  <button
    onClick={() => setUploadAutoEnabled((v) => !v)}
    className={[
      'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
      uploadAutoEnabled ? 'bg-brand/80' : 'bg-muted',
    ].join(' ')}
    role="switch"
    aria-checked={uploadAutoEnabled}
  >
    <span
      className={[
        'inline-block h-3.5 w-3.5 transform rounded-full bg-background transition-transform',
        uploadAutoEnabled ? 'translate-x-4' : 'translate-x-1',
      ].join(' ')}
    />
  </button>
</div>
```

AFTER:
```tsx
{/* Auto upload toggle */}
<SettingRow label="Auto Upload" description="Upload immediately vs. manual queue">
  <Switch checked={uploadAutoEnabled} onCheckedChange={setUploadAutoEnabled} />
</SettingRow>
```

- [ ] **Step 1.4h: Migrate site #10 (Dry Run)**

BEFORE (around line 2041):
```tsx
{/* Dry run toggle */}
<div className="flex items-center justify-between">
  <div>
    <p className="text-xs text-prose">Dry Run</p>
    <p className="text-xs text-caption">Generate thumbnails without uploading</p>
  </div>
  <button
    onClick={() => setUploadDryRun((v) => !v)}
    className={[
      'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
      uploadDryRun ? 'bg-brand/80' : 'bg-muted',
    ].join(' ')}
    role="switch"
    aria-checked={uploadDryRun}
  >
    <span
      className={[
        'inline-block h-3.5 w-3.5 transform rounded-full bg-background transition-transform',
        uploadDryRun ? 'translate-x-4' : 'translate-x-1',
      ].join(' ')}
    />
  </button>
</div>
```

AFTER:
```tsx
{/* Dry run toggle */}
<SettingRow label="Dry Run" description="Generate thumbnails without uploading">
  <Switch checked={uploadDryRun} onCheckedChange={setUploadDryRun} />
</SettingRow>
```

### Step 1.5: Migrate the 5 bare Switch sites (SectionPanel actions prop)

All 5 follow this shape:

**Template — BEFORE:**
```tsx
<SectionPanel
  title={...}
  actions={
    <button
      onClick={() => {HANDLER}}
      className={[
        'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
        {STATE} ? 'bg-brand/80' : 'bg-muted',
      ].join(' ')}
      role="switch"
      aria-checked={STATE}
    >
      <span
        className={[
          'inline-block h-3.5 w-3.5 transform rounded-full bg-background transition-transform',
          {STATE} ? 'translate-x-4' : 'translate-x-1',
        ].join(' ')}
      />
    </button>
  }
>
```

**Template — AFTER:**
```tsx
<SectionPanel
  title={...}
  actions={<Switch checked={STATE} onCheckedChange={HANDLER_NEW_SIG} />}
>
```

**Per-site table:**

| # | SectionPanel title | STATE | HANDLER_NEW_SIG |
|---|--------------------|-------|-----------------|
| 11 | "Anti-Duplicate Protection" | `antiDupEnabled` | `(checked) => { console.log('[StepMode] anti-dup toggled:', checked); setAntiDupEnabled(checked); }` |
| 12 | "Overlays de Texto" | `textOverlaysEnabled` | `setTextOverlaysEnabled` |
| 13 | "Overlay de Vídeo" | `overlayEnabled` | `setOverlayEnabled` |
| 14 | "Música de Fundo" | `musicEnabled` | `setMusicEnabled` |
| 15 | "Legendas" | `captionsEnabled` | `setCaptionsEnabled` |

- [ ] **Step 1.5: Migrate site #11 (Anti-Duplicate Protection) — preserves console.log**

BEFORE (around line 1082):
```tsx
<SectionPanel
  title="Anti-Duplicate Protection"
  description="Prevent YouTube duplicate detection"
  actions={
    <button
      onClick={() => {
        console.log('[StepMode] anti-dup toggled:', !antiDupEnabled);
        setAntiDupEnabled((v) => !v);
      }}
      className={[
        'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
        antiDupEnabled ? 'bg-brand/80' : 'bg-muted',
      ].join(' ')}
      role="switch"
      aria-checked={antiDupEnabled}
    >
      <span
        className={[
          'inline-block h-3.5 w-3.5 transform rounded-full bg-background transition-transform',
          antiDupEnabled ? 'translate-x-4' : 'translate-x-1',
        ].join(' ')}
      />
    </button>
  }
>
```

AFTER:
```tsx
<SectionPanel
  title="Anti-Duplicate Protection"
  description="Prevent YouTube duplicate detection"
  actions={
    <Switch
      checked={antiDupEnabled}
      onCheckedChange={(checked) => {
        console.log('[StepMode] anti-dup toggled:', checked);
        setAntiDupEnabled(checked);
      }}
    />
  }
>
```

- [ ] **Step 1.6a: Migrate site #12 (Overlays de Texto)**

BEFORE (around line 1213):
```tsx
<SectionPanel
  title="Overlays de Texto"
  actions={
    <button
      onClick={() => setTextOverlaysEnabled((v) => !v)}
      className={[
        'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
        textOverlaysEnabled ? 'bg-brand/80' : 'bg-muted',
      ].join(' ')}
      role="switch"
      aria-checked={textOverlaysEnabled}
    >
      <span
        className={[
          'inline-block h-3.5 w-3.5 transform rounded-full bg-background transition-transform',
          textOverlaysEnabled ? 'translate-x-4' : 'translate-x-1',
        ].join(' ')}
      />
    </button>
  }
>
```

AFTER:
```tsx
<SectionPanel
  title="Overlays de Texto"
  actions={<Switch checked={textOverlaysEnabled} onCheckedChange={setTextOverlaysEnabled} />}
>
```

- [ ] **Step 1.6b: Migrate site #13 (Overlay de Vídeo)**

BEFORE (around line 1338):
```tsx
<SectionPanel
  title="Overlay de Vídeo"
  actions={
    <button
      onClick={() => setOverlayEnabled((v) => !v)}
      className={[
        'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
        overlayEnabled ? 'bg-brand/80' : 'bg-muted',
      ].join(' ')}
      role="switch"
      aria-checked={overlayEnabled}
    >
      <span
        className={[
          'inline-block h-3.5 w-3.5 transform rounded-full bg-background transition-transform',
          overlayEnabled ? 'translate-x-4' : 'translate-x-1',
        ].join(' ')}
      />
    </button>
  }
>
```

AFTER:
```tsx
<SectionPanel
  title="Overlay de Vídeo"
  actions={<Switch checked={overlayEnabled} onCheckedChange={setOverlayEnabled} />}
>
```

- [ ] **Step 1.6c: Migrate site #14 (Música de Fundo)**

BEFORE (around line 1652):
```tsx
<SectionPanel
  title="Música de Fundo"
  actions={
    <button
      onClick={() => setMusicEnabled((v) => !v)}
      className={[
        'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
        musicEnabled ? 'bg-brand/80' : 'bg-muted',
      ].join(' ')}
      role="switch"
      aria-checked={musicEnabled}
    >
      <span
        className={[
          'inline-block h-3.5 w-3.5 transform rounded-full bg-background transition-transform',
          musicEnabled ? 'translate-x-4' : 'translate-x-1',
        ].join(' ')}
      />
    </button>
  }
>
```

AFTER:
```tsx
<SectionPanel
  title="Música de Fundo"
  actions={<Switch checked={musicEnabled} onCheckedChange={setMusicEnabled} />}
>
```

- [ ] **Step 1.6d: Migrate site #15 (Legendas)**

BEFORE (around line 1754):
```tsx
<SectionPanel
  title="Legendas"
  actions={
    <button
      onClick={() => setCaptionsEnabled((v) => !v)}
      className={[
        'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
        captionsEnabled ? 'bg-brand/80' : 'bg-muted',
      ].join(' ')}
      role="switch"
      aria-checked={captionsEnabled}
    >
      <span
        className={[
          'inline-block h-3.5 w-3.5 transform rounded-full bg-background transition-transform',
          captionsEnabled ? 'translate-x-4' : 'translate-x-1',
        ].join(' ')}
      />
    </button>
  }
>
```

AFTER:
```tsx
<SectionPanel
  title="Legendas"
  actions={<Switch checked={captionsEnabled} onCheckedChange={setCaptionsEnabled} />}
>
```

### Step 1.7: Verify

- [ ] **Step 1.7: Confirm all `role="switch"` sites are gone**

Run from repo root:
```sh
grep -n 'role="switch"' apps/web/src/components/pipeline/steps/StepMode.tsx
```
Expected: no output (zero matches).

- [ ] **Step 1.8: Confirm leftover toggle wrappers are only the 3 preserved non-toggle headers**

Run:
```sh
grep -nc 'flex items-center justify-between' apps/web/src/components/pipeline/steps/StepMode.tsx
```
Expected: `3` (lines for Overlay #N header, Preview "Aguardando download", Preview "Ready").

- [ ] **Step 1.9: Run vitest**

Run:
```sh
cd apps/web && npx vitest run
```
Expected: all tests PASS. If any test fails, read the error — likely a query by old class name or ARIA. Adjust the test OR the migration (preserve intended semantics) and re-run. Do not proceed to commit until green.

- [ ] **Step 1.10: Commit**

From repo root:
```sh
git add apps/web/src/components/pipeline/steps/StepMode.tsx
git commit -m "$(cat <<'EOF'
refactor(ui): adopt SettingRow + Switch in StepMode

Replace 15 hand-rolled `role="switch"` buttons in StepMode.tsx with the
Phase 3 <Switch> primitive. 10 sit inside new <SettingRow> wrappers; 5
go bare inside SectionPanel `actions` props.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
git status
```
Expected: commit created, working tree clean.

---

## Task 2: SegmentedControl Migration

**Files:**
- Modify: `apps/web/src/components/pipeline/steps/StepMode.tsx`

**Scope:** 5 `flex gap-2` button groups → `<SegmentedControl variant="flat">`. 1 `grid grid-cols-2 gap-2` card grid → `<SegmentedControl variant="card">`.

### Step 2.1: Add import

- [ ] **Step 2.1: Add SegmentedControl import**

Add to the ui/ import group, alphabetical:

Before:
```typescript
import { SectionPanel } from '@/components/ui/section-panel';
import { SettingRow } from '@/components/ui/setting-row';
```

After:
```typescript
import { SectionPanel } from '@/components/ui/section-panel';
import { SegmentedControl } from '@/components/ui/segmented-control';
import { SettingRow } from '@/components/ui/setting-row';
```

### Step 2.2: Migrate the 5 flat sites

All 5 follow this shape (with minor variations in the label text rendered inside each button):

**Template — BEFORE:**
```tsx
<div className="flex gap-2">
  {OPTIONS.map((opt) => (
    <button
      key={...}
      onClick={() => {SETTER}(opt)}
      className={[
        'flex-1 rounded px-2 py-1.5 text-xs transition-colors',
        {STATE} === opt
          ? 'bg-brand text-brand-foreground font-medium'
          : 'bg-surface text-subtle hover:bg-surface/80',
      ].join(' ')}
    >
      {LABEL}
    </button>
  ))}
</div>
```

**Template — AFTER:**
```tsx
<SegmentedControl
  value={STATE}
  onChange={SETTER}
  options={[
    { value: 'a', label: 'A' },
    ...
  ]}
/>
```

- [ ] **Step 2.2: Migrate Highlight Threshold (3 options with percent annotation)**

BEFORE (around line 920):
```tsx
<div className="space-y-1.5">
  <Label className="text-xs font-medium text-subtle">Highlight Threshold</Label>
  <div className="flex gap-2">
    {([50, 70, 90] as const).map((pct) => (
      <button
        key={pct}
        onClick={() => setSensitivityPct(pct)}
        className={[
          'flex-1 rounded px-2 py-1.5 text-xs transition-colors',
          sensitivityPct === pct
            ? 'bg-brand text-brand-foreground font-medium'
            : 'bg-surface text-subtle hover:bg-surface/80',
        ].join(' ')}
      >
        {pct === 50 ? 'Liberal' : pct === 70 ? 'Balanced' : 'Selective'}
        <span className="ml-1 opacity-60">({pct})</span>
      </button>
    ))}
  </div>
</div>
```

AFTER (label flattens the "(pct)" annotation into the string; SegmentedControl renders the full label as plain text so the opacity-60 styling is dropped — accepted drift):
```tsx
<div className="space-y-1.5">
  <Label className="text-xs font-medium text-subtle">Highlight Threshold</Label>
  <SegmentedControl<'50' | '70' | '90'>
    value={String(sensitivityPct) as '50' | '70' | '90'}
    onChange={(v) => setSensitivityPct(Number(v) as 50 | 70 | 90)}
    options={[
      { value: '50', label: 'Liberal (50)' },
      { value: '70', label: 'Balanced (70)' },
      { value: '90', label: 'Selective (90)' },
    ]}
  />
</div>
```

Note: `SegmentedControl<T extends string>` requires string values. Since `sensitivityPct` is `number`, wrap with `String()` on read and `Number()` on write. The type assertions keep TS happy without changing the state type.

- [ ] **Step 2.3: Migrate Overlay appearances (5 options `1×`…`5×`)**

BEFORE (around line 1407):
```tsx
<div className="space-y-1.5">
  <Label className="text-xs font-medium text-subtle">Nº de Aparições</Label>
  <div className="flex gap-2">
    {([1, 2, 3, 4, 5] as const).map((n) => (
      <button
        key={n}
        onClick={() => setOverlayAppearances(n)}
        className={[
          'flex-1 rounded px-2 py-1.5 text-xs transition-colors',
          overlayAppearances === n
            ? 'bg-brand text-brand-foreground font-medium'
            : 'bg-surface text-subtle hover:bg-surface/80',
        ].join(' ')}
      >
        {n}×
      </button>
    ))}
  </div>
</div>
```

AFTER:
```tsx
<div className="space-y-1.5">
  <Label className="text-xs font-medium text-subtle">Nº de Aparições</Label>
  <SegmentedControl<'1' | '2' | '3' | '4' | '5'>
    value={String(overlayAppearances) as '1' | '2' | '3' | '4' | '5'}
    onChange={(v) => setOverlayAppearances(Number(v) as 1 | 2 | 3 | 4 | 5)}
    options={[
      { value: '1', label: '1×' },
      { value: '2', label: '2×' },
      { value: '3', label: '3×' },
      { value: '4', label: '4×' },
      { value: '5', label: '5×' },
    ]}
  />
</div>
```

- [ ] **Step 2.4: Migrate Música mode (3 string-typed options)**

BEFORE (around line 1676):
```tsx
<div className="space-y-1.5">
  <Label className="text-xs font-medium text-subtle">Seleção</Label>
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
            ? 'bg-brand text-brand-foreground font-medium'
            : 'bg-surface text-subtle hover:bg-surface/80',
        ].join(' ')}
      >
        {label}
      </button>
    ))}
  </div>
</div>
```

AFTER (music mode is already a string union, so no `String()` juggling):
```tsx
<div className="space-y-1.5">
  <Label className="text-xs font-medium text-subtle">Seleção</Label>
  <SegmentedControl<BackgroundMusicConfig['mode']>
    value={musicMode}
    onChange={setMusicMode}
    options={[
      { value: 'random', label: 'Aleatório' },
      { value: 'library', label: 'Biblioteca' },
      { value: 'custom', label: 'Arquivo' },
    ]}
  />
</div>
```

- [ ] **Step 2.5: Migrate Captions preset (3 string-typed options)**

BEFORE (around line 1778):
```tsx
<div className="space-y-1.5">
  <Label className="text-xs font-medium text-subtle">Preset</Label>
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
            ? 'bg-brand text-brand-foreground font-medium'
            : 'bg-surface text-subtle hover:bg-surface/80',
        ].join(' ')}
      >
        {label}
      </button>
    ))}
  </div>
</div>
```

AFTER:
```tsx
<div className="space-y-1.5">
  <Label className="text-xs font-medium text-subtle">Preset</Label>
  <SegmentedControl<CaptionsConfig['preset']>
    value={captionsPreset}
    onChange={setCaptionsPreset}
    options={[
      { value: 'simple', label: 'Simples' },
      { value: 'bold', label: 'Negrito' },
      { value: 'word_by_word', label: 'Palavra a Palavra' },
    ]}
  />
</div>
```

- [ ] **Step 2.6: Migrate Upload privacy (3 string-typed options, pill → rect drift accepted)**

BEFORE (around line 1972):
```tsx
<div className="space-y-1.5">
  <Label className="text-xs font-medium text-subtle">Privacy</Label>
  <div className="flex gap-2">
    {(['private', 'unlisted', 'public'] as const).map((p) => (
      <button
        key={p}
        onClick={() => setUploadPrivacy(p)}
        className={[
          'flex-1 rounded-full px-3 py-1.5 text-xs font-medium capitalize transition-colors',
          uploadPrivacy === p
            ? 'bg-brand text-brand-foreground'
            : 'bg-surface text-subtle hover:bg-surface/80',
        ].join(' ')}
      >
        {p}
      </button>
    ))}
  </div>
</div>
```

AFTER (pre-capitalize labels because SegmentedControl drops the `capitalize` class):
```tsx
<div className="space-y-1.5">
  <Label className="text-xs font-medium text-subtle">Privacy</Label>
  <SegmentedControl<'private' | 'unlisted' | 'public'>
    value={uploadPrivacy}
    onChange={setUploadPrivacy}
    options={[
      { value: 'private', label: 'Private' },
      { value: 'unlisted', label: 'Unlisted' },
      { value: 'public', label: 'Public' },
    ]}
  />
</div>
```

### Step 2.7: Migrate the 1 card site (Anti-dup mode)

- [ ] **Step 2.7: Migrate Anti-dup mode (grid-cols-2 → SegmentedControl card)**

BEFORE (around line 1109):
```tsx
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
          ? 'border-brand bg-surface ring-1 ring-brand'
          : 'border-border bg-surface hover:border-border',
      ].join(' ')}
    >
      <span className="text-xs font-medium text-prose capitalize">{m}</span>
      <span className="text-xs text-subtle">
        {m === 'subtle' ? '1–2% changes' : '5–10% changes'}
      </span>
    </button>
  ))}
</div>
```

AFTER (pre-capitalize labels; wrap the onChange to preserve the console.log):
```tsx
<SegmentedControl<'subtle' | 'aggressive'>
  variant="card"
  value={antiDupMode}
  onChange={(m) => {
    console.log('[StepMode] anti-dup mode selected:', m);
    setAntiDupMode(m);
  }}
  options={[
    { value: 'subtle', label: 'Subtle', description: '1–2% changes' },
    { value: 'aggressive', label: 'Aggressive', description: '5–10% changes' },
  ]}
/>
```

### Step 2.8: Verify

- [ ] **Step 2.8: Confirm the migrated flat button groups are gone**

Run:
```sh
grep -n 'flex gap-2' apps/web/src/components/pipeline/steps/StepMode.tsx | grep -v 'flex-wrap\|overflow-x-auto'
```
Expected: exactly 2 matches — L1409 region _after_ migration should be gone; remaining matches are `flex gap-2` in the preset-save row (Input + Button) and any incidental layout. Inspect each match; none should be a button group driving a single-value state.

Acceptable leftover matches:
- preset-save row (`<Input>` + `<Button>` save/update)
- any non-button-group utility layout

- [ ] **Step 2.9: Confirm the anti-dup card grid is gone**

Run:
```sh
grep -n 'grid grid-cols-2 gap-2' apps/web/src/components/pipeline/steps/StepMode.tsx
```
Expected: no output.

- [ ] **Step 2.10: Run vitest**

Run:
```sh
cd apps/web && npx vitest run
```
Expected: all tests PASS.

- [ ] **Step 2.11: Commit**

```sh
git add apps/web/src/components/pipeline/steps/StepMode.tsx
git commit -m "$(cat <<'EOF'
refactor(ui): adopt SegmentedControl in StepMode

Replace 5 hand-rolled `flex gap-2` flat button groups and 1
`grid grid-cols-2 gap-2` card grid with the Phase 3 <SegmentedControl>
primitive (variants flat + card).

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
git status
```
Expected: commit created, working tree clean.

---

## Task 3: SliderRow Migration

**Files:**
- Modify: `apps/web/src/components/pipeline/steps/StepMode.tsx`

**Scope:** 11 `<input type="range">` rows → `<SliderRow>`.

### Step 3.1: Add import

- [ ] **Step 3.1: Add SliderRow import**

Before:
```typescript
import { SettingRow } from '@/components/ui/setting-row';
import { Switch } from '@/components/ui/switch';
```

After:
```typescript
import { SettingRow } from '@/components/ui/setting-row';
import { SliderRow } from '@/components/ui/slider-row';
import { Switch } from '@/components/ui/switch';
```

### Step 3.2: Migrate the 11 sliders

All 11 take one of two current shapes:

**Shape A — with Label + fixed-width value span** (used in "Escala", "Offset Final", "Opacidade", etc.):
```tsx
<div className="flex items-center gap-3">
  <Label className="text-xs font-medium text-subtle w-20 flex-shrink-0">{LABEL}</Label>
  <input
    type="range"
    min={MIN}
    max={MAX}
    value={VALUE}
    onChange={(e) => {SETTER}(Number(e.target.value))}
    className="flex-1 accent-brand"
  />
  <span className="text-xs text-subtle w-NN text-right">{VALUE_DISPLAY}</span>
</div>
```

**Shape B — with `span` label and `ml-6` indentation** (used inside checkbox-gated sub-controls like Noise strength, Blur edge %):
```tsx
<div className="ml-6 flex items-center gap-2">
  <span className="text-xs text-subtle w-NN">{LABEL}</span>
  <input
    type="range"
    min={MIN}
    max={MAX}
    value={VALUE}
    onChange={(e) => {SETTER}(Number(e.target.value))}
    className="flex-1 accent-brand"
  />
  <span className="text-xs text-subtle w-NN">{VALUE_DISPLAY}</span>
</div>
```

**Unified AFTER template:**
```tsx
<SliderRow
  label={LABEL}
  min={MIN}
  max={MAX}
  value={VALUE}
  onChange={SETTER}
  format={FORMAT}
/>
```

For Shape B sliders, wrap in the preserved `ml-6` div:
```tsx
<div className="ml-6">
  <SliderRow label={LABEL} min={MIN} max={MAX} value={VALUE} onChange={SETTER} format={FORMAT} />
</div>
```

**Per-site table:**

| # | Anchor | LABEL | MIN | MAX | VALUE expr | SETTER | FORMAT | Shape |
|---|--------|-------|-----|-----|------------|--------|--------|-------|
| 1 | noise strength (inside `{antiDupEffects.noise && …}`) | `"Strength"` | `1` | `10` | `antiDupEffects.noise_strength ?? 3` | `(v) => setEffect('noise_strength', v)` | _omit — default `String(v)`_ | B |
| 2 | blur edge (inside `{antiDupEffects.blur && …}`) | `"Edge %"` | `0` | `100` | `antiDupEffects.blur_edge_pct ?? 10` | `(v) => setEffect('blur_edge_pct', v)` | `` (v) => `${v}%` `` | B |
| 3 | overlay Escala | `"Escala"` | `10` | `200` | `overlayScalePct` | `setOverlayScalePct` | `` (v) => `${v}%` `` | A |
| 4 | overlay Offset Final | `"Offset Final"` | `0` | `10` | `overlayEndOffsetSec` | `setOverlayEndOffsetSec` | `` (v) => `${v}s` `` | A |
| 5 | watermark Opacidade | `"Opacidade"` | `0` | `100` | `brandingLogoOpacity` | `setBrandingLogoOpacity` | `` (v) => `${v}%` `` | A |
| 6 | watermark Tamanho | `"Tamanho"` | `5` | `30` | `brandingLogoScale` | `setBrandingLogoScale` | `` (v) => `${v}%` `` | A |
| 7 | music Volume | `"Volume"` | `0` | `30` | `musicVolumePct` | `setMusicVolumePct` | `` (v) => `${v}%` `` | A |
| 8 | captions Tamanho | `"Tamanho"` | `12` | `96` | `captionsFontSize` | `setCaptionsFontSize` | `` (v) => `${v}px` `` | A |
| 9 | captions outline Espessura | `"Espessura"` | `1` | `10` | `captionsOutlineWidth` | `setCaptionsOutlineWidth` | _omit_ | B (ml-6 parent) |
| 10 | captions shadow Distância | `"Distância"` | `0` | `20` | `captionsShadowDistance` | `setCaptionsShadowDistance` | _omit_ | B (ml-6 parent) |
| 11 | captions Offset Vertical | `"Offset Vertical"` | `-100` | `100` | `captionsVerticalOffset` | `setCaptionsVerticalOffset` | _omit_ | A |

- [ ] **Step 3.2: Migrate site #3 (overlay Escala) as the Shape A reference**

BEFORE (around line 1386):
```tsx
{/* Scale slider */}
<div className="flex items-center gap-3">
  <Label className="text-xs font-medium text-subtle w-20 flex-shrink-0">Escala</Label>
  <input
    type="range"
    min={10}
    max={200}
    value={overlayScalePct}
    onChange={(e) => setOverlayScalePct(Number(e.target.value))}
    className="flex-1 accent-brand"
  />
  <span className="text-xs text-subtle w-10 text-right">{overlayScalePct}%</span>
</div>
```

AFTER:
```tsx
{/* Scale slider */}
<SliderRow
  label="Escala"
  min={10}
  max={200}
  value={overlayScalePct}
  onChange={setOverlayScalePct}
  format={(v) => `${v}%`}
/>
```

- [ ] **Step 3.3: Migrate site #1 (noise strength) as the Shape B reference**

BEFORE (around line 1166 inside `{antiDupEffects.noise && …}`):
```tsx
<div className="ml-6 flex items-center gap-2">
  <span className="text-xs text-subtle w-16">Strength</span>
  <input
    type="range"
    min={1}
    max={10}
    value={antiDupEffects.noise_strength ?? 3}
    onChange={(e) => setEffect('noise_strength', Number(e.target.value))}
    className="flex-1 accent-brand"
  />
  <span className="text-xs text-subtle w-4">{antiDupEffects.noise_strength ?? 3}</span>
</div>
```

AFTER (preserve the `ml-6` offset on a wrapper so indentation is kept; SliderRow uses `gap-3` internally):
```tsx
<div className="ml-6">
  <SliderRow
    label="Strength"
    min={1}
    max={10}
    value={antiDupEffects.noise_strength ?? 3}
    onChange={(v) => setEffect('noise_strength', v)}
  />
</div>
```

- [ ] **Step 3.4: Migrate site #2 (blur edge %)**

BEFORE (around line 1193):
```tsx
<div className="ml-6 flex items-center gap-2">
  <span className="text-xs text-subtle w-16">Edge %</span>
  <input
    type="range"
    min={0}
    max={100}
    value={antiDupEffects.blur_edge_pct ?? 10}
    onChange={(e) => setEffect('blur_edge_pct', Number(e.target.value))}
    className="flex-1 accent-brand"
  />
  <span className="text-xs text-subtle w-8">{antiDupEffects.blur_edge_pct ?? 10}%</span>
</div>
```

AFTER:
```tsx
<div className="ml-6">
  <SliderRow
    label="Edge %"
    min={0}
    max={100}
    value={antiDupEffects.blur_edge_pct ?? 10}
    onChange={(v) => setEffect('blur_edge_pct', v)}
    format={(v) => `${v}%`}
  />
</div>
```

- [ ] **Step 3.5: Migrate site #4 (overlay Offset Final)**

BEFORE (around line 1428):
```tsx
{/* End offset */}
<div className="flex items-center gap-3">
  <Label className="text-xs font-medium text-subtle w-20 flex-shrink-0">Offset Final</Label>
  <input
    type="range"
    min={0}
    max={10}
    value={overlayEndOffsetSec}
    onChange={(e) => setOverlayEndOffsetSec(Number(e.target.value))}
    className="flex-1 accent-brand"
  />
  <span className="text-xs text-subtle w-6 text-right">{overlayEndOffsetSec}s</span>
</div>
```

AFTER:
```tsx
{/* End offset */}
<SliderRow
  label="Offset Final"
  min={0}
  max={10}
  value={overlayEndOffsetSec}
  onChange={setOverlayEndOffsetSec}
  format={(v) => `${v}s`}
/>
```

- [ ] **Step 3.6: Migrate site #5 (watermark Opacidade)**

BEFORE (around line 1554):
```tsx
{/* Opacity slider */}
<div className="flex items-center gap-3">
  <Label className="text-xs font-medium text-subtle w-20 flex-shrink-0">Opacidade</Label>
  <input
    type="range"
    min={0}
    max={100}
    value={brandingLogoOpacity}
    onChange={(e) => setBrandingLogoOpacity(Number(e.target.value))}
    className="flex-1 accent-brand"
  />
  <span className="text-xs text-subtle w-8 text-right">{brandingLogoOpacity}%</span>
</div>
```

AFTER:
```tsx
{/* Opacity slider */}
<SliderRow
  label="Opacidade"
  min={0}
  max={100}
  value={brandingLogoOpacity}
  onChange={setBrandingLogoOpacity}
  format={(v) => `${v}%`}
/>
```

- [ ] **Step 3.7: Migrate site #6 (watermark Tamanho)**

BEFORE (around line 1568):
```tsx
{/* Scale slider */}
<div className="flex items-center gap-3">
  <Label className="text-xs font-medium text-subtle w-20 flex-shrink-0">Tamanho</Label>
  <input
    type="range"
    min={5}
    max={30}
    value={brandingLogoScale}
    onChange={(e) => setBrandingLogoScale(Number(e.target.value))}
    className="flex-1 accent-brand"
  />
  <span className="text-xs text-subtle w-8 text-right">{brandingLogoScale}%</span>
</div>
```

AFTER:
```tsx
{/* Scale slider */}
<SliderRow
  label="Tamanho"
  min={5}
  max={30}
  value={brandingLogoScale}
  onChange={setBrandingLogoScale}
  format={(v) => `${v}%`}
/>
```

- [ ] **Step 3.8: Migrate site #7 (music Volume)**

BEFORE (around line 1736):
```tsx
{/* Volume slider */}
<div className="flex items-center gap-3">
  <Label className="text-xs font-medium text-subtle w-16 flex-shrink-0">Volume</Label>
  <input
    type="range"
    min={0}
    max={30}
    value={musicVolumePct}
    onChange={(e) => setMusicVolumePct(Number(e.target.value))}
    className="flex-1 accent-brand"
  />
  <span className="text-xs text-subtle w-8 text-right">{musicVolumePct}%</span>
</div>
```

AFTER:
```tsx
{/* Volume slider */}
<SliderRow
  label="Volume"
  min={0}
  max={30}
  value={musicVolumePct}
  onChange={setMusicVolumePct}
  format={(v) => `${v}%`}
/>
```

- [ ] **Step 3.9: Migrate site #8 (captions Tamanho)**

BEFORE (around line 1836):
```tsx
<div className="flex items-center gap-3">
  <Label className="text-xs font-medium text-subtle w-20 flex-shrink-0">Tamanho</Label>
  <input
    type="range"
    min={12}
    max={96}
    value={captionsFontSize}
    onChange={(e) => setCaptionsFontSize(Number(e.target.value))}
    className="flex-1 accent-brand"
  />
  <span className="text-xs text-subtle w-8 text-right">{captionsFontSize}px</span>
</div>
```

AFTER:
```tsx
<SliderRow
  label="Tamanho"
  min={12}
  max={96}
  value={captionsFontSize}
  onChange={setCaptionsFontSize}
  format={(v) => `${v}px`}
/>
```

- [ ] **Step 3.10: Migrate site #9 (captions outline Espessura)**

BEFORE (around line 1897 inside `{captionsOutlineEnabled && …}`):
```tsx
<div className="flex items-center gap-3">
  <Label className="text-xs font-medium text-subtle w-16 flex-shrink-0">Espessura</Label>
  <input
    type="range"
    min={1}
    max={10}
    value={captionsOutlineWidth}
    onChange={(e) => setCaptionsOutlineWidth(Number(e.target.value))}
    className="flex-1 accent-brand"
  />
  <span className="text-xs text-subtle w-4">{captionsOutlineWidth}</span>
</div>
```

This one sits inside an already-`ml-6` parent block (the outline sub-panel). Keep the existing `flex items-center gap-3` wrapper intact? No — migrate the slider row itself. The `ml-6` parent comes from the surrounding `<div className="ml-6 space-y-2">`, not this row, so we just replace the row:

AFTER:
```tsx
<SliderRow
  label="Espessura"
  min={1}
  max={10}
  value={captionsOutlineWidth}
  onChange={setCaptionsOutlineWidth}
/>
```

- [ ] **Step 3.11: Migrate site #10 (captions shadow Distância)**

BEFORE (around line 1935 inside `{captionsShadowEnabled && …}`):
```tsx
<div className="flex items-center gap-3">
  <Label className="text-xs font-medium text-subtle w-16 flex-shrink-0">Distância</Label>
  <input
    type="range"
    min={0}
    max={20}
    value={captionsShadowDistance}
    onChange={(e) => setCaptionsShadowDistance(Number(e.target.value))}
    className="flex-1 accent-brand"
  />
  <span className="text-xs text-subtle w-4">{captionsShadowDistance}</span>
</div>
```

AFTER:
```tsx
<SliderRow
  label="Distância"
  min={0}
  max={20}
  value={captionsShadowDistance}
  onChange={setCaptionsShadowDistance}
/>
```

- [ ] **Step 3.12: Migrate site #11 (captions Offset Vertical)**

BEFORE (around line 1953):
```tsx
{/* Vertical offset */}
<div className="flex items-center gap-3">
  <Label className="text-xs font-medium text-subtle w-20 flex-shrink-0">Offset Vertical</Label>
  <input
    type="range"
    min={-100}
    max={100}
    value={captionsVerticalOffset}
    onChange={(e) => setCaptionsVerticalOffset(Number(e.target.value))}
    className="flex-1 accent-brand"
  />
  <span className="text-xs text-subtle w-8 text-right">{captionsVerticalOffset}</span>
</div>
```

AFTER:
```tsx
{/* Vertical offset */}
<SliderRow
  label="Offset Vertical"
  min={-100}
  max={100}
  value={captionsVerticalOffset}
  onChange={setCaptionsVerticalOffset}
/>
```

### Step 3.13: Verify

- [ ] **Step 3.13: Confirm all `type="range"` sites are gone**

Run:
```sh
grep -n 'type="range"' apps/web/src/components/pipeline/steps/StepMode.tsx
```
Expected: no output.

- [ ] **Step 3.14: Confirm no orphan slider scaffolding remains**

Run:
```sh
grep -n 'accent-brand' apps/web/src/components/pipeline/steps/StepMode.tsx
```
Expected: no output. (`accent-brand` was only used on the raw range inputs; SliderRow handles it internally.)

- [ ] **Step 3.15: Run vitest**

Run:
```sh
cd apps/web && npx vitest run
```
Expected: all tests PASS.

- [ ] **Step 3.16: Commit**

```sh
git add apps/web/src/components/pipeline/steps/StepMode.tsx
git commit -m "$(cat <<'EOF'
refactor(ui): adopt SliderRow in StepMode

Replace 11 hand-rolled `<input type="range">` rows in StepMode.tsx
with the Phase 3 <SliderRow> primitive, using `format` for `%`/`s`/`px`
value rendering.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
git status
```
Expected: commit created, working tree clean.

---

## Final Verification

- [ ] **Step 4.1: Verify final file shape**

Run:
```sh
grep -nc 'role="switch"\|type="range"\|grid grid-cols-2 gap-2' apps/web/src/components/pipeline/steps/StepMode.tsx
```
Expected: `0`.

- [ ] **Step 4.2: Verify three commits on main**

Run:
```sh
git log --oneline -3
```
Expected (top three commits in this order):
```
xxxxxxx refactor(ui): adopt SliderRow in StepMode
xxxxxxx refactor(ui): adopt SegmentedControl in StepMode
xxxxxxx refactor(ui): adopt SettingRow + Switch in StepMode
```

- [ ] **Step 4.3: Hand back to coordinator for `npx next build`**

Signal to the coordinator that all three commits have landed and vitest is green. The coordinator runs `cd apps/web && npx next build` to verify the type-checker is happy with the refactor. If `next build` surfaces errors, fix them in a follow-up commit — do not amend the Phase 4 commits.

---

## Rollback Plan

If `next build` fails on any commit, fix in a new commit. If vitest fails mid-task, use `git diff` to inspect the most recent edits, revert the failing site with `git checkout -- <file>` (WARNING: this is a destructive action — only do this before any commit, and only if the work since the last commit is trivial to redo), then re-apply from the exemplar.

If a commit needs to be backed out entirely:
```sh
git revert <commit-sha>
```
This creates a new commit that undoes the migration. Never force-push main.
