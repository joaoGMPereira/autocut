# Design System Phase 4 — StepMode Consumer Migration

**Date:** 2026-04-23
**File:** `apps/web/src/components/pipeline/steps/StepMode.tsx` (2198 lines)
**Depends on:** Phase 3 primitives (`SettingRow`, `SegmentedControl`, `SliderRow`, `Switch`)

## Context

Phases 1–3 of the AutoCut design system refactor landed on `main`. Phase 3 shipped three new primitives plus the existing Radix `Switch`:

- `SettingRow` — `apps/web/src/components/ui/setting-row.tsx`
- `SegmentedControl` (variants `flat` + `card`) — `apps/web/src/components/ui/segmented-control.tsx`
- `SliderRow` — `apps/web/src/components/ui/slider-row.tsx`
- `Switch` — `apps/web/src/components/ui/switch.tsx`

Phase 4 is a pure consumer migration: replace hand-rolled control patterns in `StepMode.tsx` with Phase 3 primitives. Zero new primitives. Zero cross-file changes. Zero behavior change.

## Scope

32 in-scope migration sites across four primitives. 13 sites explicitly preserved inline (semantic mismatch with primitive shapes).

Line numbers below are snapshots of `main` at commit `a8af612` (before Phase 4 starts). Line numbers will shift between commits as earlier migrations land; the implementation plan must key on surrounding context (label text, handler name, JSX structure) rather than raw line numbers.

### Migrations (32 total)

**Switch + SettingRow (15 sites):**

10 sites currently shaped as `<div class="flex items-center justify-between">…<button role="switch">` — migrate to `<SettingRow label={…} description={…}><Switch/></SettingRow>`:

| Line | Context |
|------|---------|
| 768  | Reuse Existing Clips (inside InfoBanner) |
| 958  | Force Re-analyze |
| 982  | Skip Transcription |
| 1469 | Watermark (Logo) |
| 1582 | Watermark pulsante |
| 1606 | Intro animado (label + description) |
| 1630 | Outro (tela final) |
| 1993 | Schedule Sequentially |
| 2017 | Auto Upload |
| 2041 | Dry Run |

5 sites are `<SectionPanel actions={<button role="switch">…</button>}>` — migrate the `actions` prop to a bare `<Switch>` (no SettingRow, the panel already owns the label):

| Line | Context |
|------|---------|
| 1086 | Anti-Duplicate Protection |
| 1216 | Overlays de Texto |
| 1341 | Overlay de Vídeo |
| 1655 | Música de Fundo |
| 1757 | Legendas |

**SliderRow (11 sites):**

Each `<input type="range">` row becomes `<SliderRow label min max value onChange step format>` with a format function matching the current trailing span:

| Line | Unit | Format |
|------|------|--------|
| 1168 | bare int | `v => String(v)` |
| 1195 | `%`      | `v => \`${v}%\`` |
| 1389 | `%`      | `v => \`${v}%\`` |
| 1430 | `s`      | `v => \`${v}s\`` |
| 1556 | `%`      | `v => \`${v}%\`` |
| 1570 | `%`      | `v => \`${v}%\`` |
| 1738 | `%`      | `v => \`${v}%\`` |
| 1838 | `px`     | `v => \`${v}px\`` |
| 1899 | bare int | `v => String(v)` |
| 1937 | bare int | `v => String(v)` |
| 1955 | bare int | `v => String(v)` |

**SegmentedControl flat (5 sites):**

| Line | Context | Options |
|------|---------|---------|
| 922  | Highlight Threshold | 3 (Liberal/Balanced/Selective, with percent annotation in label) |
| 1409 | Overlay appearances | 5 (`1×`…`5×`) |
| 1678 | Música mode | 3 (random/library/custom) |
| 1780 | Captions preset | 3 (simple/bold/word_by_word) |
| 1974 | Upload privacy | 3 (private/unlisted/public) — pill → rect drift accepted |

**SegmentedControl card (1 site):**

| Line | Context |
|------|---------|
| 1109 | Anti-dup mode (subtle/aggressive) — labels pre-capitalized in `options` array to preserve the `capitalize` CSS effect |

### Preserved inline (13 sites)

| Line | Reason |
|------|--------|
| 705  | thumbnail card, horizontal flex, no button-group semantics |
| 736  | preset carousel — overflow-x-auto, dynamic delete affordance |
| 797  | mode-cards grid — bullet lists + checkmark badges exceed SegmentedControl card shape |
| 861  | Target Clip Duration — `flex-wrap` + 1–3 dynamic buttons w/ fallback |
| 898  | Minimum Clip Duration — `flex-wrap`, 6 options |
| 1013 | Segment Duration (longform) — `flex-wrap` + dynamic 1–3 |
| 1053 | Min Part Duration — `flex-wrap`, 7 options |
| 1241 | Overlay #N header — label + button group, not a control row |
| 1444 | Chroma Key — `flex-wrap`, 5 options |
| 1817 | Bold/Italic/Uppercase — multi-select (SegmentedControl is single-select) |
| 2069 | preset save — Input + Button row |
| 2100 | Preview "Aguardando download" header — h3 + span, not SettingRow shape |
| 2129 | Preview "Ready" header — h3 + Button, not SettingRow shape |

## Commit Plan

Three conventional commits, each on `main`, each independently verifiable:

1. `refactor(ui): adopt SettingRow + Switch in StepMode`
   — 15 Switch sites (10 SettingRow-wrapped + 5 bare in `SectionPanel actions`).
2. `refactor(ui): adopt SegmentedControl in StepMode`
   — 6 sites (5 flat + 1 card).
3. `refactor(ui): adopt SliderRow in StepMode`
   — 11 sites.

Each commit carries trailer:

```
Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
```

## Verification Gate

Per commit, before the commit is created:

```sh
cd apps/web && npx vitest run
```

Coordinator runs `npx next build` externally after each commit. If vitest fails, fix in-place and retry — never bypass with `--no-verify`. If tests pass locally but `next build` surfaces a type error, fix and land as a new follow-up commit (not `--amend`).

## Visual Drift (Accepted)

The primitives are the authoritative source of truth after Phase 3. The following drifts are intended side effects of the migration:

- **SettingRow labels**: current `<p class="text-xs text-prose">` → primitive `<p class="text-xs font-medium text-prose">`. Description shifts one shade: `text-caption` → `text-subtle`.
- **SliderRow widths**: current label widths `w-16` / `w-20 flex-shrink-0` → primitive `shrink-0` only. Current value spans `w-4` / `w-6` / `w-8` / `w-10` → primitive `min-w-[3ch] tabular-nums`. Column alignment across sibling sliders becomes content-width driven.
- **SegmentedControl flat / Upload privacy (L1974)**: current `rounded-full px-3` → primitive `rounded px-2`.

No visual regressions are expected for correctness — only aesthetic tightening that aligns StepMode with the rest of the system.

## Behavior Preservation Contract

- All state setters, handlers, and conditional branches (`{selectedMode === 'ai'}`, `{antiDupEnabled && …}`, `{!item.apply_full && …}`, etc.) remain exactly as-is.
- All `data-testid` attributes remain on their original elements where still meaningful. No new test IDs added.
- `SectionPanel.actions` accepts `ReactNode`; substituting `<Switch>` for the hand-rolled button keeps the slot contract.
- Anti-dup card (L1109) labels `subtle`/`aggressive` are transformed to `Subtle`/`Aggressive` in the `options` array to preserve the current `capitalize` CSS effect (SegmentedControl renders plain text).
- Console logs (e.g., `console.log('[StepMode] skip_regenerate toggled:', …)`) remain attached to the appropriate handlers.

## Non-goals

- No edits to `apps/web/src/app/globals.css`.
- No new primitives, no Phase 3 revisions.
- No cross-file changes.
- No zinc-token cleanup — `StepMode.tsx` contains zero `zinc-*` classes.
- No PR, no worktree. Direct to `main` per standing decision.

## References

- Phase 3 design spec: `docs/superpowers/specs/2026-04-22-design-system-phase3-design.md`
- Parent refactor spec: `docs/superpowers/specs/2026-04-22-design-system-refactor-design.md`
