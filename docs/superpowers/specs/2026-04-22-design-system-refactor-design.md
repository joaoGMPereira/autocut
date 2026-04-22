# Design System Refactor — Extract Components from Inline Markup

**Date:** 2026-04-22
**Owner:** Joao Gabriel
**Status:** Approved (brainstorm), ready for implementation plan

## Goal

Migrate AutoCut feature screens from inline markup/styling to reusable Design System primitives in `apps/web/src/components/ui/`. Codebase already uses semantic CSS tokens (`bg-brand`, `text-heading`, `bg-surface`, `text-subtle`, etc.) — this spec extracts the next layer: the components themselves.

The reference Design System bundle lives at `/Users/joaogabriel/Downloads/autocut-design-system/project/` (HTML previews, `Primitives.jsx` reference, `colors_and_type.css` token map, `README.md` guidelines).

## Non-goals

- App shell (sidebar, `layout.tsx`, root layout)
- Logo/wordmark migration to the DS-defined glyph
- Tokens (`globals.css` is locked)
- Data-fetching, state, or business-logic refactors
- Storybook or visual-regression infrastructure
- Consumer-level tests (only new primitives are tested)
- Pipeline-specialized files: `StepRail`, `LogStream`, `GateActionBar`, `ContextPanel`, `PipelineShell`, `ActiveStepArea`, `BackConfirmDialog`
- Migrating `<select>` elements to shadcn `<Select>` (deferred follow-up)

## Standing decisions (from brainstorming)

1. **Token cleanup scope: opportunistic.** When a file is touched for component extraction, also normalize lingering raw colors (`bg-zinc-*`, `text-zinc-*`, `border-zinc-*`, `bg-emerald-400/20`, etc.) to semantic tokens in that same file. Files we do not touch stay as-is. No follow-up sweep planned.
2. **PageHeader sizing: single 32px** (`text-[32px]`). Settings, queue, and channels page H1s grow from 28px → 32px. Visual regression accepted.
3. **Input strategy: rewrite shadcn `<Input>` in place** to match the AutoCut soft-bg + brand-focus pattern. Single source of truth; existing shadcn `<Input>` consumers auto-inherit the new look.
4. **Tabs: migrate queue's custom pill-tabs to shadcn `<Tabs>`** with a new `count` slot on `<TabsTrigger>`. Visual regression accepted (queue tabs change from custom pills to shadcn underlined tabs).
5. **Pipeline: full sweep included.** Pipeline files consume the new primitives. Specialized pipeline components stay in `pipeline/` (not promoted to `ui/`).
6. **Testing: render tests for new/extended primitives only.** Vitest + `@testing-library/react`. No consumer tests.
7. **Status badges: just extend `<Badge>` with variants.** No `<StatusBadge>` wrapper. Each consumer maps its domain status → variant inline.
8. **FormField: yes, with `<Label>` exported as escape hatch** for inline label cases (checkbox rows, dropdown-on-same-line).

## Architecture

- All primitives live in `apps/web/src/components/ui/`. No barrel re-exports.
- shadcn primitives modified in place: `badge.tsx`, `input.tsx`, `tabs.tsx`.
- New primitives: `page-header.tsx`, `form-field.tsx`.
- Tests colocated: `badge.test.tsx`, `input.test.tsx`, `tabs.test.tsx`, `page-header.test.tsx`, `form-field.test.tsx`.
- Pipeline-specialized files stay where they are.

## Component inventory

### Modified shadcn primitives

#### `Badge` — add 4 soft-fill variants

DS pattern: `bg-{token}/10 + border-{token}/20 + text-{token}`.

| variant | bg | border | text |
|---|---|---|---|
| `success` | `bg-success/10` | `border-success/20` | `text-success` |
| `warning` | `bg-warning/15` | `border-warning/40` | `text-warning` |
| `info` | `bg-info/10` | `border-info/20` | `text-info` |
| `brand` | `bg-brand/10` | `border-brand/40` | `text-brand` |

Existing variants (`default/secondary/destructive/outline/ghost/link`) untouched. `destructive` stays solid red (intentional — alerts and form errors should pop).

#### `Input` — rewrite default styles

```css
base:     h-9 w-full rounded-lg bg-surface border border-border px-3 py-2
          text-sm text-foreground placeholder:text-subtle
focus:    focus-visible:border-brand/60 focus-visible:ring-brand/30 focus-visible:ring-[3px]
invalid:  aria-invalid:border-destructive aria-invalid:ring-destructive/30
disabled: disabled:opacity-50 disabled:cursor-not-allowed
```

Mono usage = consumer passes `className="font-mono"`. No `variant` prop.

#### `Tabs` → `TabsTrigger` — add `count?: number` prop

Renders pill `ml-1.5 rounded-full px-1.5 py-0.5 text-[10px]` after the label when `count > 0`.

- Active state: pill bg `bg-primary-foreground/20`, text inherits.
- Inactive: pill bg `bg-muted`, text `text-muted-foreground`.
- Hidden when `count === 0` or `undefined`.

### New primitives

#### `PageHeader`

```tsx
<PageHeader
  title="Settings"
  description="Configure tools, preferences, models, and connected channels."
  actions={<Button size="sm">…</Button>}
/>
```

- Title: `font-display text-[32px] font-bold tracking-[-0.02em] text-heading leading-tight`
- Description: `text-sm text-subtle mt-1`
- Layout: `flex items-start justify-between gap-4`; collapses to single column when no `actions`
- Renders an `<h1>` for the title (one per page is the rule)

#### `FormField`

```tsx
<FormField label="Start at" description="…" error={err} required>
  <Input value={x} onChange={…} />
</FormField>
```

- Auto-generates `id` via `React.useId()` if not provided
- Wires `htmlFor` on the label and `aria-describedby` on the control
- Layout: `flex flex-col gap-1.5`
- Label: `text-xs font-medium text-subtle` (canonical label style — supersedes the 3 inline variants)
- Required marker: `<span className="text-destructive">*</span>`
- Description: `text-xs text-subtle`
- Error: `text-xs text-destructive` (replaces description visually when set; ARIA still links both)
- shadcn `<Label>` stays exported for inline cases

### Pipeline scope

In-scope for **consuming** new primitives:
- `StepRailItem`, `DownloadInfoCard`, `ClipCard`, `HighlightCard`, `ClipReviewCard`, `ChannelInfoCard`, `ThumbnailStylePicker`, `SubStepProgress`

Stay as-is (specialized, single-use):
- `StepRail`, `LogStream`, `GateActionBar`, `ContextPanel`, `PipelineShell`, `ActiveStepArea`, `BackConfirmDialog`

## Migration plan

### Commits A–D — primitives (sequential, single agent)

| # | Commit | Files |
|---|---|---|
| A | `feat(ui): extend Badge with success/warning/info/brand soft-fill variants` | `badge.tsx`, `badge.test.tsx` |
| B | `feat(ui): rewrite Input with surface-bg + brand focus ring` | `input.tsx`, `input.test.tsx`. Audit existing shadcn `<Input>` consumers (AddChannelForm, settings rows) — visual spot-check before commit. |
| C | `feat(ui): add PageHeader and FormField primitives` | `page-header.tsx`, `form-field.tsx`, tests |
| D | `feat(ui): add count slot to TabsTrigger` | `tabs.tsx`, `tabs.test.tsx` |

After each: `npx next build` + `npx vitest run` must pass.

If subagent E4 (settings) discovers 3+ clean repetitions of a settings-row pattern during its audit, add `setting-row.tsx` as a Commit C addendum before E4 dispatches.

### Commits E1–E10 — consumer migration (parallel subagents)

Dispatched as a single message with all subagent calls so they run concurrently. File-disjoint scopes prevent merge conflicts. No worktrees. Each commits with `refactor({area}): adopt design system primitives`.

| # | Area | Files | Model |
|---|---|---|---|
| E1 | shorts | `ShortsForm.tsx` (+ `ShortsResultList` if dirty) | Sonnet |
| E2 | queue | `app/queue/page.tsx`, `BulkScheduleModal.tsx`, `QueueItemCard.tsx`, `ReviewScheduleModal.tsx` | Sonnet |
| E3 | channels | `app/channels/page.tsx`, `ChannelCard.tsx`, `CommentSyncSheet.tsx` | Sonnet |
| E4 | settings | `app/settings/page.tsx`, `AppSettingsSection.tsx`, `ToolsSection.tsx`, `ModelsSection.tsx`, `ChannelsSection.tsx`, `ChannelAuthSection.tsx`, `OAuthProfilesSection.tsx` | **Opus** |
| E5 | dashboard | `app/page.tsx`, `DashboardClient.tsx` | default |
| E6 | history | `RunDetailPanel.tsx`, `RunFilters.tsx`, `RunsTable.tsx`, `StepRow.tsx` | Sonnet |
| E7 | dispatcher | `DispatcherBar.tsx` | default |
| E8 | post-opt + thumbnail | `LogoTab.tsx`, `TransitionsTab.tsx`, `BackgroundsGallery.tsx` | default |
| E9 | setup | `ToolRow.tsx`, `WhisperToolRow.tsx` | default |
| E10 | pipeline | 16 files in `pipeline/` excluding the specialized list | **Opus** |

### Subagent contract

Each subagent prompt includes:

- Path to this spec doc
- File list it owns (no overlap with sibling agents)
- Standing decisions (1–8 above)
- Required steps:
  1. Read the relevant DS preview HTML files in `/Users/joaogabriel/Downloads/autocut-design-system/project/preview/`
  2. Audit assigned files for inline patterns matching the new primitives
  3. Migrate inline patterns → primitives
  4. Normalize lingering `bg-zinc-*` / `text-zinc-*` / `border-zinc-*` / raw color stops to semantic tokens (opportunistic per decision 1)
  5. Run `npx next build` and `npx vitest run`
  6. Commit with `refactor({area}): adopt design system primitives`
  7. Return summary
- Forbidden: editing files outside its scope, editing `globals.css`, inventing new primitives (must use what Commits A–D shipped)

### Cross-area dependency

`ChannelAvatar` is exported from `ChannelCard.tsx` and consumed by `QueueItemCard`. Subagent E3 (channels) owns `ChannelCard.tsx` and must not change `ChannelAvatar`'s export shape. Subagent E2 (queue) consumes the export only. Both subagent prompts call this out.

### Failure handling

If a subagent's commit fails build/tests: that area's commit is reverted, the subagent reports back, the single coordinating agent investigates and re-runs (with fix or scope reduction). Other subagent commits stay landed.

### Commit F — final pass (single agent)

`chore(ui): final grep + verification pass`

Grep for residual inline patterns:

- `font-display text-\[`  → expect 0 hits outside `page-header.tsx`
- `rounded-lg bg-surface border border-border` → expect 0 hits in form/input contexts
- `inline-flex items-center.*rounded-full.*bg-(emerald|amber|zinc|red|blue)-` → expect 0 hits in feature files
- `text-xs (text-subtle mb-1 block|font-medium text-(caption|muted-foreground))` → expect 0 hits in feature files
- `bg-zinc-`, `text-zinc-`, `border-zinc-` in **touched** files only → expect 0

Address any straggler in this same commit. Final `npx next build` + `npx vitest run`.

## Testing strategy

Render tests for new/extended primitives only. Stack: vitest + `@testing-library/react` (already in repo, see `switch.test.tsx`). No new deps. ~5–10 assertions per file, ~200 LOC total.

| Test file | Assertions |
|---|---|
| `badge.test.tsx` | Each new variant renders `data-variant=…` + class list contains expected token classes. Default variant unchanged. |
| `input.test.tsx` | Default render has `bg-surface border-border`. Focus ring class present. `aria-invalid=true` triggers destructive border + ring classes. Disabled state has `opacity-50`. |
| `tabs.test.tsx` | `<TabsTrigger count={3}>` renders count pill with `3`. `count={0}` and `count={undefined}` hide pill. Active trigger pill uses inverted bg class. |
| `page-header.test.tsx` | Renders `<h1>` with title text. Description renders when provided, omitted when not. Actions slot renders in right column. |
| `form-field.test.tsx` | Label `htmlFor` matches child input `id` (auto-generated via `useId` if not supplied). `aria-describedby` wires to description and/or error. Required indicator (`*`) renders only when `required`. Error replaces description visually when both present. |

`npx vitest run` is the gate at every commit (A–F).

## Risks

1. **Input rewrite changes appearance of existing shadcn `<Input>` consumers.** Mitigation: visual spot-check after Commit B before continuing.
2. **Pipeline blast radius (16 files).** Mitigation: dedicated Opus subagent + single atomic commit so failure reverts cleanly. Specialized files explicitly out of scope.
3. **Subagent visual drift.** Mitigation: spec mandates "no behavior or visual change beyond changes mandated by primitive substitution." Verification = grep + build + manual spot-check at Commit F.
4. **Concurrent commits to `main`.** Mitigation: file-disjoint scopes; git serializes any near-simultaneous commits.
5. **`ChannelAvatar` cross-area dep.** Mitigation: spec calls this out in both E2 and E3 prompts.
6. **Accepted regressions:** settings/queue/channels page titles 28px → 32px (decision 2); queue tabs custom pills → shadcn underlined tabs (decision 4).

## Known follow-ups

- `Badge.destructive` stays solid red while siblings are soft. Documented as intentional. Track for follow-up if a future screen needs a soft error pill.
- `ShortsForm`'s native `<select>` elements (aspect ratio, language) keep their inline classes. Migration to shadcn `<Select>` is a separate, more invasive change.
- `AppSettingsSection` (8.8K) may reveal a `<SettingRow>` pattern. If E4 finds 3+ clean repetitions, add `setting-row.tsx` as a Commit C addendum.

## Verification

1. Build: `npx next build` passes at every commit.
2. Tests: `npx vitest run` passes at every commit.
3. Greps in Commit F return zero hits.
4. Manual spot-check after Commit F: dashboard, shorts, queue (queue + history tabs), channels, settings (each tab), pipeline (each step), history, post-opt, thumbnail, setup.
