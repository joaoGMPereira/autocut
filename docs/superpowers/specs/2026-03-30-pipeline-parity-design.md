# Pipeline Parity Design — AutoCut v2

**Date:** 2026-03-30
**Status:** Approved
**Scope:** Evoluir `/pipeline` do AutoCut (Next.js + Go) até atingir paridade com o repo Kotlin em 7 fases.

---

## 1. Decisões Arquiteturais

### Backend como fonte canônica de estado

O pipeline passa a ser uma entidade de primeira classe no backend. O SQLite (Go) é a fonte de verdade para todo contexto inter-step. O frontend armazena apenas `activeRunId`.

**Razão:** App Electron desktop — o backend é o processo principal. localStorage é cache de UI, não estado de domínio. Caminhos de arquivo, outputs de steps, highlights[] pertencem no DB.

### Session model via `pipeline_runs`

A tabela `pipeline_runs` já existe com CRUD completo. Adiciona-se:
- `step_outputs_json TEXT` — blob JSON com outputs acumulados por step
- Tabela `pipeline_run_steps` — log granular por step (status, job_id, timing)

### Step handlers permanecem independentes

Cada endpoint existente (`/api/download`, `/api/cut`, etc.) aceita `session_id` opcional. Quando presente, após `done`, persiste output. Caso ausente, comportamento atual inalterado. Sem acoplamento forçado.

### SSE pattern inalterado

POST → 202 `{job_id}` → GET `/{id}/stream`. Nenhuma mudança de protocolo.

---

## 2. Modelo de Dados

### Migração (additive, non-breaking)

```sql
-- Migration 2: pipeline parity
ALTER TABLE pipeline_runs ADD COLUMN step_outputs_json TEXT NOT NULL DEFAULT '{}';

CREATE TABLE IF NOT EXISTS pipeline_run_steps (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  run_id      INTEGER NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
  step        TEXT NOT NULL,
  status      TEXT NOT NULL DEFAULT 'pending',
  job_id      TEXT,
  error       TEXT NOT NULL DEFAULT '',
  started_at  INTEGER,
  finished_at INTEGER
);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_steps_run_id ON pipeline_run_steps(run_id);
```

### StepOutputs JSON schema

```json
{
  "download": {
    "file_path": "/path/video.mp4",
    "title": "Video Title",
    "duration_sec": 3612,
    "thumbnail_path": "/path/thumb.jpg",
    "metadata_json_path": "/path/meta.json",
    "platform": "youtube",
    "quality": "1080p"
  },
  "cut": {
    "output_path": "/path/cut.mp4",
    "start_sec": 1805,
    "end_sec": 3482
  },
  "optimize": {
    "output_path": "/path/optimized.mp4",
    "strategy": "complete",
    "removed_silences": 47,
    "original_sec": 3612,
    "final_sec": 3201
  },
  "transcript": {
    "json_path": "/path/transcript.json",
    "language": "pt",
    "segment_count": 284
  },
  "highlights": [
    { "start_sec": 1805, "end_sec": 3482, "title": "Topic", "confidence": 0.87, "time_range": "30:05 ao 58:02" }
  ],
  "shorts": [
    { "file_path": "/path/short_1.mp4", "thumbnail_path": "/path/t.jpg", "subtitle_path": "/path/s.srt", "title": "Short", "duration_sec": 47 }
  ],
  "thumbnail": {
    "output_path": "/path/thumb.png",
    "strategy": "branded"
  },
  "upload": {
    "youtube_id": "abc123",
    "youtube_url": "https://youtube.com/watch?v=abc123",
    "channel_id": 2,
    "scheduled_at": null
  }
}
```

---

## 3. API Contracts

### Novos endpoints

```
POST   /api/pipeline/runs
       Body: { url: string, mode: "manual"|"auto", channel_id?: number }
       → 201 { run_id: number }

GET    /api/pipeline/runs/:id
       → 200 { run: PipelineRun, step_outputs: StepOutputs, steps: PipelineRunStep[] }

GET    /api/pipeline/runs
       Query: ?limit=20&offset=0&channel_id=N
       → 200 { runs: PipelineRun[], total: number }

DELETE /api/pipeline/runs/:id
       → 204

POST   /api/highlights
       Body: { video_path, session_id?, language?, min_duration?, max_duration?, threshold?, long_form? }
       → 202 { job_id }

GET    /api/highlights/:id/stream
       SSE: progress(step, percent) | highlight(start,end,title,confidence) | done | error
```

### Endpoints existentes — adições opcionais ao body

```
POST /api/download    + session_id?, quality?, with_thumbnail?, with_metadata?
POST /api/cut         + session_id?
POST /api/optimize    + session_id?, strategy?, preview?
POST /api/transcript  + session_id?, transcript_path? (aceita path de arquivo além de inline json)
POST /api/analyze     + session_id?
POST /api/shorts      + session_id?, count?, min_duration?, max_duration?, with_subtitles?, with_thumbnail?
POST /api/thumbnail   + session_id?, strategy, input_path
POST /api/upload      + session_id?, channel_id, ai_metadata?, scheduled_at?, playlist_id?, privacy?
```

### Bug fix obrigatório

Todos os `done` events do Go passam a incluir `success: true`. Frontend usa `event.type === "done"` para status, não `event.data.success`.

---

## 4. Frontend Architecture

### Stores

**`pipelineStore.ts`** (novo):
```ts
interface PipelineRun {
  id: number; url: string; status: string; progress: number
  currentStep: string; stepOutputs: StepOutputs; steps: PipelineRunStep[]
}
interface PipelineState {
  activeRunId: number | null
  run: PipelineRun | null
  isLoading: boolean
  createRun(goUrl: string, url: string): Promise<number>
  loadRun(goUrl: string, id: number): Promise<void>
  refreshRun(goUrl: string): Promise<void>
  clearRun(): void
}
```

**`highlightStore.ts`** (novo, FASE 2):
```ts
interface HighlightJob { status: JobStatus; logs: string[]; highlights: Highlight[] }
interface HighlightState {
  jobs: Record<string, HighlightJob>
  startHighlights(goUrl: string, payload: HighlightsPayload): Promise<string>
}
```

**Stores existentes:** `downloadStore` e `processorStore` capturam `file_path`/`output` do `done` event (fix bug).

### Pipeline Page — Layout Vertical com Conectores

Componentes em `src/components/pipeline/`:
```
PipelineHeader.tsx      — URL input + botão new run
StepCard.tsx            — card genérico: header(nome,badge) + slot + LogViewer
StepConnector.tsx       — linha vertical dashed/solid + label do dado que flui
steps/
  DownloadStep.tsx      — quality, with_thumbnail, with_metadata
  OptimizeStep.tsx      — strategy selector (audio|transcript|complete), preview mode
  TranscriptStep.tsx    — language selector
  HighlightsStep.tsx    — threshold, min/max duration, HighlightList
  ShortsStep.tsx        — count, min/max duration, with_subtitles, with_thumbnail
  ThumbnailStep.tsx     — strategy selector
  UploadStep.tsx        — channel selector, privacy, AI metadata, schedule
LogViewer.tsx           — reutilizado do atual
HighlightList.tsx       — lista com time ranges e confidence badges
```

**StepCard estados:** `idle | running | done | error | skipped`
**Connector:** linha sólida quando predecessor=done (com label do path), tracejada quando idle/error.

---

## 5. Fases de Implementação

### FASE 1 — Pipeline com chaining automático (fundação)
- Bug fix SSE stores
- DB migration (step_outputs_json + pipeline_run_steps)
- PipelineRunRepo: UpdateStepOutput, ListAll, GetWithSteps
- PipelineHandler: POST/GET/LIST/DELETE /api/pipeline/runs
- Todos handlers: session_id opcional + UpdateStepOutput após done
- pipelineStore.ts
- Pipeline page redesign (vertical flow, todos os 7 StepCards)
- Download: quality + with_thumbnail + with_metadata

### FASE 2 — Highlights e detecção inteligente
- HighlightsHandler: POST /api/highlights + SSE stream
- Lógica: Whisper cache + TopicTransitionDetector + HighlightDetector (merge/filter)
- Frontend: HighlightsStep + HighlightList + highlightStore

### FASE 3 — Shorts pipeline completo
- ShortsHandler: melhorar com count, min/max duration, subtitles, thumbnail por short
- Processamento paralelo (até 5 goroutines) com progress aggregado
- Frontend: ShortsStep com progresso por short

### FASE 4 — Upload YouTube completo
- UploadHandler: channel_id, ai_metadata (Ollama), schedule, playlist, privacy
- OAuth por perfil já existe — wiring com channel_id
- Frontend: UploadStep com todos os campos

### FASE 5 — Thumbnail avançado
- ThumbnailHandler: 4 estratégias (branded, centered, shorts, ai)
- Frontend: ThumbnailStep com strategy picker e preview

### FASE 6 — Persistência e histórico
- GET /api/pipeline/runs (list) endpoint
- History panel/page com runs anteriores
- Click em run → carrega no pipeline page

### FASE 7 — Batch e automação
- POST /api/pipeline/batch { urls[], preset_id? }
- Preset system: salvar/carregar configurações de pipeline
- Queue view: múltiplos runs em paralelo

---

## 6. Contexto técnico

- Go: 1.23, `github.com/joaoGMPereira/autocut/server`, CGO_ENABLED=0, modernc.org/sqlite
- Next.js: 16, App Router, `'use client'` only when needed
- Zustand: 5.x, sem persist (state de domínio no DB)
- shadcn/ui: button, scroll-area, badge, separator, tooltip (já instalados)
- Portas dev: Go=4071, Next.js=3201
- goUrl via `useAppStore().goUrl` — nunca hardcoded
- SSE pattern: POST → 202 {job_id} → EventSource GET /{id}/stream
- Migrations: runner incremental em `internal/database/migrations.go`
