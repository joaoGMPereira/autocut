# Design System Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend shadcn primitives (`Badge`, `Input`, `Tabs`) and add new ones (`PageHeader`, `FormField`); then migrate every feature screen to use them. Eliminate inline markup duplication, normalize lingering raw colors to semantic tokens (opportunistically on touched files).

**Architecture:** Six sequential primitives commits (Tasks 1–6) by a single agent; one parallel-dispatch task (Task 7) that fires 12 area-migration subagents at once; one final verification task (Task 8). All work lands directly on `main` — no PRs.

**Tech Stack:** TypeScript 5.8 / React 19, Next.js 16, Tailwind v4, shadcn/ui ("new-york"), `class-variance-authority`, `radix-ui`, vitest 4 + `@testing-library/react`.

**Spec:** [`docs/superpowers/specs/2026-04-22-design-system-refactor-design.md`](../specs/2026-04-22-design-system-refactor-design.md)

---

## Critical context before starting

- **Working directory** for all `npm`/`npx` commands: `apps/web/`. Git commands run from repo root `/Users/joaogabriel/Projects/YoutubeProjects/AutoCut/`.
- **Tokens are locked.** Do NOT touch `apps/web/src/app/globals.css`. All tokens already defined: `bg-brand`, `bg-success`, `bg-warning`, `bg-info`, `bg-destructive`, `bg-surface`, `bg-card`, `border-border`, `text-heading`, `text-subtle`, `text-foreground`, etc.
- **DS reference bundle:** `/Users/joaogabriel/Downloads/autocut-design-system/project/`. Key files for primitives: `preview/badges.html`, `preview/inputs.html`, `preview/buttons.html`, `ui_kits/desktop/Primitives.jsx`, `colors_and_type.css`, `README.md`.
- **Existing test infra:** vitest config at `apps/web/vitest.config.ts`, jsdom environment, setup file `src/test/setup.ts` polyfills `ResizeObserver` (used by radix Slider). Existing test reference: `apps/web/src/components/ui/switch.test.tsx`.
- **Build command:** `npx next build` from `apps/web/`. **Test command:** `npx vitest run` from `apps/web/`.
- **Existing shadcn `<Input>` consumers** (16 files — must visually verify after Task 2): `app/channels/page.tsx`, `settings/AppSettingsSection.tsx`, `settings/OAuthProfilesSection.tsx`, `channels/CommentSyncSheet.tsx`, `channels/ChannelConfigSheet.tsx`, `pipeline/steps/StepReviewMetadata.tsx`, `pipeline/steps/StepThumbnailConfig.tsx`, `post-opt/LogoTab.tsx`, `post-opt/TransitionsTab.tsx`, `post-opt/ColorInputField.tsx`, `post-opt/TextStyleEditorPanel.tsx`, `post-opt/SpeedTab.tsx`, `post-opt/AntiDupTab.tsx`, `post-opt/TextOverlayTab.tsx`, `setup/CustomPathDialog.tsx`, `history/RunFilters.tsx`.
- **`ChannelAvatar`** is exported from `channels/ChannelCard.tsx` and consumed by `queue/QueueItemCard.tsx`. The channels subagent owns the export; the queue subagent must NOT change `ChannelAvatar`'s shape.
- **Pipeline files OUT of scope** (specialized, single-use, do not touch in this refactor): `pipeline/StepRail.tsx`, `pipeline/LogStream.tsx`, `pipeline/GateActionBar.tsx`, `pipeline/ContextPanel.tsx`, `pipeline/PipelineShell.tsx`, `pipeline/ActiveStepArea.tsx`, `pipeline/BackConfirmDialog.tsx`.
- **`cn` helper:** `import { cn } from '@/lib/utils'` — twMerge + clsx. Always use it for class composition.
- **Commit messages** use Conventional Commits with the trailer `Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>`.

---

## File structure

### Created files
| Path | Responsibility |
|---|---|
| `apps/web/src/components/ui/page-header.tsx` | `<PageHeader>` primitive — title (`<h1>` 32px display) + optional description + actions slot |
| `apps/web/src/components/ui/form-field.tsx` | `<FormField>` wrapper — Label + control slot + optional description/error, ARIA wired |
| `apps/web/src/components/ui/badge.test.tsx` | Render tests for new soft-fill variants |
| `apps/web/src/components/ui/input.test.tsx` | Render tests for new default + invalid + disabled |
| `apps/web/src/components/ui/page-header.test.tsx` | Render tests for title/description/actions slots |
| `apps/web/src/components/ui/form-field.test.tsx` | Render tests for ARIA wiring + required + error |
| `apps/web/src/components/ui/tabs.test.tsx` | Render tests for `count` slot on `TabsTrigger` |

### Modified files (primitives)
| Path | Change |
|---|---|
| `apps/web/src/components/ui/badge.tsx` | Add `success`, `warning`, `info`, `brand` soft-fill variants |
| `apps/web/src/components/ui/input.tsx` | Rewrite default styles to AutoCut soft-bg + brand focus ring |
| `apps/web/src/components/ui/tabs.tsx` | Extend `TabsTrigger` with `count?: number` prop + count pill rendering |

### Modified files (consumer migration — owned by Task 7 subagents)
See Task 7 area table for the per-subagent file list (40+ files across shorts, queue, channels, settings, dashboard, history, dispatcher, post-opt, thumbnail, setup, pipeline).

---

## Task 1: Extend Badge with soft-fill variants

**Files:**
- Modify: `apps/web/src/components/ui/badge.tsx`
- Create: `apps/web/src/components/ui/badge.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `apps/web/src/components/ui/badge.test.tsx`:

```tsx
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Badge } from './badge';

describe('Badge', () => {
  it('renders default variant unchanged', () => {
    render(<Badge>default</Badge>);
    const badge = screen.getByText('default');
    expect(badge).toHaveAttribute('data-variant', 'default');
    expect(badge.className).toContain('bg-primary');
  });

  it('renders destructive variant unchanged (solid red)', () => {
    render(<Badge variant="destructive">err</Badge>);
    const badge = screen.getByText('err');
    expect(badge).toHaveAttribute('data-variant', 'destructive');
    expect(badge.className).toContain('bg-destructive');
  });

  describe('soft-fill variants', () => {
    it.each([
      ['success', ['bg-success/10', 'border-success/20', 'text-success']],
      ['warning', ['bg-warning/15', 'border-warning/40', 'text-warning']],
      ['info', ['bg-info/10', 'border-info/20', 'text-info']],
      ['brand', ['bg-brand/10', 'border-brand/40', 'text-brand']],
    ] as const)('renders %s with soft-fill classes', (variant, classes) => {
      render(<Badge variant={variant}>label</Badge>);
      const badge = screen.getByText('label');
      expect(badge).toHaveAttribute('data-variant', variant);
      classes.forEach((c) => expect(badge.className).toContain(c));
    });
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd apps/web && npx vitest run src/components/ui/badge.test.tsx
```
Expected: FAIL — soft-fill variants not defined; type error on `variant="success"`.

- [ ] **Step 3: Add the four variants to badge.tsx**

Open `apps/web/src/components/ui/badge.tsx`. Inside the `cva` `variants.variant` object (after the `link` entry, before the closing brace), add:

```tsx
        success:
          "border-success/20 bg-success/10 text-success [a&]:hover:bg-success/15",
        warning:
          "border-warning/40 bg-warning/15 text-warning [a&]:hover:bg-warning/20",
        info:
          "border-info/20 bg-info/10 text-info [a&]:hover:bg-info/15",
        brand:
          "border-brand/40 bg-brand/10 text-brand [a&]:hover:bg-brand/15",
```

The full `variants.variant` block now contains 10 entries: `default`, `secondary`, `destructive`, `outline`, `ghost`, `link`, `success`, `warning`, `info`, `brand`.

- [ ] **Step 4: Run test to verify it passes**

```bash
cd apps/web && npx vitest run src/components/ui/badge.test.tsx
```
Expected: PASS — all 6 tests green (`default`, `destructive`, plus 4 soft-fill variants).

- [ ] **Step 5: Build check**

```bash
cd apps/web && npx next build
```
Expected: success. No type errors.

- [ ] **Step 6: Commit**

```bash
git add apps/web/src/components/ui/badge.tsx apps/web/src/components/ui/badge.test.tsx
git commit -m "$(cat <<'EOF'
feat(ui): extend Badge with success/warning/info/brand soft-fill variants

Adds four DS soft-fill variants (bg-token/10 + border-token/20 + text-token).
Existing variants (default/secondary/destructive/outline/ghost/link) untouched
— destructive intentionally stays solid for alert/error emphasis.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Rewrite Input with surface-bg + brand focus ring

**Files:**
- Modify: `apps/web/src/components/ui/input.tsx`
- Create: `apps/web/src/components/ui/input.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `apps/web/src/components/ui/input.test.tsx`:

```tsx
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Input } from './input';

describe('Input', () => {
  it('renders with surface bg and border classes by default', () => {
    render(<Input aria-label="x" />);
    const input = screen.getByLabelText('x');
    expect(input.className).toContain('bg-surface');
    expect(input.className).toContain('border-border');
    expect(input.className).toContain('rounded-lg');
    expect(input.className).toContain('h-9');
  });

  it('applies brand focus ring class', () => {
    render(<Input aria-label="x" />);
    const input = screen.getByLabelText('x');
    expect(input.className).toContain('focus-visible:border-brand/60');
    expect(input.className).toContain('focus-visible:ring-brand/30');
  });

  it('applies destructive ring when aria-invalid', () => {
    render(<Input aria-label="x" aria-invalid />);
    const input = screen.getByLabelText('x');
    expect(input.className).toContain('aria-invalid:border-destructive');
    expect(input.className).toContain('aria-invalid:ring-destructive/30');
  });

  it('applies disabled styles', () => {
    render(<Input aria-label="x" disabled />);
    const input = screen.getByLabelText('x');
    expect(input.className).toContain('disabled:opacity-50');
    expect(input).toBeDisabled();
  });

  it('passes className through', () => {
    render(<Input aria-label="x" className="font-mono custom-extra" />);
    const input = screen.getByLabelText('x');
    expect(input.className).toContain('font-mono');
    expect(input.className).toContain('custom-extra');
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd apps/web && npx vitest run src/components/ui/input.test.tsx
```
Expected: FAIL — current `<Input>` uses `bg-transparent border-input rounded-md`, not `bg-surface border-border rounded-lg`.

- [ ] **Step 3: Rewrite input.tsx**

Replace the entire contents of `apps/web/src/components/ui/input.tsx`:

```tsx
import * as React from "react"

import { cn } from "@/lib/utils"

function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <input
      type={type}
      data-slot="input"
      className={cn(
        "h-9 w-full min-w-0 rounded-lg bg-surface border border-border px-3 py-2 text-sm text-foreground placeholder:text-subtle",
        "shadow-xs transition-[color,box-shadow,border-color] outline-none",
        "selection:bg-primary selection:text-primary-foreground",
        "file:inline-flex file:h-7 file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground",
        "disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50",
        "focus-visible:border-brand/60 focus-visible:ring-brand/30 focus-visible:ring-[3px]",
        "aria-invalid:border-destructive aria-invalid:ring-destructive/30",
        className,
      )}
      {...props}
    />
  )
}

export { Input }
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd apps/web && npx vitest run src/components/ui/input.test.tsx
```
Expected: PASS — all 5 tests green.

- [ ] **Step 5: Run full vitest suite (regression check)**

```bash
cd apps/web && npx vitest run
```
Expected: PASS — `switch.test.tsx`, `badge.test.tsx`, `input.test.tsx` all green.

- [ ] **Step 6: Build check**

```bash
cd apps/web && npx next build
```
Expected: success.

- [ ] **Step 7: Visual spot-check existing consumers**

Start dev server in a separate terminal:
```bash
cd apps/web && npm run dev
```

Open http://localhost:3201 and visually verify the following routes still render acceptably (inputs should now show surface bg + lime focus rings):
- `/channels` → `AddChannelForm` (top of page)
- `/settings` tab "App" → `AppSettingsSection`
- `/settings` tab "OAuth" → `OAuthProfilesSection`
- `/settings` tab "Channels" → opens `ChannelConfigSheet` / `CommentSyncSheet` from row actions
- `/post-opt` → `LogoTab`, `TransitionsTab`, `SpeedTab`, `AntiDupTab`, `TextOverlayTab` (each has `<Input>` consumers)
- `/setup` → `ToolRow` "Set custom path" opens `CustomPathDialog`
- `/history` → `RunFilters` search input
- `/pipeline` step "Review Metadata" → `StepReviewMetadata` inputs
- `/pipeline` step "Thumbnail Config" → `StepThumbnailConfig` inputs

**Acceptance:** every input visually fits on its surface (no white-bg leakage, no ring-color clash). If any consumer looks broken, stop and discuss before proceeding — do NOT add a `variant="bare"` escape hatch preemptively.

Stop the dev server: `Ctrl-C` in that terminal.

- [ ] **Step 8: Commit**

```bash
git add apps/web/src/components/ui/input.tsx apps/web/src/components/ui/input.test.tsx
git commit -m "$(cat <<'EOF'
feat(ui): rewrite Input with surface-bg + brand focus ring

Replaces the shadcn default (bg-transparent + ring-ring/50) with the
AutoCut surface pattern (bg-surface + border-border + rounded-lg) and a
brand-tinted 3px focus ring. Aligns the primitive with the inline
pattern used everywhere else in the app — single source of truth.

Existing consumers visually verified: channels, settings tabs, post-opt
tabs, setup, history, and pipeline review/thumbnail steps.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Add PageHeader primitive

**Files:**
- Create: `apps/web/src/components/ui/page-header.tsx`
- Create: `apps/web/src/components/ui/page-header.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `apps/web/src/components/ui/page-header.test.tsx`:

```tsx
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { PageHeader } from './page-header';

describe('PageHeader', () => {
  it('renders title as an h1 with display font classes', () => {
    render(<PageHeader title="Settings" />);
    const heading = screen.getByRole('heading', { level: 1 });
    expect(heading).toHaveTextContent('Settings');
    expect(heading.className).toContain('font-display');
    expect(heading.className).toContain('text-[32px]');
    expect(heading.className).toContain('text-heading');
  });

  it('omits description when not provided', () => {
    render(<PageHeader title="X" />);
    expect(document.querySelector('p[data-slot="page-header-description"]')).toBeNull();
  });

  it('renders description when provided', () => {
    render(<PageHeader title="X" description="Manage things" />);
    expect(screen.getByText('Manage things')).toBeInTheDocument();
  });

  it('omits actions slot when not provided', () => {
    render(<PageHeader title="X" />);
    expect(document.querySelector('[data-slot="page-header-actions"]')).toBeNull();
  });

  it('renders actions slot when provided', () => {
    render(<PageHeader title="X" actions={<button>Add</button>} />);
    expect(screen.getByRole('button', { name: 'Add' })).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd apps/web && npx vitest run src/components/ui/page-header.test.tsx
```
Expected: FAIL — module `./page-header` does not exist.

- [ ] **Step 3: Create the PageHeader component**

Create `apps/web/src/components/ui/page-header.tsx`:

```tsx
import * as React from "react"

import { cn } from "@/lib/utils"

interface PageHeaderProps {
  title: React.ReactNode
  description?: React.ReactNode
  actions?: React.ReactNode
  className?: string
}

function PageHeader({ title, description, actions, className }: PageHeaderProps) {
  return (
    <div
      data-slot="page-header"
      className={cn("flex items-start justify-between gap-4", className)}
    >
      <div className="flex flex-col">
        <h1 className="font-display text-[32px] font-bold tracking-[-0.02em] text-heading leading-tight">
          {title}
        </h1>
        {description && (
          <p
            data-slot="page-header-description"
            className="text-sm text-subtle mt-1"
          >
            {description}
          </p>
        )}
      </div>
      {actions && (
        <div data-slot="page-header-actions" className="flex items-center gap-2 shrink-0">
          {actions}
        </div>
      )}
    </div>
  )
}

export { PageHeader }
export type { PageHeaderProps }
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd apps/web && npx vitest run src/components/ui/page-header.test.tsx
```
Expected: PASS — all 5 tests green.

- [ ] **Step 5: Build check**

```bash
cd apps/web && npx next build
```
Expected: success.

- [ ] **Step 6: Commit**

```bash
git add apps/web/src/components/ui/page-header.tsx apps/web/src/components/ui/page-header.test.tsx
git commit -m "$(cat <<'EOF'
feat(ui): add PageHeader primitive

Single-h1 page header with optional description and actions slot.
Codifies the font-display 32px tight-tracked title pattern that was
inlined across 6 page files.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Add FormField primitive

**Files:**
- Create: `apps/web/src/components/ui/form-field.tsx`
- Create: `apps/web/src/components/ui/form-field.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `apps/web/src/components/ui/form-field.test.tsx`:

```tsx
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { FormField } from './form-field';
import { Input } from './input';

describe('FormField', () => {
  it('wires label htmlFor to child input id (auto-generated)', () => {
    render(
      <FormField label="Name">
        <Input data-testid="i" />
      </FormField>,
    );
    const label = screen.getByText('Name').closest('label') as HTMLLabelElement;
    const input = screen.getByTestId('i');
    expect(label.htmlFor).toBeTruthy();
    expect(input.id).toBe(label.htmlFor);
  });

  it('respects child-supplied id', () => {
    render(
      <FormField label="Name">
        <Input id="my-id" data-testid="i" />
      </FormField>,
    );
    const label = screen.getByText('Name').closest('label') as HTMLLabelElement;
    expect(label.htmlFor).toBe('my-id');
    expect((screen.getByTestId('i') as HTMLInputElement).id).toBe('my-id');
  });

  it('omits required marker by default', () => {
    render(
      <FormField label="Name">
        <Input data-testid="i" />
      </FormField>,
    );
    expect(screen.queryByText('*')).not.toBeInTheDocument();
  });

  it('renders required marker when required', () => {
    render(
      <FormField label="Name" required>
        <Input data-testid="i" />
      </FormField>,
    );
    expect(screen.getByText('*')).toBeInTheDocument();
  });

  it('wires aria-describedby to description', () => {
    render(
      <FormField label="Name" description="hint text">
        <Input data-testid="i" />
      </FormField>,
    );
    const input = screen.getByTestId('i');
    const describedBy = input.getAttribute('aria-describedby');
    expect(describedBy).toBeTruthy();
    const desc = screen.getByText('hint text');
    expect(desc.id).toBe(describedBy);
  });

  it('error replaces description and marks aria-invalid', () => {
    render(
      <FormField label="Name" description="hint" error="bad value">
        <Input data-testid="i" />
      </FormField>,
    );
    expect(screen.queryByText('hint')).not.toBeInTheDocument();
    expect(screen.getByText('bad value')).toBeInTheDocument();
    const input = screen.getByTestId('i');
    expect(input).toHaveAttribute('aria-invalid', 'true');
    const describedBy = input.getAttribute('aria-describedby');
    expect(describedBy).toBeTruthy();
    expect(screen.getByText('bad value').id).toBe(describedBy);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd apps/web && npx vitest run src/components/ui/form-field.test.tsx
```
Expected: FAIL — module `./form-field` does not exist.

- [ ] **Step 3: Create the FormField component**

Create `apps/web/src/components/ui/form-field.tsx`:

```tsx
"use client"

import * as React from "react"

import { Label } from "@/components/ui/label"
import { cn } from "@/lib/utils"

interface FormFieldProps {
  label: React.ReactNode
  description?: React.ReactNode
  error?: React.ReactNode
  required?: boolean
  htmlFor?: string
  className?: string
  children: React.ReactElement<{
    id?: string
    "aria-describedby"?: string
    "aria-invalid"?: boolean
  }>
}

function FormField({
  label,
  description,
  error,
  required,
  htmlFor,
  className,
  children,
}: FormFieldProps) {
  const generatedId = React.useId()
  const id = htmlFor ?? children.props.id ?? generatedId

  const showDescription = !error && description != null
  const descriptionId = showDescription ? `${id}-description` : undefined
  const errorId = error != null ? `${id}-error` : undefined
  const ariaDescribedBy =
    [descriptionId, errorId].filter(Boolean).join(" ") || undefined

  const child = React.cloneElement(children, {
    id,
    "aria-describedby":
      ariaDescribedBy ?? children.props["aria-describedby"],
    "aria-invalid":
      error != null ? true : children.props["aria-invalid"],
  })

  return (
    <div
      data-slot="form-field"
      className={cn("flex flex-col gap-1.5", className)}
    >
      <Label
        htmlFor={id}
        className="text-xs font-medium text-subtle"
      >
        {label}
        {required && <span className="text-destructive ml-0.5">*</span>}
      </Label>
      {child}
      {error != null ? (
        <p id={errorId} className="text-xs text-destructive">
          {error}
        </p>
      ) : description != null ? (
        <p id={descriptionId} className="text-xs text-subtle">
          {description}
        </p>
      ) : null}
    </div>
  )
}

export { FormField }
export type { FormFieldProps }
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd apps/web && npx vitest run src/components/ui/form-field.test.tsx
```
Expected: PASS — all 6 tests green.

- [ ] **Step 5: Build check**

```bash
cd apps/web && npx next build
```
Expected: success.

- [ ] **Step 6: Commit**

```bash
git add apps/web/src/components/ui/form-field.tsx apps/web/src/components/ui/form-field.test.tsx
git commit -m "$(cat <<'EOF'
feat(ui): add FormField primitive

Wraps Label + control + helper/error text with auto-wired htmlFor and
aria-describedby. Standalone <Label> stays exported for inline cases
(checkbox rows, dropdown-on-same-line) where the wrapper does not fit.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Add count slot to TabsTrigger

**Files:**
- Modify: `apps/web/src/components/ui/tabs.tsx`
- Create: `apps/web/src/components/ui/tabs.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `apps/web/src/components/ui/tabs.test.tsx`:

```tsx
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Tabs, TabsList, TabsTrigger, TabsContent } from './tabs';

function Demo({ count, defaultValue = 'a' }: { count?: number; defaultValue?: string }) {
  return (
    <Tabs defaultValue={defaultValue}>
      <TabsList>
        <TabsTrigger value="a" count={count}>Alpha</TabsTrigger>
        <TabsTrigger value="b">Beta</TabsTrigger>
      </TabsList>
      <TabsContent value="a">A</TabsContent>
      <TabsContent value="b">B</TabsContent>
    </Tabs>
  );
}

describe('TabsTrigger count slot', () => {
  it('renders count pill when count > 0', () => {
    render(<Demo count={3} />);
    const trigger = screen.getByRole('tab', { name: /Alpha/ });
    const pill = trigger.querySelector('[data-slot="tabs-trigger-count"]');
    expect(pill).not.toBeNull();
    expect(pill).toHaveTextContent('3');
  });

  it('hides pill when count is 0', () => {
    render(<Demo count={0} />);
    const trigger = screen.getByRole('tab', { name: /Alpha/ });
    expect(trigger.querySelector('[data-slot="tabs-trigger-count"]')).toBeNull();
  });

  it('hides pill when count is undefined', () => {
    render(<Demo />);
    const trigger = screen.getByRole('tab', { name: /Alpha/ });
    expect(trigger.querySelector('[data-slot="tabs-trigger-count"]')).toBeNull();
  });

  it('count pill carries the active-state inversion class', () => {
    render(<Demo count={5} defaultValue="a" />);
    const trigger = screen.getByRole('tab', { name: /Alpha/ });
    const pill = trigger.querySelector('[data-slot="tabs-trigger-count"]')!;
    expect(pill.className).toContain('group-data-[state=active]/tabs-trigger:bg-primary-foreground/20');
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd apps/web && npx vitest run src/components/ui/tabs.test.tsx
```
Expected: FAIL — `count` is not a known prop on `TabsTrigger`; pill not rendered.

- [ ] **Step 3: Modify tabs.tsx**

In `apps/web/src/components/ui/tabs.tsx`:

(a) Replace the entire `function TabsTrigger(...)` block with:

```tsx
type TabsTriggerProps = React.ComponentProps<typeof TabsPrimitive.Trigger> & {
  count?: number
}

function TabsTrigger({
  className,
  count,
  children,
  ...props
}: TabsTriggerProps) {
  return (
    <TabsPrimitive.Trigger
      data-slot="tabs-trigger"
      className={cn(
        "group/tabs-trigger relative inline-flex h-[calc(100%-1px)] flex-1 items-center justify-center gap-1.5 rounded-md border border-transparent px-2 py-1 text-sm font-medium whitespace-nowrap text-foreground/60 transition-all group-data-[orientation=vertical]/tabs:w-full group-data-[orientation=vertical]/tabs:justify-start hover:text-foreground focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:outline-1 focus-visible:outline-ring disabled:pointer-events-none disabled:opacity-50 group-data-[variant=default]/tabs-list:data-[state=active]:shadow-sm group-data-[variant=line]/tabs-list:data-[state=active]:shadow-none dark:text-muted-foreground dark:hover:text-foreground [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
        "group-data-[variant=line]/tabs-list:bg-transparent group-data-[variant=line]/tabs-list:data-[state=active]:bg-transparent dark:group-data-[variant=line]/tabs-list:data-[state=active]:border-transparent dark:group-data-[variant=line]/tabs-list:data-[state=active]:bg-transparent",
        "data-[state=active]:bg-background data-[state=active]:text-foreground dark:data-[state=active]:border-input dark:data-[state=active]:bg-input/30 dark:data-[state=active]:text-foreground",
        "after:absolute after:bg-foreground after:opacity-0 after:transition-opacity group-data-[orientation=horizontal]/tabs:after:inset-x-0 group-data-[orientation=horizontal]/tabs:after:bottom-[-5px] group-data-[orientation=horizontal]/tabs:after:h-0.5 group-data-[orientation=vertical]/tabs:after:inset-y-0 group-data-[orientation=vertical]/tabs:after:-right-1 group-data-[orientation=vertical]/tabs:after:w-0.5 group-data-[variant=line]/tabs-list:data-[state=active]:after:opacity-100",
        className
      )}
      {...props}
    >
      {children}
      {typeof count === "number" && count > 0 && (
        <span
          data-slot="tabs-trigger-count"
          className={cn(
            "ml-1.5 inline-flex min-w-[1.25rem] items-center justify-center rounded-full px-1.5 py-0.5 text-[10px] font-medium",
            "bg-muted text-muted-foreground",
            "group-data-[state=active]/tabs-trigger:bg-primary-foreground/20 group-data-[state=active]/tabs-trigger:text-foreground",
          )}
        >
          {count}
        </span>
      )}
    </TabsPrimitive.Trigger>
  )
}
```

The only changes vs. the original are:
1. New `TabsTriggerProps` type alias adding `count?: number`.
2. `count` extracted from props.
3. `children` extracted explicitly so we can render content + pill side-by-side.
4. `group/tabs-trigger` added to the trigger's classes (enables `group-data-[state=active]/tabs-trigger:` on descendants).
5. Conditional pill `<span>` rendered after `children`.

- [ ] **Step 4: Run test to verify it passes**

```bash
cd apps/web && npx vitest run src/components/ui/tabs.test.tsx
```
Expected: PASS — all 4 tests green.

- [ ] **Step 5: Run full vitest suite**

```bash
cd apps/web && npx vitest run
```
Expected: PASS — `switch`, `badge`, `input`, `page-header`, `form-field`, `tabs` all green.

- [ ] **Step 6: Build check**

```bash
cd apps/web && npx next build
```
Expected: success.

- [ ] **Step 7: Commit**

```bash
git add apps/web/src/components/ui/tabs.tsx apps/web/src/components/ui/tabs.test.tsx
git commit -m "$(cat <<'EOF'
feat(ui): add count slot to TabsTrigger

Optional count?: number prop renders a pill after the label when count > 0.
Active trigger inverts the pill bg via group-data-[state=active] selector.
Hidden when count is 0 or undefined.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Settings-row pattern check (decision gate)

**Files:**
- Read only: `apps/web/src/components/settings/AppSettingsSection.tsx`, `ToolsSection.tsx`, `OAuthProfilesSection.tsx`, `ChannelsSection.tsx`, `ChannelAuthSection.tsx`, `ModelsSection.tsx`

This task is a single-agent decision gate. It does NOT mutate code unless the criterion is met.

- [ ] **Step 1: Audit settings sections for a repeating row pattern**

Read each settings section file end-to-end. Look for a clean repeating row pattern that has all of:
- Same outer container shape (e.g. `flex items-center justify-between border-b py-3`)
- Same left block (icon + title + description)
- Same right block (control: switch/select/input)
- Appears 3+ times across the cluster

Report findings: list each candidate with the file path and a short description.

- [ ] **Step 2: Decide**

If the pattern appears **3+ times with consistent shape**, proceed to Step 3. If not (1–2 instances or shapes vary too much), skip to Step 5 (no commit).

- [ ] **Step 3: Create setting-row.tsx (only if pattern found)**

Create `apps/web/src/components/ui/setting-row.tsx` with the API matching what was found. Example skeleton:

```tsx
import * as React from "react"
import { cn } from "@/lib/utils"

interface SettingRowProps {
  title: React.ReactNode
  description?: React.ReactNode
  icon?: React.ReactNode
  control: React.ReactNode
  className?: string
}

function SettingRow({ title, description, icon, control, className }: SettingRowProps) {
  return (
    <div
      data-slot="setting-row"
      className={cn(
        "flex items-center justify-between gap-4 py-3 border-b border-border last:border-b-0",
        className,
      )}
    >
      <div className="flex items-start gap-3 min-w-0">
        {icon && <div className="text-subtle shrink-0 mt-0.5">{icon}</div>}
        <div className="flex flex-col min-w-0">
          <p className="text-sm font-medium text-foreground">{title}</p>
          {description && (
            <p className="text-xs text-subtle mt-0.5">{description}</p>
          )}
        </div>
      </div>
      <div className="shrink-0">{control}</div>
    </div>
  )
}

export { SettingRow }
```

Adjust the API to match the actual pattern found in Step 1.

Add a render test `setting-row.test.tsx` covering: title rendered, description omitted/rendered, icon slot, control slot, last-row no border.

- [ ] **Step 4: Run tests + build + commit (only if Step 3 happened)**

```bash
cd apps/web && npx vitest run src/components/ui/setting-row.test.tsx && npx next build
```

```bash
git add apps/web/src/components/ui/setting-row.tsx apps/web/src/components/ui/setting-row.test.tsx
git commit -m "$(cat <<'EOF'
feat(ui): add SettingRow primitive

Codifies the repeating settings-row layout (title + description + control)
discovered during the settings audit (3+ uses across N files).

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 5: Record the decision (always)**

Whether or not `SettingRow` was added, record the audit outcome in the plan execution notes (one paragraph in the response back to the coordinator). The settings subagent in Task 7 will be told whether `SettingRow` exists or not.

---

## Task 7: Dispatch consumer migration subagents (parallel)

This is a single coordinator task. The executor calls the `Agent` tool **12 times in one message** so all subagents run in parallel. Each subagent receives the prompt below verbatim (substituting `{AREA}` / `{FILES}` / `{MODEL}`).

**Cross-cutting rules every subagent receives:**

- Read the spec doc first: `docs/superpowers/specs/2026-04-22-design-system-refactor-design.md`.
- Read the relevant DS preview HTML files in `/Users/joaogabriel/Downloads/autocut-design-system/project/preview/` (badges, inputs, buttons, component-pills, component-stat-card, component-step-rail).
- **Standing decisions** (from spec):
  1. Token cleanup is *opportunistic*: when you touch a file, also normalize lingering `bg-zinc-*`, `text-zinc-*`, `border-zinc-*`, raw color stops (`bg-emerald-400/20`, `text-amber-400`, `bg-red-950`, etc.) to semantic tokens. Mapping table:
     - `bg-zinc-900` → `bg-card`
     - `bg-zinc-800/50` → `bg-card/50`
     - `bg-zinc-700` → `bg-muted`
     - `bg-zinc-600/40` → `bg-muted/60` (judgment call — pick the closest token)
     - `border-zinc-700` / `border-zinc-800` → `border-border`
     - `border-zinc-600` → `border-border` (or `border-muted` if more contrast wanted)
     - `text-zinc-200` → `text-foreground`
     - `text-zinc-300` → `text-foreground`
     - `text-zinc-400` → `text-caption`
     - `text-zinc-500` → `text-subtle`
     - `text-zinc-600` → `text-subtle`
     - `bg-emerald-{400,500}/{10,20}` → `<Badge variant="success">` (full migration), or `bg-success/10 border-success/20 text-success` if not a badge
     - `bg-amber-{400}/{10,15,20}` → `<Badge variant="warning">` or `bg-warning/10 ...`
     - `bg-blue-{400,500}/{10,20}` → `<Badge variant="info">` or `bg-info/10 ...`
     - `bg-red-950 / border-red-800 / text-red-400` → `bg-destructive/10 border-destructive/20 text-destructive`
     - `text-emerald-400` → `text-success`
     - `text-amber-400` → `text-warning`
     - `text-blue-400` → `text-info`
     - `text-red-400` / `text-red-500` → `text-destructive`
  2. Page H1 size = single 32px. Use `<PageHeader>` for every page H1.
  3. shadcn `<Input>` has been rewritten — use it for all text/number/url/date inputs.
  4. Migrate inline badge patterns (`inline-flex items-center rounded-full px-2 py-0.5 ...`) to `<Badge variant="…">`.
  5. Migrate inline label+input pairs to `<FormField>` (one canonical label style: `text-xs font-medium text-subtle`).
  6. **Do NOT** invent new primitives. If you find a 3+ pattern that's not covered, note it in your final report — do not extract.
  7. **Do NOT** edit `apps/web/src/app/globals.css`.
  8. **Do NOT** edit files outside your assigned scope.
  9. **No behavior change** — preserve all event handlers, conditional rendering, computed values, refs, and effects exactly as they are.
  10. Native `<select>` elements stay as native `<select>` (deferred follow-up). Style them with the existing inline classes — do NOT migrate to shadcn `Select`.
  11. Native `<button>` elements that act as controls (toggle pills, filter chips) — convert to shadcn `<Button>` only when the button is unambiguously a "button" semantically. Filter pills, segmented controls, etc. that have very specific styling can stay as native `<button>` with token-normalized classes.

- **Verification gate** (must pass before commit): from `apps/web/`, run `npx next build` AND `npx vitest run`. Both must succeed. If either fails, fix or revert before committing.
- **Commit message:** `refactor({area}): adopt design system primitives` with a body listing the primitives adopted, the inline patterns removed, and the token normalizations applied. Co-author trailer.
- **Final report** (as the response to the coordinator): brief bullet list of (a) primitives adopted, (b) inline patterns removed, (c) token normalizations applied, (d) any 3+ patterns spotted that should become future primitives, (e) confirmation that build + tests passed.

### Subagent dispatch table

Run all 12 in parallel via 12 `Agent` tool calls in a single message.

| Subagent ID | Area | Files | Model |
|---|---|---|---|
| **E1** | shorts | `apps/web/src/components/shorts/ShortsForm.tsx`, `apps/web/src/components/shorts/ShortsResultList.tsx` | Sonnet |
| **E2** | queue | `apps/web/src/app/queue/page.tsx`, `apps/web/src/components/queue/BulkScheduleModal.tsx`, `apps/web/src/components/queue/QueueItemCard.tsx`, `apps/web/src/components/queue/ReviewScheduleModal.tsx` | Sonnet |
| **E3** | channels | `apps/web/src/app/channels/page.tsx`, `apps/web/src/components/channels/ChannelCard.tsx`, `apps/web/src/components/channels/CommentSyncSheet.tsx`, `apps/web/src/components/channels/ChannelConfigSheet.tsx` | Sonnet |
| **E4** | settings | `apps/web/src/app/settings/page.tsx`, `apps/web/src/components/settings/AppSettingsSection.tsx`, `apps/web/src/components/settings/ToolsSection.tsx`, `apps/web/src/components/settings/ModelsSection.tsx`, `apps/web/src/components/settings/ChannelsSection.tsx`, `apps/web/src/components/settings/ChannelAuthSection.tsx`, `apps/web/src/components/settings/OAuthProfilesSection.tsx` | **Opus** |
| **E5** | dashboard | `apps/web/src/app/page.tsx`, `apps/web/src/components/dashboard/DashboardClient.tsx` | default |
| **E6** | history | `apps/web/src/components/history/RunDetailPanel.tsx`, `apps/web/src/components/history/RunFilters.tsx`, `apps/web/src/components/history/RunsTable.tsx`, `apps/web/src/components/history/StepRow.tsx` | Sonnet |
| **E7** | dispatcher | `apps/web/src/components/dispatcher/DispatcherBar.tsx` | default |
| **E8** | post-opt + thumbnail | `apps/web/src/components/post-opt/LogoTab.tsx`, `apps/web/src/components/post-opt/TransitionsTab.tsx`, `apps/web/src/components/post-opt/ColorInputField.tsx`, `apps/web/src/components/post-opt/TextStyleEditorPanel.tsx`, `apps/web/src/components/post-opt/SpeedTab.tsx`, `apps/web/src/components/post-opt/AntiDupTab.tsx`, `apps/web/src/components/post-opt/TextOverlayTab.tsx`, `apps/web/src/components/thumbnail/BackgroundsGallery.tsx` | Sonnet |
| **E9** | setup | `apps/web/src/components/setup/ToolRow.tsx`, `apps/web/src/components/setup/WhisperToolRow.tsx`, `apps/web/src/components/setup/CustomPathDialog.tsx` | default |
| **E10a** | pipeline-small | `apps/web/src/components/pipeline/ChannelInfoCard.tsx`, `ClipCard.tsx`, `ClipReviewCard.tsx`, `DownloadInfoCard.tsx`, `HighlightCard.tsx`, `StepRailItem.tsx`, `SubStepProgress.tsx`, `ThumbnailStylePicker.tsx`, `apps/web/src/components/pipeline/steps/LandscapeConfigPanel.tsx`, `StepExecute.tsx`, `StepGeneratingClips.tsx`, `StepReviewClips.tsx`, `StepReviewHighlights.tsx`, `StepUrl.tsx` | **Opus** |
| **E10b** | pipeline-stepmode | `apps/web/src/components/pipeline/steps/StepMode.tsx` (2197 lines — single huge file) | **Opus** |
| **E10c** | pipeline-large | `apps/web/src/components/pipeline/steps/StepReviewMetadata.tsx` (535 lines), `StepThumbnailConfig.tsx` (606 lines), `StepUpload.tsx` (476 lines) | **Opus** |

### Subagent prompt template

Below is the full prompt template. The coordinator substitutes `{AREA_ID}`, `{AREA_NAME}`, `{FILE_LIST}` per row.

```
You are migrating the {AREA_NAME} area of the AutoCut web app to use the
newly extended Design System primitives. You are NOT exploring open
problems — the design and decisions are already made in the spec. Your job
is mechanical: read the spec, read the files in your scope, apply the
substitutions, run the gates, commit.

## Spec
Read in full before doing anything:
docs/superpowers/specs/2026-04-22-design-system-refactor-design.md

## Plan reference
docs/superpowers/plans/2026-04-22-design-system-refactor.md
(See "Task 7" — your area is {AREA_ID}.)

## Design System reference
Read the relevant DS HTML previews in:
/Users/joaogabriel/Downloads/autocut-design-system/project/preview/
(badges.html, inputs.html, buttons.html, component-pills.html,
component-stat-card.html, component-step-rail.html). These are the visual
contracts.

## Available primitives (already shipped in commits A–E of this plan)
- <PageHeader title description? actions?> from @/components/ui/page-header
- <FormField label description? error? required? htmlFor?>{control}</FormField> from @/components/ui/form-field
- <Badge variant="default|secondary|destructive|outline|ghost|link|success|warning|info|brand"> from @/components/ui/badge
- <Input> from @/components/ui/input — has been rewritten to bg-surface + brand focus ring
- <TabsTrigger value count? > from @/components/ui/tabs — count pill renders when > 0
- <Button variant="default|destructive|outline|secondary|ghost|link|brand" size="default|xs|sm|lg|icon|icon-xs|icon-sm|icon-lg"> from @/components/ui/button
- <Label> from @/components/ui/label — escape hatch for inline label cases (checkbox rows, dropdown-on-same-line)
- (If Task 6 added it: <SettingRow title description? icon? control> from @/components/ui/setting-row — settings subagent will be told whether it exists)

## Your scope
Modify ONLY these files:
{FILE_LIST}

Touching any other file is forbidden.

## Migration steps for each file
1. Identify inline patterns that have a primitive equivalent:
   - H1 page title pattern → <PageHeader>
   - Inline label+input pair → <FormField> wrapping an <Input>
   - Inline rounded-full status badges → <Badge variant="…">
   - Raw shadcn-style buttons → <Button>
2. Replace inline classnames `rounded-lg bg-surface border border-border ... focus:border-brand/60` on plain <input> elements with <Input> from @/components/ui/input.
3. For inline label markup (`text-xs text-subtle mb-1 block` / `text-xs font-medium text-caption` / `text-xs font-medium text-muted-foreground`), wrap the label+control pair in <FormField label="…">.
4. For inline status badges (`inline-flex items-center rounded-full ... bg-{emerald|amber|blue|red|zinc}-…`), replace with <Badge variant="success|warning|info|destructive|secondary"> + the original text content.
5. Apply the token normalization map from the plan's "Cross-cutting rules" to any remaining raw color usage in the file.
6. Preserve ALL event handlers, refs, conditional rendering, computed values, effects, and prop signatures exactly. Zero behavior change.
7. Native <select> stays native (deferred). Native <button> elements that are filter pills / segmented controls may stay native if the styling doesn't fit shadcn <Button>.

## Cross-area dependencies (only if relevant to your area)
- E2 (queue): QueueItemCard.tsx imports ChannelAvatar from channels/ChannelCard.tsx. Treat ChannelAvatar as read-only — its export shape is owned by E3.
- E3 (channels): ChannelAvatar is a named export from ChannelCard.tsx and is consumed by QueueItemCard.tsx. Do NOT change ChannelAvatar's export shape, props, or signature.
- E2 (queue): the page-level pill-tabs implementation in app/queue/page.tsx must be replaced with shadcn <Tabs> + <TabsTrigger value count>. Status filter ("Para Subir" / "Histórico") becomes two <TabsTrigger>s with count props. The two-state setStatusFilter handler stays.

## Verification gate (run from apps/web/)
1. npx next build — must succeed
2. npx vitest run — must succeed
3. Self-grep your owned files for residual raw colors:
   grep -nE 'bg-(zinc|emerald|amber|blue|red)-[0-9]|text-(zinc|emerald|amber|blue|red)-[0-9]|border-zinc-[0-9]' {your file paths}
   Expected: zero hits in your files (intentional opaque hex values for one-off avatar fallbacks etc. are fine — call those out in the report).

## Commit
git add {your file paths}
git commit -m "refactor({AREA_ID}): adopt design system primitives

- Adopted: <list primitives used: PageHeader, FormField, Input, Badge variants, Tabs count slot, Button>
- Removed inline: <list patterns: H1 inline / form labels / status badges / pill tabs / etc.>
- Token normalization: <files where zinc-* / raw colors mapped to tokens>

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"

## Final report (return to coordinator)
Brief bullet list:
- (a) Primitives adopted
- (b) Inline patterns removed
- (c) Token normalizations applied (which files)
- (d) Any 3+ patterns spotted that should become future primitives (note only — do NOT extract)
- (e) Confirmation that npx next build + npx vitest run both passed
- (f) Commit SHA
```

### Coordinator workflow

- [ ] **Step 1: Confirm pre-conditions**

Run from repo root:
```bash
git status
```
Expected: clean tree (Tasks 1–6 have been committed).

```bash
cd apps/web && npx next build && npx vitest run
```
Expected: success — primitives are stable.

- [ ] **Step 2: Dispatch all 12 subagents in parallel**

Send a single message containing **12 Agent tool calls**. Each call uses the `Agent` tool with:
- `description`: e.g. `"Migrate {AREA_NAME} area"` (3-5 words)
- `subagent_type`: `general-purpose`
- `model`: `opus` for E4, E10a, E10b, E10c; `sonnet` for E1, E2, E3, E6, E8; omit (default) for E5, E7, E9
- `prompt`: the full template above with `{AREA_ID}`, `{AREA_NAME}`, `{FILE_LIST}` substituted

**The dispatch must be a single tool-use block with 12 Agent calls — not 12 separate messages.**

- [ ] **Step 3: Receive reports**

For each subagent that returns:
- If success report + commit SHA: mark area done.
- If failure: read the failure summary, decide whether to (a) re-dispatch with a clarification, (b) split the area further, or (c) handle the residual files inline as a coordinator-level commit.

- [ ] **Step 4: Sanity-check the resulting branch state**

Run from repo root:
```bash
git log --oneline -20
```
Expected: 12 new commits (one per subagent), all on `main`, plus the 6 primitive commits from Tasks 1–6 (or 5 if Task 6 skipped SettingRow).

```bash
cd apps/web && npx next build && npx vitest run
```
Expected: success across the whole suite.

If any subagent's commit broke the build/tests for another subagent's area (rare — file-disjoint scopes prevent it but cross-area imports could): the coordinator investigates and fixes inline.

---

## Task 8: Final grep + verification pass

**Files:**
- Read-only audit across `apps/web/src/`
- Possibly modify any straggler files surfaced by grep

- [ ] **Step 1: Grep for inline H1 pattern outside PageHeader**

```bash
cd apps/web && grep -rn 'font-display text-\[' src/ --include='*.tsx'
```
Expected: ONE hit only — `src/components/ui/page-header.tsx`.

If any other file matches, that subagent missed a `<PageHeader>` migration. Fix inline by editing the file and replacing the inline H1 with `<PageHeader title="…" />`.

- [ ] **Step 2: Grep for inline input pattern**

```bash
cd apps/web && grep -rn 'rounded-lg bg-surface border border-border' src/ --include='*.tsx'
```
Expected: ZERO hits in form/input contexts. Hits inside `bg-surface` cards (non-input usage like preview boxes) are acceptable — confirm by reading the matching lines.

If any `<input>`-element line still has the inline pattern, replace with `<Input>`.

- [ ] **Step 3: Grep for inline status badges**

```bash
cd apps/web && grep -rnE 'inline-flex items-center.*rounded-full.*bg-(emerald|amber|zinc|red|blue)-' src/ --include='*.tsx'
```
Expected: ZERO hits in feature files. Replace any straggler with `<Badge variant="…">`.

- [ ] **Step 4: Grep for inline form-label patterns**

```bash
cd apps/web && grep -rnE 'text-xs (text-subtle mb-1 block|font-medium text-(caption|muted-foreground))' src/ --include='*.tsx'
```
Expected: ZERO hits in feature files. Replace any straggler with `<FormField>` or standalone `<Label>` from shadcn.

- [ ] **Step 5: Grep for residual raw zinc/emerald/amber/blue/red color usage in touched files**

```bash
cd apps/web && grep -rnE 'bg-(zinc|emerald|amber|blue|red)-[0-9]|text-(zinc|emerald|amber|blue|red)-[0-9]|border-(zinc|emerald|amber|blue|red)-[0-9]' src/ --include='*.tsx'
```
Inspect each remaining hit:
- If the file was *not* touched by Task 7 (i.e. not in any subagent's scope), leave it (Q1/B opportunistic — untouched files stay).
- If the file *was* touched, normalize the color to a semantic token using the map in the Task 7 prompt.

- [ ] **Step 6: Run full vitest suite**

```bash
cd apps/web && npx vitest run
```
Expected: PASS — all primitive tests + the existing switch test.

- [ ] **Step 7: Run full Next.js build**

```bash
cd apps/web && npx next build
```
Expected: success. No type errors. No new warnings introduced.

- [ ] **Step 8: Manual route spot-check**

Start the dev server:
```bash
cd apps/web && npm run dev
```

Visit each route at http://localhost:3201 and visually confirm nothing is broken. Acceptance = page loads, no console errors, no visibly-broken layouts. Visual regressions explicitly accepted in spec (page H1 sizes growing 28→32px on settings/queue/channels; queue tabs becoming shadcn underlined-tabs) are expected.

Routes:
- `/` — dashboard
- `/shorts` — shorts form
- `/queue` — queue page (toggle "Para Subir" / "Histórico" tabs, verify count pills render)
- `/channels` — channel list + add form, click "Authorize" on a channel to confirm OAuth flow still triggers
- `/settings` — each tab (Tools, App, Models, Channels, OAuth)
- `/setup` — setup gate (toggle a tool detail open)
- `/pipeline` — start a pipeline run from a URL, click through each step, verify gate UI still works
- `/post-opt` — each tab (Logo, Transitions, Speed, AntiDup, TextOverlay)
- `/thumbnail` — backgrounds gallery
- `/history` — runs table + filters

Stop dev server: `Ctrl-C`.

- [ ] **Step 9: Commit any straggler fixes (only if Steps 1–5 found anything)**

```bash
git add <files modified in this task>
git commit -m "$(cat <<'EOF'
chore(ui): final grep + verification pass

Cleans up residual inline patterns missed by area subagents and confirms
build + tests + manual spot-check all green.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

If Steps 1–5 found nothing, no commit is needed — the verification itself is the value.

- [ ] **Step 10: Final summary report**

Write a one-paragraph summary to the user:
- Total commits added (Tasks 1–8)
- Primitives shipped (Badge variants, Input rewrite, PageHeader, FormField, Tabs count slot, optionally SettingRow)
- Areas migrated (12)
- Build + tests status
- Any deferred follow-ups noted (e.g. native `<select>` migration to shadcn `Select`, soft `Badge.destructive` if needed later)

---

## Self-review (already applied)

**Spec coverage check:**
- Decision 1 (opportunistic token cleanup) → covered in Task 7 cross-cutting rules + Task 8 Step 5 grep.
- Decision 2 (single 32px PageHeader) → encoded in Task 3 component + Task 7 rule "Page H1 size = single 32px".
- Decision 3 (Input rewrite) → Task 2.
- Decision 4 (queue tabs migrate to shadcn Tabs) → Task 5 + Task 7 cross-area note for E2.
- Decision 5 (pipeline included) → Task 7 E10a/b/c.
- Decision 6 (render tests for new primitives) → Tasks 1, 2, 3, 4, 5 each include a test step.
- Decision 7 (Badge variants, no StatusBadge wrapper) → Task 1.
- Decision 8 (FormField with Label escape hatch) → Task 4 + Task 7 rule "shadcn <Label> stays exported".

**Type consistency:** `count?: number` used consistently in Tabs test + Tabs impl + subagent prompt. `variant="success|warning|info|brand"` used consistently in Badge test + impl + subagent prompts.

**No placeholders:** every code block contains the actual content. Subagent prompt template is fully formed; substitution variables (`{AREA_ID}` etc.) are explicitly defined.

---

## Execution handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-22-design-system-refactor.md`. Two execution options:

1. **Subagent-Driven (recommended)** — Dispatch a fresh subagent per task (Tasks 1–6), review between tasks, then a coordinator runs Task 7's parallel dispatch. Fast iteration, clean two-stage review.

2. **Inline Execution** — Execute Tasks 1–6 in this session via `executing-plans`, batch with checkpoints. Coordinator runs Task 7 inline.

Which approach?
