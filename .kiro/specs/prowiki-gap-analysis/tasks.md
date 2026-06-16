# Implementation Plan: ProWiki Gap Analysis

## Overview

This plan resolves 17 identified gaps in the ProWiki codebase, spanning ingestion unification, LLM extraction pipeline completion, code-style baseline, graph synthesis, token routing, prompt registry, PII scrubbing, CLI/API surface, observability, web UI, schema indexes, key rotation, AST parsing, tokenizer accuracy, and entry-point consolidation. Tasks are ordered to build foundations first (domain types, migrations, shared infrastructure) before layering services and then the CLI/API/UI surface on top.

## Tasks

- [x] 1. Database migrations and domain model extensions
  - [x] 1.1 Add migration 000003 with indexes and constraints
    - Create `migrations/000003_indexes.up.sql` with all five `CREATE INDEX IF NOT EXISTS` / `CREATE UNIQUE INDEX IF NOT EXISTS` statements from the design
    - Create `migrations/000003_indexes.down.sql` that drops each index and constraint
    - _Requirements: 13.1, 13.2, 13.3, 13.4, 13.5, 13.6_

  - [x] 1.2 Add `payload` and `fs_location` columns via migration
    - Extend migration 000003 (or a new 000004) with `ALTER TABLE job_queue ADD COLUMN payload TEXT NOT NULL DEFAULT ''` and `ALTER TABLE projects ADD COLUMN fs_location TEXT NOT NULL DEFAULT ''`
    - Add `INSERT OR IGNORE` rows for `level_4_edge_case` and `intersection_synthesis` stages in `000002_seed_prompts.up.sql` equivalent (new migration to avoid touching existing files)
    - _Requirements: 6.3, 6.4_

  - [x] 1.3 Extend domain types with new fields and functions
    - Add `Payload string` field to `domain.Job` struct in `internal/domain/entities.go`
    - Add `FSLocation string` field to `domain.Project` struct
    - Add `FeaturePair` struct to `internal/domain/entities.go`
    - Add `MacroPipeline` struct to `internal/domain/entities.go`
    - Create `internal/domain/language.go` with the `LanguageFromPath(path string) Language` pure function
    - Add `ErrConflict` sentinel to `internal/domain/errors.go`
    - _Requirements: 1.5, 13.4, 13.5_

  - [x]* 1.4 Write property test for LanguageFromPath
    - **Property 3: Language detection from file path**
    - **Validates: Requirements 1.5, 7.6**
    - _File: `internal/domain/language_test.go`_


- [x] 2. Structured logger and metrics packages
  - [x] 2.1 Create `internal/logger` package
    - Implement `Logger` struct wrapping `log/slog` with JSON handler, RFC3339Milli time format
    - Expose `New(level slog.Level, w io.Writer) *Logger` and `Info`, `Warn`, `Error`, `Debug` methods
    - Read `PROWIKI_LOG_LEVEL` env var for default level; default to `info` when unset or unrecognized
    - _Requirements: 10.1, 10.4, 10.5_

  - [x]* 2.2 Write property test for structured logger JSON output
    - **Property 22: Structured log lines are valid JSON with required fields**
    - **Validates: Requirements 10.1**
    - _File: `internal/logger/logger_test.go`_

  - [x] 2.3 Create `internal/metrics` package
    - Implement `Registry` struct with `JobsTotal` CounterVec, `LLMRequests` CounterVec, and `QueueDepth` GaugeVec
    - Implement `NewRegistry()`, `IncJobsTotal`, `IncLLMRequests`, `SetQueueDepth`, and `Handler()` methods
    - _Requirements: 11.1, 11.5, 11.6_

  - [x]* 2.4 Write unit tests for metrics registry
    - Test counter increments produce correct label values
    - Test `Handler()` returns a valid `http.Handler`
    - _Requirements: 11.1_


- [x] 3. Tokenizer — TikToken BPE counter
  - [x] 3.1 Implement `TikTokenCounter` in `internal/tokenizer`
    - Add dependency `github.com/pkoukk/tiktoken-go` to `go.mod`
    - Implement `TikTokenCounter` struct with `Count(text string) (int, error)` using `cl100k_base` encoding
    - Constructor returns `(*TikTokenCounter, error)`; error if encoding cannot be loaded
    - _Requirements: 16.1, 16.2, 16.3_

  - [x] 3.2 Update `domain.Counter` interface and implement `CountMessages`
    - Update `domain.Counter` interface in `internal/domain/interfaces.go` to include `CountMessages(msgs []domain.Message) (int, error)`
    - Implement `CountMessages` on both `TikTokenCounter` and `HeuristicCounter`: sum `Count(msg.Content)` + 4 per message
    - _Requirements: 16.6_

  - [x]* 3.3 Write property test for heuristic counter accuracy
    - **Property 19: Heuristic counter within factor of 2 of BPE**
    - **Validates: Requirements 16.5**
    - _File: `internal/tokenizer/counter_test.go`_

  - [x]* 3.4 Write property test for CountMessages sum invariant
    - **Property 20: CountMessages sum invariant**
    - **Validates: Requirements 16.6**
    - _File: `internal/tokenizer/counter_test.go`_


- [x] 4. AST parser — language-aware structural hashing
  - [x] 4.1 Implement language-specific stripping rules in `internal/ast/parser.go`
    - Add stripping for Go/JS/TS: `//` line comments, `/* */` block comments (multi-line)
    - Add stripping for Python: `#` line comments, `"""..."""` and `'''...'''` docstrings
    - Apply blank-line stripping as the final normalization pass for all languages including unknown
    - Return a valid non-empty hash for unknown/empty language without error
    - _Requirements: 15.1, 15.2, 15.3, 15.4, 15.6_

  - [x]* 4.2 Write property test for Go/JS/TS comment invariance
    - **Property 16: AST hash ignores comments (Go/JS/TS)**
    - **Validates: Requirements 15.1, 15.3**
    - _File: `internal/ast/parser_test.go`_

  - [x]* 4.3 Write property test for Python docstring invariance
    - **Property 17: AST hash ignores Python docstrings and comments**
    - **Validates: Requirements 15.2**
    - _File: `internal/ast/parser_test.go`_

  - [x]* 4.4 Write property test for blank-line invariant
    - **Property 18: AST hash blank-line invariant**
    - **Validates: Requirements 15.4, 15.6**
    - _File: `internal/ast/parser_test.go`_


- [x] 5. Scrubber — new PII/secrets regex rules
  - [x] 5.1 Add four new regex patterns to `internal/scrub/scrubber.go`
    - JWT: `eyJ[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]*`
    - PEM block: `-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----[\s\S]*?-----END[^\n]*PRIVATE KEY-----`
    - GitHub PAT: `gh[pousr]_[A-Za-z0-9]{36}`
    - Hex in KV: `(?i)(?:password|secret|key|token|api_key)["\s:=]+["']?([0-9a-fA-F]{32,})["']?`
    - Confirm `Scrub` return signature is `(redacted string, hits int)`
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_

  - [x]* 5.2 Write property test for JWT redaction
    - **Property 13: JWT token redaction**
    - **Validates: Requirements 7.1, 7.5**
    - _File: `internal/scrub/scrubber_test.go`_

  - [x]* 5.3 Write property test for PEM private key block redaction
    - **Property 14: PEM private key block redaction**
    - **Validates: Requirements 7.2**
    - _File: `internal/scrub/scrubber_test.go`_

  - [x]* 5.4 Write property test for GitHub PAT and hex-in-KV redaction
    - **Property 15: GitHub PAT and hex-in-KV redaction**
    - **Validates: Requirements 7.3, 7.4, 7.5**
    - _File: `internal/scrub/scrubber_test.go`_


- [x] 6. Checkpoint — Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 7. Ingest service — unify and replace versioning package
  - [x] 7.1 Update store interfaces in `internal/app/ingest`
    - Add `FileVersionStore.LatestByFileID`, `InsertVersion`, `SetLatest`, `CloneJunctions` methods
    - Add `JobStore.EnqueueMany` and `ResetCompletedForTarget` methods
    - Implement any missing methods on the SQLite store implementations
    - _Requirements: 1.1_

  - [x] 7.2 Implement clone-on-unchanged-hash path in `internal/app/ingest/service.go`
    - When AST hash matches latest version: insert new `FileVersion`, clone `file_features` junctions, mark new as latest and old as non-latest within one transaction, skip enqueuing `StageLevel1Overview`
    - Use `domain.LanguageFromPath` to derive language for the parser call
    - _Requirements: 1.2, 1.5_

  - [x] 7.3 Write property test for ingest clone-on-unchanged-hash
    - **Property 1: Ingest clone-on-unchanged-hash**
    - **Validates: Requirements 1.2**
    - _File: `internal/app/ingest/service_test.go`_

  - [x] 7.4 Implement cascade-invalidation path in `internal/app/ingest/service.go`
    - When AST hash differs: insert new `FileVersion`, mark latest, reset completed jobs to `pending` with priority+10, enqueue one `StageLevel1Overview` job targeting new version, within one transaction
    - _Requirements: 1.3_

  - [x] 7.5 Write property test for ingest cascade invalidation on changed hash
    - **Property 2: Ingest cascade invalidation on changed hash**
    - **Validates: Requirements 1.3**
    - _File: `internal/app/ingest/service_test.go`_

  - [x] 7.6 Delete `internal/versioning` package
    - Remove all files in `internal/versioning/`
    - Update any import references to the deleted package
    - _Requirements: 1.4_


- [ ] 8. Prompt registry — wire DBRegistry and improve error messages
  - [~] 8.1 Wire `DBRegistry` in DI container
    - In `internal/di/container.go`, replace `prompt.NewHardcodedRegistry()` with `prompt.NewDBRegistry(promptStore)`
    - Inject the registry into all services that accept `domain.Registry`
    - _Requirements: 6.1_

  - [~] 8.2 Fix `PromptStore.Active` to return `domain.ErrNotFound`
    - Ensure `internal/prompt/db_registry.go` returns `domain.ErrNotFound` (not nil or empty struct) when no active row exists for a stage
    - _Requirements: 6.2_

  - [~] 8.3 Improve `Render` error message to include stage name
    - Update `Render` in `internal/prompt/registry.go` to include template stage name in template execution errors
    - _Requirements: 6.5_

  - [ ]* 8.4 Write property test for prompt render error includes stage name
    - **Property 12: Prompt render error includes stage name**
    - **Validates: Requirements 6.5**
    - _File: `internal/prompt/registry_test.go`_


- [ ] 9. LLM client — key provider and token budget
  - [~] 9.1 Define `domain.KeyProvider` interface and `EnvKeyProvider` implementation
    - Add `KeyProvider` interface with `APIKey`, `Refresh`, and `Rotate` methods to `internal/domain/interfaces.go`
    - Implement `EnvKeyProvider` in `internal/llm/provider.go` that reads the env var at call time (not cached)
    - Update `LitellmClient` to accept a `KeyProvider` instead of a static key string
    - _Requirements: 14.1, 14.5_

  - [~] 9.2 Implement `History.TrimToBudget` and context overflow retry logic
    - Ensure `History.TrimToBudget(limit int)` preserves system message and most-recent messages while keeping total token count ≤ limit
    - In extraction service callers: on `ErrContextOverflow`, call `TrimToBudget(limit/2)` and retry once; return error if second call also overflows
    - _Requirements: 5.3, 5.5_

  - [ ]* 9.3 Write property test for TrimToBudget respects budget
    - **Property 10: Token budget respected before LLM call**
    - **Validates: Requirements 5.3**
    - _File: `internal/llm/history_test.go`_

  - [ ]* 9.4 Write property test for context overflow retry halves budget
    - **Property 11: Context overflow retry halves budget**
    - **Validates: Requirements 5.5**
    - _File: `internal/llm/history_test.go`_

  - [~] 9.5 Implement `DiscoverBoundary` startup call in daemon
    - On daemon startup, call `llm.DiscoverBoundary` for each model tier; on error log warn and set `safe_token_limit = 4096`
    - Skip `DiscoverBoundary` if `safe_token_limit > 0` and `updated_at` is within 24 hours
    - _Requirements: 5.1, 5.2_


- [ ] 10. Extraction service — complete L2/L3/L4 stages
  - [~] 10.1 Implement `ProcessEntity` in `internal/app/extract/service.go`
    - Fetch `FileVersion`, render `level_2_entity` prompt, call LLM, parse `entities[]`, persist each as `domain.Entity` via `EntityStore.Create`
    - Enqueue `StageLevel3Feature` job regardless of whether entity list is empty
    - _Requirements: 2.2_

  - [ ]* 10.2 Write property test for ProcessEntity fan-out count
    - **Property 5: Extraction pipeline fan-out (Level 2)**
    - **Validates: Requirements 2.2**
    - _File: `internal/app/extract/service_test.go`_

  - [~] 10.3 Implement `ProcessFeature` in `internal/app/extract/service.go`
    - Query features linked to file version via `file_features`; if < 2 features, mark job completed without LLM call
    - If ≥ 2 features, render `level_3_feature` prompt for each ordered pair, call LLM, persist results as `FeatureInteraction`
    - _Requirements: 2.3_

  - [ ]* 10.4 Write property test for ProcessFeature pair threshold
    - **Property 6: Feature pair threshold for Level 3**
    - **Validates: Requirements 2.3**
    - _File: `internal/app/extract/service_test.go`_

  - [~] 10.5 Implement `ProcessEdgeCase` in `internal/app/extract/service.go`
    - Render `level_4_edge_case` prompt, call LLM, parse edge-case annotations, persist each as `StyleAnomaly` with `code_style_id = 0`
    - _Requirements: 2.4_

  - [~] 10.6 Fix `ProcessOverview` fan-out and language derivation
    - Validate `ProcessOverview` persists exactly N features and N `file_features` rows, enqueues exactly one `StageLevel2Entity` job
    - Replace hardcoded `"go"` language string with `domain.LanguageFromPath(file.Path)` when calling scrubber
    - _Requirements: 2.1, 7.6_

  - [ ]* 10.7 Write property test for ProcessOverview fan-out count
    - **Property 4: Extraction pipeline fan-out (Level 1)**
    - **Validates: Requirements 2.1**
    - _File: `internal/app/extract/service_test.go`_


- [ ] 11. Style evaluator — baseline discovery
  - [~] 11.1 Implement baseline discovery in `internal/app/style/evaluator.go`
    - On `Evaluate` entry, call `StyleStore.CountByProject`; if < `MIN_STYLE_SAMPLES`, query most-connected file versions, call LLM Tier 2, parse style rules, persist via `StyleStore.Create` in same transaction
    - Use `LLMConfigStore.GetByTier(ctx, projectID, domain.ModelTier1)` for model lookup; return `domain.ErrNotFound` if no config exists
    - _Requirements: 3.1, 3.2, 3.3, 3.4_

  - [ ]* 11.2 Write property test for style baseline threshold
    - **Property 7: Style baseline threshold**
    - **Validates: Requirements 3.1, 3.3**
    - _File: `internal/app/style/evaluator_test.go`_


- [ ] 12. Graph synthesizer — intersection and macro synthesis
  - [~] 12.1 Implement `GraphStore.GetUnprocessedFeaturePairs` and `CreateMacroPipeline`
    - Add `GetUnprocessedFeaturePairs(ctx, projectID)` SQL query returning pairs where `f1.id < f2.id`, same project, no existing interaction in either direction
    - Add `GraphStore.CreateMacroPipeline` to persist `MacroPipeline` records
    - _Requirements: 4.4_

  - [ ]* 12.2 Write property test for unprocessed feature pairs query
    - **Property 8: Unprocessed feature pairs query**
    - **Validates: Requirements 4.4**
    - _File: `internal/store/graph_store_test.go`_

  - [~] 12.3 Implement `EnqueuePendingIntersections` in `internal/app/graph/synthesizer.go`
    - Call `GraphStore.GetUnprocessedFeaturePairs`; for each pair enqueue `StageIntersectionSynthesis` job with `TargetID = A.ID` and `B.ID` encoded in `job.Payload`
    - When zero pairs remain, call `GraphStore.DiscoverMacroPipelines` and insert `MacroPipeline` records and `StageMacroSynthesis` jobs
    - _Requirements: 4.1, 4.3_

  - [ ]* 12.4 Write property test for idle daemon enqueues intersection jobs
    - **Property 9: Idle daemon enqueues intersection jobs**
    - **Validates: Requirements 4.1**
    - _File: `internal/app/graph/synthesizer_test.go`_

  - [~] 12.5 Implement `Synthesize` (intersection) and macro-synthesis handler
    - `Synthesize(ctx, job)`: fetch Feature A by `job.TargetID`, Feature B by ID from `job.Payload`, render `intersection_synthesis` prompt, call LLM, persist as `FeatureInteraction`
    - Add macro-synthesis handler that fetches `MacroPipeline`, renders summary prompt, updates `MacroPipeline.Description`
    - _Requirements: 4.2, 4.5_


- [ ] 13. Service dispatcher and worker daemon wiring
  - [~] 13.1 Wire all stage cases in `internal/worker/dispatcher.go`
    - Replace all "not yet fully implemented" stubs with real handler calls for all 7 stages
    - Add `StageMacroSynthesis` case calling macro-synthesis handler
    - _Requirements: 2.5_

  - [~] 13.2 Wire key rotation logic in `internal/worker/daemon.go`
    - On `ErrAuthRotation`: set `rotationInProgress` flag (atomic), drain in-flight jobs, call `KeyProvider.Rotate(KeyProvider.Refresh())`, re-queue triggering job, clear flag
    - After 3 consecutive rotation failures, log structured error and halt polling loop permanently
    - _Requirements: 14.2, 14.3, 14.4_

  - [~] 13.3 Wire idle-state intersection enqueueing and metrics in daemon
    - When `ClaimBatch` returns zero jobs, call `graph.EnqueuePendingIntersections`
    - After each `ClaimBatch` and `Complete`/`Fail`, call `metrics.SetQueueDepth`; on job state transitions call `metrics.IncJobsTotal`
    - Replace `log.Printf`/`fmt.Printf` calls in daemon with structured logger
    - _Requirements: 4.1, 10.2, 11.2, 11.4_

  - [~] 13.4 Update DI container to inject logger, metrics, and new services
    - Construct `logger`, `metrics.Registry`, `TikTokenCounter` (with `HeuristicCounter` fallback), `prompt.DBRegistry`, `EnvKeyProvider`
    - Inject into all consumers: daemon, LLM client, extraction service, style evaluator, graph synthesizer, API server
    - _Requirements: 10.5, 11.6, 16.3, 16.4_


- [~] 14. Checkpoint — Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 15. Store layer — unique constraint error handling and new store methods
  - [~] 15.1 Handle unique constraint violations in `EntityStore` and `FeatureStore`
    - In `internal/store/entity_store.go`: detect SQLite UNIQUE constraint error on `Create` and return `domain.ErrConflict`
    - In `internal/store/feature_store.go`: same handling for `uq_features_project_name`
    - _Requirements: 13.4, 13.5_

  - [ ]* 15.2 Write unit tests for constraint error handling
    - Test that inserting a duplicate entity returns `domain.ErrConflict`
    - Test that inserting a duplicate feature name within the same project returns `domain.ErrConflict`
    - _Requirements: 13.4, 13.5_

  - [~] 15.3 Add `StyleStore.CountByProject` and LLM metrics instrumentation to `LiteLLM_Client`
    - Add `CountByProject(ctx, projectID) (int, error)` to `internal/store/style_store.go`
    - In `internal/llm/client.go`, call `metrics.IncLLMRequests(modelName, "success"/"error")` once per outgoing call (after all internal retries)
    - _Requirements: 3.1, 11.3_


- [ ] 16. REST API — new endpoints and error middleware
  - [~] 16.1 Add error-handling middleware and `errorResponse` helper
    - Add `errorResponse` helper to `internal/api/server.go` mapping `domain.ErrNotFound` → 404, others → 500
    - _Requirements: 9.12, 9.13_

  - [ ]* 16.2 Write property test for ErrNotFound → HTTP 404
    - **Property 21: API ErrNotFound consistently maps to HTTP 404**
    - **Validates: Requirements 9.12**
    - _File: `internal/api/handlers_test.go`_

  - [~] 16.3 Implement entity and graph endpoints
    - `GET /api/entities` → query all entities for active project, return JSON array (empty array on zero results)
    - `GET /api/entities/:id` → return entity + linked feature IDs; 404 if not found
    - `GET /api/graph` → return all `FeatureInteraction` records with fields `id`, `from_feature_id`, `to_feature_id`, `description`, `created_at`
    - _Requirements: 9.1, 9.2, 9.3_

  - [~] 16.4 Implement macro, DLQ, and retry endpoints
    - `GET /api/macros` → return all `MacroPipeline` records
    - `GET /api/dlq` → return all DLQ items joined with source job fields
    - `POST /api/dlq/:id/retry` → atomically move item back to `job_queue` with `status=pending, retry_count=0`; 404 if not found
    - _Requirements: 9.4, 9.5, 9.6_

  - [~] 16.5 Implement prompt and style endpoints
    - `GET /api/prompts` → return all templates grouped by stage
    - `PUT /api/prompts/:id` → update template text, increment version; 400 if template missing/empty
    - `GET /api/styles` → return all `CodeStyle` records
    - `GET /api/styles/:id/anomalies` → return style anomalies; 404 if style not found
    - _Requirements: 9.7, 9.8, 9.9, 9.10_

  - [~] 16.6 Add `/metrics` endpoint
    - Register `GET /metrics` route delegating to `metrics.Handler()` (promhttp)
    - Ensure `Content-Type: text/plain; version=0.0.4; charset=utf-8`
    - _Requirements: 9.11, 11.5_


- [ ] 17. Cobra CLI — complete verb coverage
  - [~] 17.1 Implement `prowiki get` command in `internal/cli/get.go`
    - Support resources: `files`, `features`, `entities`, `jobs`, `prompts`, `projects`
    - Respect `-o` format flag (`table`/`json`/`yaml`; exit non-zero with error on unknown format) and `-l` limit flag
    - Support `-w` watch mode: re-poll every 2 seconds, clear terminal between updates, exit on SIGINT/SIGTERM
    - _Requirements: 8.1, 8.6, 8.7, 8.8_

  - [~] 17.2 Implement `prowiki describe` command in `internal/cli/describe.go`
    - Return full record for given numeric ID including all scalar and timestamp fields
    - _Requirements: 8.2_

  - [~] 17.3 Implement `prowiki retry` command in `internal/cli/retry.go`
    - Atomically delete DLQ row and reset `job_queue` row to `pending, retry_count=0` in one transaction
    - Exit non-zero with stderr message if job ID not in DLQ
    - _Requirements: 8.3_

  - [~] 17.4 Implement `prowiki run` command in `internal/cli/run.go`
    - Run ingestion, then start daemon in foreground; emit structured JSON logs to stdout by default
    - On `--progress`: emit human-readable poll-cycle counts to stderr
    - Exit non-zero if ingestion errors before daemon starts
    - _Requirements: 8.4, 8.5, 10.2, 10.3_

  - [~] 17.5 Update `prowiki serve` command and root command flag registration
    - `prowiki serve` starts API server + daemon in two goroutines, blocks until SIGINT/SIGTERM
    - Register all global flags with Viper using `PROWIKI_` prefix
    - _Requirements: 8.9, 8.10_

  - [~] 17.6 Update `prowiki init` command to persist `fs_location`
    - On `prowiki init`, insert `projects` row with `name = filepath.Base(dir)` and `fs_location = dir`
    - _Requirements: 8.11_


- [ ] 18. Web UI — three-pane layout and admin panel
  - [~] 18.1 Refactor `web/index.html` and `web/app.js` to three-pane layout
    - Left navigation sidebar, center content pane, right context/detail pane using existing CSS custom properties
    - Remove any external font URL references from `web/style.css`; use system font stack or local assets
    - _Requirements: 12.1, 12.8_

  - [~] 18.2 Implement file detail, feature map, and code explorer views
    - On file click: fetch `GET /api/files/:id`, render summary, features, entities in center pane without reload
    - "Feature Map" view: fetch `GET /api/graph`, render as interactive directed graph using Cytoscape.js
    - "Code Explorer" view: display file content as preformatted text with inline `StyleAnomaly` hover indicators
    - _Requirements: 12.2, 12.3, 12.4_

  - [~] 18.3 Implement Admin panel (DLQ, Prompts, Projects)
    - DLQ list with Retry button per row (`POST /api/dlq/:id/retry`)
    - Prompts editor with textarea per stage and Save button (`PUT /api/prompts/:id`); show "Saved" confirmation or error message
    - Project registration form (`POST /api/projects`)
    - _Requirements: 12.5_

  - [~] 18.4 Implement adaptive polling with CSS transition for job counts
    - CSS 300ms transition on job status count updates (not synchronous DOM text replacement)
    - Double polling interval (2s → 30s cap) when consecutive polls return identical counts; reset to 2s on any change
    - _Requirements: 12.6, 12.7_


- [ ] 19. Entry-point consolidation
  - [~] 19.1 Consolidate to single entry point at `cmd/prowiki/main.go`
    - Ensure `cmd/prowiki/main.go` contains only `func main() { cli.Execute() }`
    - Convert root `main.go` to a non-`main` package (add `//go:build ignore` build constraint) or delete it
    - Verify Makefile `build` target references `./cmd/prowiki` only
    - _Requirements: 17.1, 17.2, 17.3, 17.4_

- [~] 20. Final checkpoint — Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.


## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP delivery
- All property tests use `github.com/flyingmutant/rapid` with a minimum of 100 iterations; scrubber property tests use 500 iterations
- All property and unit tests use mocked LLM clients and in-memory SQLite (`:memory:`) — no network calls
- Each property test comment references the design property number and requirements clause
- Existing migration files MUST NOT be modified; schema changes go into new numbered migration files
- The `internal/versioning` package is deleted entirely; ensure no stale imports remain after removal
- The `domain.Counter` interface signature change (adding `CountMessages`) is a breaking change — all implementations must be updated before the DI container compiles

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1", "1.2", "1.3"] },
    { "id": 1, "tasks": ["1.4", "2.1", "2.3", "3.1", "4.1", "5.1"] },
    { "id": 2, "tasks": ["2.2", "2.4", "3.2", "4.2", "4.3", "4.4", "5.2", "5.3", "5.4"] },
    { "id": 3, "tasks": ["3.3", "3.4", "7.1", "8.1", "8.2", "8.3", "9.1"] },
    { "id": 4, "tasks": ["7.2", "8.4", "9.2", "9.5", "10.6", "15.3"] },
    { "id": 5, "tasks": ["7.3", "7.4", "9.3", "9.4", "10.1", "10.5", "11.1", "12.1", "15.1"] },
    { "id": 6, "tasks": ["7.5", "10.2", "10.3", "10.7", "11.2", "12.2", "15.2", "7.6"] },
    { "id": 7, "tasks": ["10.4", "12.3", "12.5", "13.1", "13.2"] },
    { "id": 8, "tasks": ["12.4", "13.3", "13.4"] },
    { "id": 9, "tasks": ["16.1"] },
    { "id": 10, "tasks": ["16.2", "16.3", "16.4", "16.5", "16.6"] },
    { "id": 11, "tasks": ["17.1", "17.2", "17.3", "17.4", "17.5", "17.6", "18.1"] },
    { "id": 12, "tasks": ["18.2", "18.3", "18.4", "19.1"] }
  ]
}
```
