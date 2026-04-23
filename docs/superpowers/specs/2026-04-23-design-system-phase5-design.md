# Design System Phase 5 — Primitive Extraction From StepMode Leftovers

**Date:** 2026-04-23
**Owner:** Joao Gabriel
**Status:** Approved, ready for implementation plan
**Depends on:** Phases 1–4 (`SectionPanel`, `SettingRow`, `SegmentedControl` flat+card, `SliderRow`, `Switch`, Radix `Input`, `Button`)

## Context

Phase 4 (commit range `d09f30b..67ba2ce`) migrated 32 sites in `apps/web/src/components/pipeline/steps/StepMode.tsx` to Phase 3 primitives and left 13 sites inline because they did not fit any Phase 3 primitive. Phase 5 extracts the four of those 13 patterns whose semantics repeat and ships the migration.

Reuse audit against the rest of `apps/web/src/` was performed before scoping. Results drove the ship/kill/defer call for each candidate — four ship.

## Phase 5 Candidates — Decisions

| # | Candidate | StepMode sites | Other consumers | Decision |
|---|-----------|----------------|-----------------|----------|
| 1 | Wrap variant of `SegmentedControl` | 5 | 0 | **SHIP** — `wrap?: boolean` prop on existing flat variant |
| 2 | `ToggleGroup` (multi-select) | 1 | 0 | **SHIP** — new primitive, clean types |
| 3 | `InputWithAction` | 1 | 1 (ChannelConfigSheet pattern-add) | **SHIP** — 2 consumers, same Enter-to-submit contract |
| 4 | `PanelHeader` | 2 | 0 | **SHIP** — scoped to Preview section, still a named concept |

## Non-goals

- Preset carousel (`StepMode.tsx:~720`) — unique overflow-x-auto w/ dynamic delete; stays inline.
- Mode-cards grid — bullet lists + checkmark badges; not a button-group shape.
- Overlay #N header — per-overlay action cluster; not a PanelHeader shape.
- Thumbnail card (`StepMode.tsx:~705`) — horizontal-flex card, no header row.
- Tokens (`apps/web/src/app/globals.css` locked).
- Renaming existing primitives.
- Backfilling the same primitives into other pipeline steps (out-of-scope files identified below stay untouched).

## Standing decisions (carried from Phases 1–4)

1. Work directly on `main`. No PR, no worktree.
2. Conventional Commits with trailer `Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>` (matches Phases 3 & 4 trailer convention).
3. **Zero behavior change.** Primitives produce identical handler signatures. All `data-testid`s, state setters, conditional branches preserved.
4. TDD cycle per primitive: `*.test.tsx` sibling with Vitest + `@testing-library/react`.
5. Per-commit gate from `apps/web/`: `npx vitest run`. Never `--no-verify`. Coordinator runs `npx next build` externally after each commit.
6. Tokens locked — do NOT edit `apps/web/src/app/globals.css`.
7. Next.js 16 caveat: all Phase 5 surface is pure client components; no Next.js APIs touched.

## New / extended primitives

### A. `SegmentedControl` — add `wrap?: boolean` prop

File: `apps/web/src/components/ui/segmented-control.tsx`

Adds a third layout mode to the `flat` variant. When `wrap` is true:
- Container: `flex flex-wrap gap-2` (instead of `flex gap-2`).
- Buttons drop `flex-1` so they size to content.

All other flat styling (`rounded px-2 py-1.5 text-xs`, active/inactive colors, `aria-pressed`) is preserved. `card` variant unchanged. `wrap` is ignored when `variant="card"`.

Signature:
```ts
interface SegmentedControlProps<T extends string> {
  options: SegmentedOption<T>[]
  value: T
  onChange: (value: T) => void
  variant?: 'flat' | 'card'
  wrap?: boolean          // NEW — flat only
  className?: string
}
```

New tests (appended to existing `segmented-control.test.tsx`):
- `wrap=true` → root carries `flex-wrap`; buttons do NOT carry `flex-1`.
- `wrap=false` (default) → root carries `flex gap-2`; buttons carry `flex-1` (regression guard).
- `wrap` has no effect when `variant="card"` (root still `grid`).

### B. `ToggleGroup` — `apps/web/src/components/ui/toggle-group.tsx`

Multi-select sibling to `SegmentedControl` flat. Each option is an independently toggleable boolean; no single "active" value.

```tsx
import * as React from 'react'
import { cn } from '@/lib/utils'

interface ToggleGroupOption<K extends string> {
  key: K
  label: string
}

interface ToggleGroupProps<K extends string> {
  options: ToggleGroupOption<K>[]
  value: Record<K, boolean>
  onChange: (key: K, next: boolean) => void
  className?: string
}

function ToggleGroup<K extends string>({
  options,
  value,
  onChange,
  className,
}: ToggleGroupProps<K>) {
  return (
    <div
      data-slot="toggle-group"
      className={cn('flex gap-2', className)}
    >
      {options.map((opt) => {
        const active = value[opt.key]
        return (
          <button
            key={opt.key}
            type="button"
            onClick={() => onChange(opt.key, !active)}
            aria-pressed={active}
            className={cn(
              'flex-1 rounded px-2 py-1.5 text-xs transition-colors',
              active
                ? 'bg-brand text-brand-foreground font-medium'
                : 'bg-surface text-subtle hover:bg-surface/80',
            )}
          >
            {opt.label}
          </button>
        )
      })}
    </div>
  )
}

export { ToggleGroup }
export type { ToggleGroupProps, ToggleGroupOption }
```

Tests (`toggle-group.test.tsx`):
- Renders every option label.
- Active options carry `bg-brand`; inactive carry `bg-surface`.
- `aria-pressed` tracks value map per key.
- Clicking an active key fires `onChange(key, false)`.
- Clicking an inactive key fires `onChange(key, true)`.
- Independent toggles — flipping one key does not mutate others (verified via controlled test harness).
- `className` merges.

**Why not extend `SegmentedControl` with a `multiple` mode:** the value/onChange type contract flips from `T` / `(T) => void` to `Record<K, boolean>` / `(K, boolean) => void`. A union would poison the caller ergonomics of the 6 existing SegmentedControl consumers. Cleaner to ship a separate 40-line primitive.

### C. `InputWithAction` — `apps/web/src/components/ui/input-with-action.tsx`

Paired `<Input>` + trailing `<Button>` with Enter-to-submit semantics. Two real consumers today (StepMode preset save + ChannelConfigSheet pattern add) share the exact same shape: controlled text input, Button on the right, Enter key triggers the same handler as clicking the button, submit disabled matches `!value.trim()`.

```tsx
import * as React from 'react'
import { cn } from '@/lib/utils'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'

interface InputWithActionProps {
  value: string
  onValueChange: (value: string) => void
  onSubmit: () => void
  actionLabel: React.ReactNode
  actionDisabled?: boolean
  placeholder?: string
  inputClassName?: string
  actionVariant?: React.ComponentProps<typeof Button>['variant']
  actionSize?: React.ComponentProps<typeof Button>['size']
  className?: string
  'data-testid'?: string
}

function InputWithAction({
  value,
  onValueChange,
  onSubmit,
  actionLabel,
  actionDisabled,
  placeholder,
  inputClassName,
  actionVariant = 'outline',
  actionSize = 'sm',
  className,
  ...rest
}: InputWithActionProps) {
  return (
    <div
      data-slot="input-with-action"
      className={cn('flex gap-2', className)}
    >
      <Input
        type="text"
        value={value}
        onChange={(e) => onValueChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && !actionDisabled) {
            onSubmit()
          }
        }}
        placeholder={placeholder}
        className={cn('flex-1', inputClassName)}
        data-testid={rest['data-testid']}
      />
      <Button
        type="button"
        onClick={onSubmit}
        disabled={actionDisabled}
        variant={actionVariant}
        size={actionSize}
        className="text-xs"
      >
        {actionLabel}
      </Button>
    </div>
  )
}

export { InputWithAction }
export type { InputWithActionProps }
```

Tests (`input-with-action.test.tsx`):
- Renders input with current value and button with label.
- Typing fires `onValueChange` with new string.
- Pressing Enter fires `onSubmit` once; Enter is ignored when `actionDisabled` is true.
- Clicking the button fires `onSubmit`; button is disabled when `actionDisabled` is true.
- `className` merges on root; `inputClassName` merges on input.
- Non-Enter keys (e.g. `Escape`, `a`) do NOT fire `onSubmit`.

### D. `PanelHeader` — `apps/web/src/components/ui/panel-header.tsx`

Header row for inline cards (not `SectionPanel`). Two StepMode sites today; future consumer (mode-submit bar, overlay banners) can opt in without re-invention.

```tsx
import * as React from 'react'
import { cn } from '@/lib/utils'

interface PanelHeaderProps {
  title: React.ReactNode
  actions?: React.ReactNode
  className?: string
}

function PanelHeader({ title, actions, className }: PanelHeaderProps) {
  return (
    <div
      data-slot="panel-header"
      className={cn('flex items-center justify-between', className)}
    >
      <h3 className="text-sm font-medium text-prose">{title}</h3>
      {actions && <div className="shrink-0">{actions}</div>}
    </div>
  )
}

export { PanelHeader }
export type { PanelHeaderProps }
```

Tests (`panel-header.test.tsx`):
- Renders title as `h3` with `text-sm font-medium text-prose`.
- Actions render when provided; action wrapper absent when not.
- `className` merges on root.
- Renders ReactNode title (not just string).

**Why not reuse `SectionPanel`:** Preview is a single free-form card with mixed non-titled content (progress bar, video player, error text). `SectionPanel` imposes eyebrow-title + `space-y-3` child rhythm that would fight the Preview layout. `PanelHeader` is a strictly-header primitive with zero container semantics.

## Consumer migrations

### E. `refactor(ui): adopt SegmentedControl wrap in StepMode`

5 sites in `StepMode.tsx`. Target Clip Duration and Segment Duration are dynamic — the IIFE that computes options stays, it just returns an `options` array passed to `<SegmentedControl wrap>` instead of mapping buttons inline.

| Site | Context | Options source | Drift |
|------|---------|----------------|-------|
| `~849` | Target Clip Duration (AI) | IIFE — dynamic 1–3 w/ fallback `[1020, 1560, 3120]` | none |
| `~886` | Minimum Clip Duration (AI) | Static `[60,120,300,480,600,900]` | none |
| `~955` | Segment Duration (Longform) | IIFE — dynamic 1–3 w/ fallback | none |
| `~995` | Min Part Duration (Longform) | Static `[0,60,120,300,480,600,900]` | none |
| `~1312` | Chroma Key | Static `['none','green','black','white','blue']` | **capitalize drop** — pre-capitalize labels in options array (`'Nenhum','Green','Black','White','Blue'`); `px-3` → `px-2` |

All four numeric sites (Target Clip, Min Clip, Segment, Min Part) follow the Phase 4 Highlight Threshold cast pattern: option `value` is the stringified number, caller stringifies on read and `Number()`s on write. Explicit literal union for type safety:

```tsx
<SegmentedControl<`${number}`>
  wrap
  value={String(minDurationSecs) as `${number}`}
  onChange={(v) => setMinDurationSecs(Number(v))}
  options={[
    { value: '60',  label: '1 min' },
    { value: '120', label: '2 min' },
    // …
  ]}
/>
```

Chroma Key uses a literal string union directly (no numeric cast): `'none' | 'green' | 'black' | 'white' | 'blue'`.

Label rendering rules:
- AI Target Clip / Longform Segment / Min Clip / Min Part: option `label` is the human-readable minutes string (e.g. `"17 min"`, `"Off"`).
- Chroma Key: option `label` is pre-capitalized text; no CSS transform needed.

### F. `refactor(ui): adopt ToggleGroup in StepMode captions`

1 site (`StepMode.tsx:~1550`). Replaces the 3-button inline group (bold/italic/uppercase) with:
```tsx
<ToggleGroup<'bold' | 'italic' | 'uppercase'>
  options={[
    { key: 'bold', label: 'Negrito' },
    { key: 'italic', label: 'Itálico' },
    { key: 'uppercase', label: 'Maiúsculas' },
  ]}
  value={{ bold: captionsBold, italic: captionsItalic, uppercase: captionsUppercase }}
  onChange={(k, next) => {
    if (k === 'bold') setCaptionsBold(next)
    else if (k === 'italic') setCaptionsItalic(next)
    else if (k === 'uppercase') setCaptionsUppercase(next)
  }}
/>
```

No visual drift — DOM and classes identical to the inline version.

### G. `refactor(ui,channels): adopt InputWithAction in preset save and pattern add`

Two files, one commit.

| File | Line (approx) | Action label | Notes |
|------|---------------|--------------|-------|
| `apps/web/src/components/pipeline/steps/StepMode.tsx` | `~1720` | `nameExists ? 'Atualizar' : 'Criar'` | `onValueChange` also runs `setActivePresetName` lookup — wrap both setters in the caller's handler |
| `apps/web/src/components/channels/ChannelConfigSheet.tsx` | `~317` | `adding ? 'Adding…' : 'Add'` | `size='sm'` matches current button |

Behavior preservation:
- StepMode side-effect (active preset lookup) stays in the caller's `onValueChange` handler.
- ChannelConfigSheet `placeholder="artist name or pattern…"` preserved.
- `data-testid`s remain on original slots where present.

### H. `refactor(ui): adopt PanelHeader in StepMode Preview`

2 sites inside the Preview card in `StepMode.tsx` (approx lines 1750 and 1779). Preview card wrapper (`rounded-lg border border-border bg-card p-4` + `space-y-3`) stays — it is not a `SectionPanel` shape (see rationale above). Only the two header rows swap:

```tsx
// Downloading state
<PanelHeader
  title="Preview"
  actions={<span className="text-xs text-subtle">Aguardando download...</span>}
/>

// Ready state
<PanelHeader
  title="Preview"
  actions={
    <Button onClick={...} disabled={...} variant="secondary" size="sm" className="text-xs">
      {previewStatus === 'generating' ? 'Generating…' : previewStatus === 'ready' ? 'Regenerate Preview' : 'Generate Preview'}
    </Button>
  }
/>
```

No visual drift.

## Commit sequence

| # | Commit | Files | Gate |
|---|--------|-------|------|
| A | `feat(ui): add wrap prop to SegmentedControl` | `segmented-control.tsx` + `segmented-control.test.tsx` | `npx vitest run` |
| B | `feat(ui): add ToggleGroup primitive` | `toggle-group.tsx` + `toggle-group.test.tsx` | `npx vitest run` |
| C | `feat(ui): add InputWithAction primitive` | `input-with-action.tsx` + `input-with-action.test.tsx` | `npx vitest run` |
| D | `feat(ui): add PanelHeader primitive` | `panel-header.tsx` + `panel-header.test.tsx` | `npx vitest run` |
| E | `refactor(ui): adopt SegmentedControl wrap in StepMode` | `StepMode.tsx` (5 sites) | `npx vitest run` |
| F | `refactor(ui): adopt ToggleGroup in StepMode captions` | `StepMode.tsx` (1 site) | `npx vitest run` |
| G | `refactor(ui,channels): adopt InputWithAction` | `StepMode.tsx` + `ChannelConfigSheet.tsx` | `npx vitest run` |
| H | `refactor(ui): adopt PanelHeader in StepMode Preview` | `StepMode.tsx` (2 sites) | `npx vitest run` |

Each commit carries trailer:

```
Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
```

Coordinator runs `npx next build` externally after each commit. If vitest fails, fix in-place and retry — never bypass with `--no-verify`. If tests pass locally but `next build` surfaces a type error, fix and land as a new follow-up commit (not `--amend`).

## Visual Drift (Accepted)

Phase 5 primitives are authoritative after they ship. The following drifts are intentional:

- **Chroma Key option group (Commit E, site `~1312`)**: current `rounded px-3 py-1.5 capitalize` → primitive `rounded px-2 py-1.5`; labels pre-capitalized in options (`'Nenhum','Green','Black','White','Blue'`). Aligns with the rest of the flat SegmentedControl family.
- **ToggleGroup captions row (Commit F)**: no drift — DOM identical, pure primitive extraction.
- **InputWithAction (Commit G)**: StepMode preset row currently uses `className="flex-1 h-8 text-xs"` on the Input; primitive uses `flex-1` + inheritance from shared `Input` defaults. Exact pixel match depends on `Input` base size — if the `h-8 text-xs` override is required, pass via `inputClassName`. **Required**: confirm with visual spot-check on `npm run dev` before merging Commit G.
- **PanelHeader (Commit H)**: no drift — wraps exact current DOM (`<div class="flex items-center justify-between"><h3 …>{title}</h3>{actions}</div>`).

## Behavior Preservation Contract

- **All** state setters, handlers, conditional branches preserved verbatim.
- **All** `data-testid` attributes remain on their original elements.
- `SegmentedControl wrap`: the dynamic IIFEs for Target Clip Duration / Segment Duration become pure `const options = (() => { … })()` expressions. Side effects are none — the original IIFEs were pure option computations.
- `ToggleGroup`: three independent setters replaced with a single `(key, next) =>` switch. Console logs (if any — check before migration) reattached to the new handler.
- `InputWithAction`: StepMode's `setActivePresetName` side-effect runs inside the new `onValueChange` handler; Enter-submit routes to `handleSavePreset` as today. ChannelConfigSheet's Enter-submit routes to `handleAdd`.
- `PanelHeader`: Preview card outer layout (`space-y-3`), progress bar, video player, error text all untouched.

## Scan-beyond-StepMode audit (decision log)

For each candidate, grep outside `StepMode.tsx` was run before deciding:

- **Candidate 1** — `grep 'bg-brand text-brand-foreground' src/ --include='*.tsx'`: zero hits outside `button.tsx`, `ChannelCard.tsx` checkmark badge (not a group), and StepMode. Verdict: ship as prop extension.
- **Candidate 2** — `grep` for multi-select toggle patterns: no other consumers. Verdict: ship (user directive) as separate primitive; 1 consumer justified because `SegmentedControl multiple` poisons types for 6 existing callers.
- **Candidate 3** — Input + Button + Enter patterns: `ChannelConfigSheet.tsx:317` matches exactly. `OAuthProfilesSection.tsx:139` (Input + Input + Button, no Enter), `StepThumbnailConfig.tsx:465` (Input + Button + Button, no Enter), `StepUrl.tsx:164` (Input in own block, Button in own block, `space-y-3` not `flex gap-2`) all excluded — different shapes. Verdict: ship with two consumers.
- **Candidate 4** — `flex items-center justify-between` + `<h3>` patterns: the other `<h3>` sites (`StepThumbnailConfig.tsx:297`, `:441`, `:509`; `LandscapeConfigPanel.tsx:23`; `AppSettingsSection.tsx:155`) are standalone headings with no right-side actions. Verdict: ship (user directive); two consumers justify a 15-line primitive.

## Critical files

**New / modified primitives:**
- `apps/web/src/components/ui/segmented-control.tsx` (extended)
- `apps/web/src/components/ui/segmented-control.test.tsx` (extended)
- `apps/web/src/components/ui/toggle-group.tsx` (new)
- `apps/web/src/components/ui/toggle-group.test.tsx` (new)
- `apps/web/src/components/ui/input-with-action.tsx` (new)
- `apps/web/src/components/ui/input-with-action.test.tsx` (new)
- `apps/web/src/components/ui/panel-header.tsx` (new)
- `apps/web/src/components/ui/panel-header.test.tsx` (new)

**Modified consumers:**
- `apps/web/src/components/pipeline/steps/StepMode.tsx` — Commits E, F, G, H.
- `apps/web/src/components/channels/ChannelConfigSheet.tsx` — Commit G.

## Verification end-to-end

1. Per-commit: `cd apps/web && npx vitest run` passes.
2. After each commit: coordinator runs `npx next build` — must pass.
3. After Commit H: from `apps/web/`, expect 0 hits outside primitives and tests:
   ```sh
   grep -rn 'flex-wrap gap-2.*bg-brand\|bg-brand text-brand-foreground font-medium.*rounded px' src/ --include='*.tsx'
   ```
4. Manual spot-check: `npm run dev` → open pipeline, exercise AI mode + Long Form mode + captions bold/italic/uppercase + preset save + Preview card in both download + ready states. Confirm no regressions.

## References

- Phase 3 spec: `docs/superpowers/specs/2026-04-22-design-system-phase3-design.md`
- Phase 4 spec: `docs/superpowers/specs/2026-04-23-design-system-phase4-design.md`
- Parent refactor spec: `docs/superpowers/specs/2026-04-22-design-system-refactor-design.md`
