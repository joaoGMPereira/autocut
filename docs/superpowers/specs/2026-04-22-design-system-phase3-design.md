# Design System Phase 3 — SettingRow, SegmentedControl, SliderRow + Tier 1 Consumer Migration

**Date:** 2026-04-22
**Owner:** Joao Gabriel
**Status:** Approved, ready for implementation plan

## Context

Phases 1 and 2 shipped foundational primitives (Badge, Input, PageHeader, FormField, Tabs, Textarea, InfoBanner, SectionPanel) and migrated all 12+ feature areas. Phase 3 extracts three new primitives that converge in StepMode.tsx and migrates three Tier 1 sm-cluster files that Phase 2 left out of scope. StepMode consumer migration (35+ swap sites) is intentionally deferred to Phase 4 — the primitives ship now so Phase 4 is purely mechanical consumer work.

## Non-goals

- StepMode.tsx consumer migration (Phase 4)
- CheckboxButton extraction (confirmed divergent: 3 sizes, inconsistent border widths, different unselected backgrounds)
- InfoBanner tone variants (all 21 existing uses are neutral; destructive blocks are intentional separate inline divs)
- Native `<select>` migration (already 100% shadcn Select — nothing to do)
- StatCard, NotificationPill, EntityListRow, InstallLogPanel (not found as repeated patterns in codebase)
- Tokens (`globals.css` locked)
- Consumer-level tests

## Standing decisions (carried from Phase 1/2)

1. Work directly on `main`. No PR, no worktree.
2. Conventional Commits with trailer `Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>`.
3. **Zero behavior change.** Primitives produce identical DOM semantics + handler signatures.
4. TDD cycle per primitive: `*.test.tsx` sibling with vitest + `@testing-library/react`.
5. Verification gate per commit: `cd apps/web && npx vitest run`. Coordinator runs `npx next build` after primitive batch and after consumer batch. Subagents skip `npx next build`.
6. Tokens locked — do NOT edit `apps/web/src/app/globals.css`.
7. Opportunistic token cleanup in touched files only.
8. Next.js 16 caveat: Phase 3 candidates are pure client components — no Next.js surface touched.

## New primitives

### A. `<SettingRow>` — `apps/web/src/components/ui/setting-row.tsx`

Layout wrapper for a horizontal label-left / control-right row. Used 15× in StepMode (ToggleSwitch pattern) and likely more across settings screens.

```tsx
import * as React from "react"
import { cn } from "@/lib/utils"

interface SettingRowProps {
  label: React.ReactNode
  description?: React.ReactNode
  className?: string
  children: React.ReactNode
}

function SettingRow({ label, description, className, children }: SettingRowProps) {
  return (
    <div
      data-slot="setting-row"
      className={cn("flex items-center justify-between gap-3", className)}
    >
      <div className="flex flex-col min-w-0">
        <p className="text-xs font-medium text-prose">{label}</p>
        {description && (
          <p data-slot="setting-row-description" className="text-xs text-subtle">
            {description}
          </p>
        )}
      </div>
      <div className="shrink-0">{children}</div>
    </div>
  )
}

export { SettingRow }
export type { SettingRowProps }
```

Tests (`setting-row.test.tsx`):
- Renders label text.
- Description renders when provided, omitted when not.
- Children render in the control slot.
- `className` merges via `cn`.
- `data-slot="setting-row"` present on root.

### B. `<SegmentedControl>` — `apps/web/src/components/ui/segmented-control.tsx`

Button-group selector in two layout variants. Used 5–9× in StepMode (5 flat button-groups + some card-grid selectors).

```tsx
import * as React from "react"
import { cn } from "@/lib/utils"

interface SegmentedOption<T extends string> {
  value: T
  label: string
  description?: string
}

interface SegmentedControlProps<T extends string> {
  options: SegmentedOption<T>[]
  value: T
  onChange: (value: T) => void
  variant?: "flat" | "card"
  className?: string
}

// Tailwind cannot purge dynamic class names — use a static lookup for grid-cols.
const GRID_COLS: Record<number, string> = {
  2: "grid-cols-2",
  3: "grid-cols-3",
  4: "grid-cols-4",
}

function SegmentedControl<T extends string>({
  options,
  value,
  onChange,
  variant = "flat",
  className,
}: SegmentedControlProps<T>) {
  if (variant === "card") {
    const colClass = GRID_COLS[options.length] ?? "grid-cols-2"
    return (
      <div
        data-slot="segmented-control"
        data-variant="card"
        className={cn("grid gap-2", colClass, className)}
      >
        {options.map((opt) => (
          <button
            key={opt.value}
            type="button"
            onClick={() => onChange(opt.value)}
            className={cn(
              "flex flex-col items-start rounded-lg border p-3 text-left transition-all",
              opt.value === value
                ? "border-brand bg-surface ring-1 ring-brand"
                : "border-border bg-surface hover:border-border/60",
            )}
            aria-pressed={opt.value === value}
          >
            <span className="text-xs font-medium text-prose">{opt.label}</span>
            {opt.description && (
              <span className="text-xs text-subtle">{opt.description}</span>
            )}
          </button>
        ))}
      </div>
    )
  }

  return (
    <div
      data-slot="segmented-control"
      data-variant="flat"
      className={cn("flex gap-2", className)}
    >
      {options.map((opt) => (
        <button
          key={opt.value}
          type="button"
          onClick={() => onChange(opt.value)}
          className={cn(
            "flex-1 rounded px-2 py-1.5 text-xs transition-colors",
            opt.value === value
              ? "bg-brand text-brand-foreground font-medium"
              : "bg-surface text-subtle hover:bg-surface/80",
          )}
          aria-pressed={opt.value === value}
        >
          {opt.label}
        </button>
      ))}
    </div>
  )
}

export { SegmentedControl }
export type { SegmentedControlProps, SegmentedOption }
```

Tests (`segmented-control.test.tsx`):
- Flat variant renders all option labels.
- Active option has `bg-brand` class; inactive has `bg-surface`.
- Card variant renders labels + descriptions.
- Card active option has `border-brand ring-brand`; inactive has `border-border`.
- `onChange` fires with correct value on click.
- `aria-pressed` reflects active state.
- `className` merges.

### C. `<SliderRow>` — `apps/web/src/components/ui/slider-row.tsx`

Label + native range input + formatted value display in a single horizontal row. Used 11× in StepMode with varying format suffixes (none, `%`, `px`).

```tsx
import * as React from "react"
import { cn } from "@/lib/utils"

interface SliderRowProps {
  label: string
  min: number
  max: number
  value: number
  onChange: (value: number) => void
  step?: number
  format?: (value: number) => string
  className?: string
}

function SliderRow({
  label,
  min,
  max,
  value,
  onChange,
  step,
  format,
  className,
}: SliderRowProps) {
  const display = format ? format(value) : String(value)
  return (
    <div
      data-slot="slider-row"
      className={cn("flex items-center gap-3", className)}
    >
      <span className="text-xs font-medium text-subtle shrink-0">{label}</span>
      <input
        type="range"
        min={min}
        max={max}
        step={step}
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="flex-1 accent-brand"
      />
      <span className="text-xs text-subtle text-right shrink-0 tabular-nums">
        {display}
      </span>
    </div>
  )
}

export { SliderRow }
export type { SliderRowProps }
```

Tests (`slider-row.test.tsx`):
- Renders label text.
- Range input has correct `min`, `max`, `value`.
- Default format: display span shows raw `String(value)`.
- Custom `format`: display span shows formatted value (e.g. `"50%"`).
- `onChange` fires with numeric value on input change.
- `step` forwarded to input when provided.
- `className` merges.

## Consumer migrations

### D. Tier 1 sm-cluster — `refactor(pipeline,channels): adopt SectionPanel size="sm"`

Three files Phase 2 left out of scope. All have identical `rounded-lg border border-border bg-card p-3` wrappers with uppercase eyebrow titles matching `SectionPanel size="sm"` exactly.

| File | Eyebrow → `title` prop | Notes |
|------|------------------------|-------|
| `apps/web/src/components/pipeline/ContextPanel.tsx` | `"Info"` | `space-y-1` children absorbed by SectionPanel `space-y-3` |
| `apps/web/src/components/pipeline/DownloadInfoCard.tsx` | `"Download"` | Children: thumbnail + progress bar; no handler changes |
| `apps/web/src/components/channels/ChannelInfoCard.tsx` | `"Channel"` | Single datum; simplest swap |

The one non-eyebrow card (StepMode line 705, horizontal-flex thumbnail card) stays inline — single instance, unique layout, revisited in Phase 4.

One Sonnet subagent handles all three files in a single commit.

## Session execution

### Commit sequence

| # | Commit | Owner | Gate |
|---|--------|-------|------|
| A | `feat(ui): add SettingRow primitive` | Coordinator | `npx vitest run` |
| B | `feat(ui): add SegmentedControl primitive` | Coordinator | `npx vitest run` |
| C | `feat(ui): add SliderRow primitive` | Coordinator | `npx vitest run` + `npx next build` |
| D | `refactor(pipeline,channels): adopt SectionPanel size="sm"` | Sonnet subagent | `npx vitest run` (subagent); coordinator runs `npx next build` after |
| E | `chore(ui): Phase 3 verification pass` (skip if no stragglers) | Coordinator | grep + `npx next build` |

### Verification greps (Commit E)

From `apps/web/`:
```bash
# Expect 0 hits outside section-panel.tsx and the 3 migrated files:
grep -rn 'rounded-lg border border-border bg-card p-3' src/ --include='*.tsx'

# Expect 0 residual raw color tokens in touched files:
grep -rn 'bg-zinc-\|text-zinc-\|border-zinc-' src/components/pipeline/ContextPanel.tsx \
  src/components/pipeline/DownloadInfoCard.tsx \
  src/components/channels/ChannelInfoCard.tsx
```

### Subagent prompt (D — Sonnet)

```
You are migrating three small components to the already-shipped <SectionPanel size="sm"> primitive.
No design questions — purely mechanical.

Plan file: /Users/joaogabriel/.claude/plans/<phase3-plan>.md

Your files:
- apps/web/src/components/pipeline/ContextPanel.tsx
- apps/web/src/components/pipeline/DownloadInfoCard.tsx
- apps/web/src/components/channels/ChannelInfoCard.tsx

Primitive available: <SectionPanel size="sm" title="…"> from @/components/ui/section-panel

Rules:
1. Zero behavior change. Preserve all handlers, props, conditional renders.
2. Do NOT edit globals.css.
3. Do NOT run npx next build (coordinator handles it).
4. Opportunistic token cleanup: map bg-zinc-*/text-zinc-*/border-zinc-* to semantic tokens.

Migration checklist per file:
- Find the <div className="rounded-lg border border-border bg-card p-3 …"> wrapper.
- Find the inline eyebrow <p className="text-xs font-semibold uppercase …">TITLE</p>.
- Replace wrapper + eyebrow with <SectionPanel size="sm" title="TITLE">.
- Remove the eyebrow <p> (title now in the prop).

Verification gate (from apps/web/):
1. npx vitest run — must pass.
2. grep -n 'rounded-lg border border-border bg-card p-3' your-files → expect 0 hits.

Commit:
git add <your files>
git commit -m "refactor(pipeline,channels): adopt SectionPanel size=\"sm\"

- ContextPanel, DownloadInfoCard, ChannelInfoCard: SectionPanel size=sm adoption
- Removed inline eyebrow pattern (p-3 wrapper + uppercase <p>)

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"

Report: (a) primitives adopted with file counts, (b) inline patterns removed,
(c) token normalizations if any, (d) confirm vitest passed, (e) commit SHA.
```

## Phase 4 seed

Phase 4 ("StepMode simplification") will migrate StepMode.tsx using the three newly shipped primitives plus the already-shipped `<Switch>` from `/components/ui/switch.tsx`:
- 15× ToggleSwitch inline `<button role="switch">` → `<Switch>` inside `<SettingRow>`
- 5–9× flat/card button-groups → `<SegmentedControl variant="flat|card">`
- 11× native `<input type="range">` rows → `<SliderRow format={…}>`

Estimated: one Opus subagent, StepMode.tsx only, split into two passes if context pressure.

## Critical files

**New:**
- `apps/web/src/components/ui/setting-row.tsx` + `setting-row.test.tsx`
- `apps/web/src/components/ui/segmented-control.tsx` + `segmented-control.test.tsx`
- `apps/web/src/components/ui/slider-row.tsx` + `slider-row.test.tsx`

**Modified (consumers):**
- `apps/web/src/components/pipeline/ContextPanel.tsx`
- `apps/web/src/components/pipeline/DownloadInfoCard.tsx`
- `apps/web/src/components/channels/ChannelInfoCard.tsx`

## Existing utilities reused

- `cn` from `@/lib/utils` — use in every new primitive.
- `SectionPanel` from `@/components/ui/section-panel` — consumers import this, no changes.
- `Switch` from `@/components/ui/switch` — already Radix-based; Phase 4 uses it.
- Vitest config at `apps/web/vitest.config.ts` + setup `apps/web/src/test/setup.ts` — unchanged.

## Verification end-to-end

1. Per-commit: `npx vitest run` passes.
2. After primitive commits (A–C): `npx next build` passes.
3. After consumer batch (D): `npx vitest run && npx next build` both pass.
4. After E: greps listed above return zero hits.
5. Manual spot-check: `npm run dev` → visit `/pipeline` (ContextPanel visible in step sidebar), open channel config sheet (ChannelInfoCard), trigger a download (DownloadInfoCard). Confirm no visible regressions.
