# E2E Tests: Review Metadata Pipeline Step

Preciso criar testes E2E para o step de Review Metadata do pipeline do AutoCut. O teste deve rodar como se fosse um usuario passando pelo fluxo completo.

## Contexto

O AutoCut tem um pipeline de processamento de videos com estados sequenciais. O step `WAITING_REVIEW_METADATA` fica entre `WAITING_THUMBNAIL_CONFIG` e `WAITING_REVIEW_CLIPS`. Nesse step, o backend pode gerar metadata via Claude CLI (titulo, descricao, tags, thumbnail_text por clip), e o usuario pode editar e avancar.

## Infraestrutura E2E existente

Todos os testes E2E ficam em `e2e/specs/` e usam Playwright. A infra ja existe:

- **Config**: `e2e/playwright.config.ts` — serial, 1 worker, timeout 120s
- **Helpers**: `e2e/helpers/` — api.ts, db.ts, pipeline.ts, setup.ts, env.ts
- **Pattern DB**: `getPipelineClips(runId)` retorna clips do SQLite direto (better-sqlite3)
- **Pattern API**: `advanceGate(runId, body)` -> POST `/api/pipeline/runs/:id/advance`
- **Pattern poll**: `waitForState(runId, target, timeout)` — poll ate estado desejado
- **Seed**: `resetPipelineData()` limpa runs/clips/highlights; `seedFakeChannel()` cria canal fake

Os testes full pipeline (`e2e/specs/pipeline-longform-full.spec.ts`) usam `test.describe.serial` com `sharedRunId` compartilhado entre steps. Exemplo de pattern:

```typescript
test.describe.serial('Pipeline Longform (real)', () => {
  test.skip(process.env.E2E_NETWORK !== 'true', 'requires network');
  test.beforeAll(() => { resetPipelineData(); seedFakeWhisperModel(); });

  test('C1: URL -> clips', async () => {
    const runId = await createRun();
    sharedRunId = runId;
    await advanceGate(runId, { url: FIXTURE_URL });
    // ...poll, assert...
  });

  test('D1: thumbnail generation', async () => {
    // usa sharedRunId do C1
  });
});
```

## APIs do Metadata (spec 112)

3 endpoints implementados em `server/internal/api/handlers/metadata_handler.go`:

1. **POST `/api/metadata/runs/{id}/generate`** — batch AI generation (async, 202)
   - Requer estado `WAITING_REVIEW_METADATA`
   - Requer Claude CLI disponivel (503 se nao tem)
   - Retorna `{ job_id, total_clips }`

2. **POST `/api/metadata/runs/{id}/clips/{clipId}/generate`** — single clip regen (sync, 200)
   - Requer estado `WAITING_REVIEW_METADATA`
   - Retorna `{ clip_id, title, description, tags, thumbnail_text, category_id }`

3. **POST `/api/pipeline/runs/{id}/gates/review-metadata`** — save edits + advance
   - Body: `{ clips: [{ id, title, description, tags, thumbnail_text }] }`
   - Avanca `WAITING_REVIEW_METADATA` -> `WAITING_REVIEW_CLIPS`
   - Retorna `{ state: "WAITING_REVIEW_CLIPS" }`

## Transicao de estado

`WAITING_THUMBNAIL_CONFIG` -> advance via `/api/pipeline/runs/{id}/advance` -> `WAITING_REVIEW_METADATA`

## DB helper existente — precisa evoluir

`getPipelineClips()` em `e2e/helpers/db.ts` retorna: `{ id, run_id, thumbnail_style, thumbnail_path, file_path }`. Precisa adicionar os campos de metadata: `title, description, tags, thumbnail_text`.

## O que criar

### 1. Evoluir `e2e/helpers/db.ts`

Adicionar `title`, `description`, `tags`, `thumbnail_text` ao SELECT e tipo de retorno de `getPipelineClips()`.

### 2. Criar `e2e/specs/112-metadata.spec.ts`

Testes API-level (sem UI/Electron) que cobrem o fluxo de metadata como se fosse um usuario:

**Setup**: Para chegar ao estado `WAITING_REVIEW_METADATA`, precisa:
- Usar `resetPipelineData()` no beforeAll
- Criar um run, avancar ate `WAITING_THUMBNAIL_CONFIG` (precisa de rede — yt-dlp + FFmpeg)
- Ou: usar DB direto para forcar o estado com `openDbRW()` se quiser testes rapidos sem rede

**Sugestao de testes (serial, shared run)**:

```
E1: Rejeita batch generate fora do estado WAITING_REVIEW_METADATA (409 Conflict)
E2: Batch generate retorna 202 com job_id (se Claude CLI disponivel) ou 503 (se nao disponivel)
E3: POST review-metadata salva edits e avanca para WAITING_REVIEW_CLIPS
E4: Verifica que clips tem metadata persistida no DB apos review-metadata
E5: Rejeita review-metadata fora do estado correto
```

**Abordagem para seeding rapido (sem rede)**: Criar helper que insere run + clips diretamente no SQLite para testar os endpoints de metadata isoladamente — sem precisar do pipeline completo. Algo como:

```typescript
function seedRunAtState(state: string): number {
  const db = openDbRW();
  // INSERT INTO pipeline_runs (state, url, ...) VALUES (?, ...)
  // INSERT INTO pipeline_clips (run_id, ...) VALUES (?, ...)
  db.close();
  return runId;
}
```

### 3. Estender `pipeline-longform-full.spec.ts` (opcional)

Apos D1 (thumbnail), adicionar:
- E1: advance thumbnail config -> `WAITING_REVIEW_METADATA`
- F1: POST review-metadata com dados editados -> `WAITING_REVIEW_CLIPS`
- Verificar DB reflete metadata

## Arquivos de referencia

Leia estes arquivos para entender os patterns e contratos:

- `e2e/specs/pipeline-longform-full.spec.ts` — pattern de teste serial full-pipeline
- `e2e/helpers/db.ts` — helpers de DB (precisa evoluir getPipelineClips)
- `e2e/helpers/pipeline.ts` — advanceGate, waitForState, assertClipFilesExist
- `e2e/helpers/api.ts` — createRun, getRun
- `e2e/helpers/env.ts` — API_BASE, TEST_DATA_DIR
- `e2e/helpers/setup.ts` — seedFakeWhisperModel, systemHas
- `server/internal/api/handlers/metadata_handler.go` — 3 endpoints + request/response types
- `server/internal/pipeline/service.go` — state machine (linhas ~308-312: advanceSimple THUMBNAIL->METADATA->CLIPS)

## Regras

- Seguir exatamente o pattern dos testes existentes (imports, helpers, describe.serial, skip conditions)
- Testar via API (fetch), nao via UI — como os testes de pipeline existentes
- Usar `expect` do Playwright
- DB assertions via `better-sqlite3` direto (pattern existente)
- Nao testar geracao AI real (Claude CLI pode nao estar no PATH do CI) — testar os endpoints de save/advance que nao dependem de AI
- Para testes que dependem de Claude CLI, usar `test.skip` condicional
- Arquivo de saida: `e2e/specs/112-metadata.spec.ts`
