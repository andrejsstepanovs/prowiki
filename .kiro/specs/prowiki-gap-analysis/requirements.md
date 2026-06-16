# Requirements Document

## Introduction

ProWiki is a persistent background daemon that continuously scans, parses, and documents a Go codebase using LLMs. The system uses an embedded SQLite database as its queue and knowledge store, a Cobra/Viper CLI, and a go-litellm client for LLM connectivity. A gap analysis of the current codebase against the documented architecture identifies a set of incomplete, missing, or architecturally divergent components that must be built or corrected.

This document captures requirements for every gap — grouped by subsystem — so that each can be independently implemented and tested.

---

## Glossary

- **Daemon**: The background polling worker (`internal/worker/daemon.go`) that claims jobs from the queue and dispatches them to handlers.
- **DI_Container**: The dependency-injection container (`internal/di/container.go`) that wires all services together.
- **DLQ**: Dead Letter Queue — storage for jobs that have exhausted their retry budget.
- **Extraction_Service**: The service at `internal/app/extract/service.go` responsible for driving 4-level LLM extraction.
- **Graph_Synthesizer**: The service at `internal/app/graph/synthesizer.go` responsible for discovering feature interactions.
- **Heuristic_Parser**: The AST parser at `internal/ast/parser.go` that computes structural hashes via comment-stripped line normalization.
- **History**: The token-aware conversation history manager at `internal/llm/history.go`.
- **Ingest_Service**: The canonical ingestion service responsible for file versioning, AST hashing, and job enqueuing.
- **Job**: A unit of work in the `job_queue` table, carrying a `stage`, `target_id`, `target_type`, and `priority`.
- **LiteLLM_Client**: The go-litellm-backed LLM client at `internal/llm/client.go`.
- **Metrics_Endpoint**: A Prometheus-compatible `/metrics` HTTP handler.
- **Prompt_Registry**: The interface for retrieving and rendering versioned prompt templates, implemented as a DB-backed registry.
- **Scrubber**: The regex-based PII/secrets redactor at `internal/scrub/scrubber.go`.
- **Service_Dispatcher**: The job-stage router at `internal/worker/dispatcher.go`.
- **Style_Evaluator**: The code-style anomaly detector at `internal/app/style/evaluator.go`.
- **Tokenizer**: The token-counting utility at `internal/tokenizer/counter.go`.
- **Walker**: The filesystem scanner at `internal/scanner/walker.go`.

---

## Requirements

### Requirement 1: Unify the Dual Ingestion Services

**User Story:** As a developer, I want a single, canonical ingestion path from file discovery to job enqueuing, so that file versioning logic is not duplicated across two packages.

#### Acceptance Criteria

1. THE System SHALL maintain exactly one ingestion service responsible for reading files, computing SHA-256 content hashes, computing AST structural hashes, creating or updating `File` and `FileVersion` records, and enqueuing `StageLevel1Overview` jobs. The canonical location SHALL be `internal/app/ingest`.
2. WHEN a file's AST structural hash is identical to the latest stored version's `ast_hash`, THE Ingest_Service SHALL insert a new `FileVersion` record, clone all `file_features` junction rows from the previous `file_version_id` to the new one within the same transaction, mark the new version as latest, and mark the old version as non-latest — without enqueuing any `StageLevel1Overview` job.
3. WHEN a file's AST structural hash differs from the latest stored version's `ast_hash`, THE Ingest_Service SHALL insert a new `FileVersion`, mark it as latest within the same transaction, reset any `completed` `job_queue` rows whose `target_id` matches the old `file_version_id` back to `pending` with priority incremented by 10 (cascading invalidation), and enqueue a new `StageLevel1Overview` job targeting the new `FileVersion.ID`.
4. THE `internal/versioning` package SHALL be removed; the DI_Container SHALL wire the `internal/app/ingest` service as the sole ingestion implementation.
5. IF the file extension is `.go`, THEN THE Ingest_Service SHALL pass `domain.Language("go")` to the Heuristic_Parser. IF the extension is `.py`, THEN it SHALL pass `domain.Language("python")`. IF the extension is `.js`, THEN it SHALL pass `domain.Language("javascript")`. IF the extension is `.ts`, THEN it SHALL pass `domain.Language("typescript")`. Otherwise it SHALL pass `domain.Language("")`.

---

### Requirement 2: Complete the 4-Level LLM Extraction Pipeline

**User Story:** As a developer, I want all four extraction stages to be fully wired, so that the daemon can progress from a raw file version to entities, feature interactions, and edge cases without manual intervention.

#### Acceptance Criteria

1. WHEN a `StageLevel1Overview` job is dispatched, THE Extraction_Service SHALL fetch the `FileVersion` by `job.TargetID`, scrub its content via the Scrubber, render the `level_1_overview` prompt, call the LLM, parse the response into `summary` and `features[]`, persist the summary via `FileVersionStore.UpdateSummary`, persist each feature via `FeatureStore.Create` and `FeatureStore.AddToFileVersion`, and enqueue a `StageLevel2Entity` job targeting the same `FileVersion.ID`. IF the LLM call or JSON parse fails, THE Extraction_Service SHALL return the error without marking the job completed so the queue's retry logic applies.
2. WHEN a `StageLevel2Entity` job is dispatched, THE Extraction_Service SHALL fetch the `FileVersion` by `job.TargetID`, render the `level_2_entity` prompt with the file content, call the LLM, parse the response into `entities[]`, persist each as a `domain.Entity` record via `EntityStore.Create`, and enqueue a `StageLevel3Feature` job targeting the same `FileVersion.ID`. IF the parsed entity list is empty, THE Extraction_Service SHALL still enqueue `StageLevel3Feature` and mark the job completed.
3. WHEN a `StageLevel3Feature` job is dispatched, THE Extraction_Service SHALL query all `Feature` records linked to `job.TargetID` via the `file_features` junction. IF two or more features exist, THE Extraction_Service SHALL render the `level_3_feature` prompt for each ordered pair, call the LLM, and persist each result as a `FeatureInteraction` via `Graph_Synthesizer.Synthesize`. IF fewer than two features exist, THE Extraction_Service SHALL mark the job completed without calling the LLM.
4. WHEN a `StageLevel4EdgeCase` job is dispatched, THE Extraction_Service SHALL render the `level_4_edge_case` prompt with the file content, call the LLM, parse the response into a list of edge-case annotations (each having at minimum a `description` string field), and persist each annotation as a `StyleAnomaly` record with `code_style_id = 0` and the annotation description as `rationale`.
5. WHEN the Service_Dispatcher receives a job with `Stage` equal to any of `StageLevel1Overview`, `StageLevel2Entity`, `StageLevel3Feature`, `StageLevel4EdgeCase`, `StageStyleEvaluation`, or `StageIntersectionSynthesis`, THE Service_Dispatcher SHALL invoke the appropriate handler and return its result without returning a "not yet fully implemented" error.
6. THE DI_Container SHALL construct the Extraction_Service, Style_Evaluator, and Graph_Synthesizer and pass them to the Service_Dispatcher constructor so the worker Daemon uses the fully-wired dispatcher.

---

### Requirement 3: Code Style Baseline Discovery

**User Story:** As a developer, I want the daemon to automatically discover code-style rules from a project's codebase, so that the Style_Evaluator has a baseline to compare against without requiring manual rule entry.

#### Acceptance Criteria

1. WHEN a `StageStyleEvaluation` job is dispatched and fewer than `MIN_STYLE_SAMPLES` (default 3) `CodeStyle` records exist for the project, THE Style_Evaluator SHALL first execute baseline discovery: query the `MIN_STYLE_SAMPLES` most-connected file versions (those with the highest count of `file_features` rows), call the LLM Tier 2 model with a baseline-extraction prompt, parse the response into a list of style rules, and persist each as a `CodeStyle` record associated with the project.
2. WHEN baseline discovery produces one or more new `CodeStyle` records, THE Style_Evaluator SHALL persist them via `StyleStore.Create` within the same transaction as the overall style evaluation job, so a crash does not leave partial baseline data.
3. WHEN a `StageStyleEvaluation` job is dispatched and at least `MIN_STYLE_SAMPLES` `CodeStyle` records already exist, THE Style_Evaluator SHALL retrieve all `CodeStyle` rules for the project via `StyleStore.ListByProject`, then for each rule evaluate the file version's content against that rule, and persist each result as either a `FileStyle` (compliant) or a `StyleAnomaly` (non-compliant with a rationale string).
4. WHEN the Style_Evaluator resolves the LLM model for evaluation, IT SHALL query `LLMConfigStore.GetByTier(ctx, project_id, domain.ModelTier1)` and use the returned model name in the completion request. IF no `LLMConfig` row exists for `ModelTier1`, THE Style_Evaluator SHALL return `domain.ErrNotFound` and fail the job so it is retried after configuration is supplied.

---

### Requirement 4: Graph Synthesis and Agentic Idle State

**User Story:** As a developer, I want the daemon to synthesize macro-pipeline graphs during idle periods, so that the knowledge base automatically surfaces high-level architectural patterns.

#### Acceptance Criteria

1. WHEN the Daemon's `ClaimBatch` returns zero jobs, THE Daemon SHALL call `Graph_Synthesizer.EnqueuePendingIntersections(ctx, projectID)` which queries `GraphStore.GetUnprocessedFeaturePairs` and, for each unprocessed pair `(A, B)`, inserts one `StageIntersectionSynthesis` job with `TargetID = A.ID` and `ProjectID` carrying `B.ID` — then resumes normal polling.
2. WHEN a `StageIntersectionSynthesis` job is dispatched, THE Service_Dispatcher SHALL call `Graph_Synthesizer.Synthesize(ctx, job)` which fetches `Feature A` by `job.TargetID` and `Feature B` by a second ID encoded in the job payload, renders the `intersection_synthesis` prompt for the pair, calls the LLM, and persists the result as a `FeatureInteraction` row via `GraphStore.CreateInteraction`.
3. WHEN `GraphStore.GetUnprocessedFeaturePairs` returns zero rows (all pairs have interactions), THE Graph_Synthesizer SHALL call `GraphStore.DiscoverMacroPipelines(ctx, projectID)` and for each returned chain that does not already have a matching `MacroPipeline` record (matched by `node_sequence`), insert one `MacroPipeline` record and one `macro_synthesis` stage job targeting the new `MacroPipeline.ID`.
4. THE GraphStore SHALL expose `GetUnprocessedFeaturePairs(ctx, projectID) ([]FeaturePair, error)` implemented as a SQL query that returns all `(f1.id, f2.id)` pairs where `f1.id < f2.id`, both features share the same `project_id`, and no `feature_interactions` row exists with `from_feature_id = f1.id AND to_feature_id = f2.id` or `from_feature_id = f2.id AND to_feature_id = f1.id`.
5. WHEN a `macro_synthesis` job is dispatched, THE Service_Dispatcher SHALL call a macro-synthesis handler that fetches the `MacroPipeline` by `job.TargetID`, renders a summary prompt, and updates `MacroPipeline.Description` with the LLM response.

---

### Requirement 5: Dynamic Context Boundary Discovery and Token Routing

**User Story:** As a developer, I want each LLM call to be aware of the model's token limit and automatically trim context when needed, so that no request fails due to context overflow after the system has been running for a while.

#### Acceptance Criteria

1. WHEN the Daemon starts, THE System SHALL call `llm.DiscoverBoundary(ctx, model, endpoint, apiKey)` for each model tier registered in `LLMConfig`. IF `DiscoverBoundary` returns an error, THE Daemon SHALL log a warning, set `safe_token_limit` to a conservative default of 4096, persist it, and continue startup rather than aborting.
2. IF a `LLMConfig` row for a model tier already has `safe_token_limit > 0` and its `updated_at` timestamp is within the configured refresh interval (default 24 hours), THEN THE Daemon SHALL skip `DiscoverBoundary` for that tier.
3. WHEN assembling a completion request, THE Extraction_Service SHALL retrieve `safe_token_limit` from `LLMConfig` for the job's model tier. IF no `LLMConfig` row exists, IT SHALL use 4096 as the fallback limit. IT SHALL then call `History.TrimToBudget(limit)` on the assembled message list before passing it to the LiteLLM_Client.
4. WHEN the LiteLLM_Client builds a completion request, IT SHALL set the `Model` field from the `LLMConfig.ModelName` looked up by tier, not from a string literal embedded in service code.
5. IF a completion call returns `domain.ErrContextOverflow`, THEN THE caller SHALL call `History.TrimToBudget(limit / 2)` to halve the context and retry the completion exactly once. IF the retry also returns `domain.ErrContextOverflow`, THE caller SHALL return the error to the job dispatcher without further retries.

---

### Requirement 6: Prompt Registry Wiring and Version Control

**User Story:** As a developer, I want all prompt templates to be loaded from the database registry, so that prompts can be updated at runtime without a code deployment.

#### Acceptance Criteria

1. THE DI_Container SHALL construct `prompt.NewDBRegistry(promptStore)` and pass it as the `domain.Registry` to all services that accept a registry interface, replacing any usage of `prompt.NewHardcodedRegistry()`.
2. WHEN `PromptStore.Active(ctx, stage)` finds no row with `is_active = true` for the requested stage, THE implementation SHALL return `domain.ErrNotFound` (not a nil pointer or empty struct), so callers can detect the absence and apply a fallback.
3. THE `000002_seed_prompts.up.sql` migration SHALL contain `INSERT OR IGNORE` statements for all six stage identifiers: `level_1_overview`, `level_2_entity`, `level_3_feature`, `level_4_edge_case`, `style_evaluation`, and `intersection_synthesis`.
4. THE `000002_seed_prompts.up.sql` migration SHALL include a non-empty prompt template for the `level_4_edge_case` stage (the stage is currently missing from the migration file).
5. WHEN `Prompt_Registry.Render(tmpl, vars)` is called and the Go `text/template` execution fails (e.g., a referenced variable is absent from `vars`), THE method SHALL return an error whose message includes both the template stage name and the underlying template error text, without panicking.

---

### Requirement 7: PII/Secrets Scrubbing Completeness

**User Story:** As a developer, I want the scrubber to catch a broad set of secret patterns before code is sent to the LLM, so that no sensitive credentials leak to external API endpoints.

#### Acceptance Criteria

1. THE Scrubber SHALL redact all substrings matching `eyJ[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]*` (JWT tokens) by replacing them with the token `[REDACTED_SECRET]`.
2. THE Scrubber SHALL redact all content between and including `-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----` and the corresponding `-----END ... PRIVATE KEY-----` lines by replacing the entire block with `[REDACTED_SECRET]`.
3. THE Scrubber SHALL redact all substrings matching `gh[pousr]_[A-Za-z0-9]{36}` (GitHub personal access tokens) by replacing them with `[REDACTED_SECRET]`.
4. THE Scrubber SHALL redact all occurrences of `[0-9a-fA-F]{32,}` that appear as the right-hand side of a key-value assignment (e.g., `key = "..."`, `KEY=...`, `"key": "..."`) by replacing only the hex value portion with `[REDACTED_SECRET]`.
5. WHEN the Scrubber finishes processing a string, IT SHALL return the redacted string and an integer count of the number of distinct non-overlapping replacement sites (each replacement of `[REDACTED_SECRET]` counts as one site regardless of the pattern that matched). The caller is responsible for deciding whether to log a warning based on the returned count.
6. WHEN the Extraction_Service calls the Scrubber, IT SHALL derive the `domain.Language` value from the file path extension of the `File` record associated with the `FileVersion` being processed rather than passing the hardcoded string `"go"`.

---

### Requirement 8: Cobra CLI — Complete Verb Coverage

**User Story:** As a developer, I want a full set of CLI verbs, so that I can manage projects, inspect the knowledge base, and control the daemon entirely from the command line.

#### Acceptance Criteria

1. THE CLI SHALL implement `prowiki get <resource>` where `<resource>` is one of `files`, `features`, `entities`, `jobs`, `prompts`, or `projects`. The command SHALL return rows from the corresponding store and render them in the format specified by the `-o` flag (default `table`).
2. THE CLI SHALL implement `prowiki describe <resource> <id>` returning the full record for the given numeric ID, including all scalar fields: `id`, name or path, all descriptive text fields, all foreign-key IDs, status (where present), and `created_at`/`updated_at` timestamps.
3. WHEN `prowiki retry <job-id>` is executed and the job ID exists in the `dead_letter_queue`, THE CLI SHALL atomically delete the DLQ row and reset the corresponding `job_queue` row's status to `pending` and `retry_count` to `0` in the same transaction. IF the job ID does not exist in the DLQ, THE CLI SHALL exit with a non-zero status and print an error message to stderr.
4. THE CLI SHALL implement `prowiki run [--dir <path>] [--workers <n>] [--progress]` that runs ingestion then starts the Daemon in the foreground of the same process, emitting one structured JSON object per log line to stdout by default.
5. IF `prowiki run` ingestion returns an error before the Daemon starts, THEN THE CLI SHALL print the error to stderr and exit with a non-zero status without starting the Daemon.
6. THE CLI SHALL support a global `-o <format>` flag accepting `table`, `json`, or `yaml`. IF an unrecognized format is supplied, THE CLI SHALL exit with a non-zero status and print `"unsupported output format: <value>"` to stderr.
7. THE CLI SHALL support a global `-l <n>` flag where `n` is a positive integer (`n ≥ 1`) that limits the maximum number of rows returned by any `get` command.
8. THE CLI SHALL support a global `-w` flag on `get` commands that re-polls the data source every 2 seconds, clears the terminal between updates, and exits when the process receives SIGINT or SIGTERM.
9. THE `prowiki serve [--port <n>]` command SHALL start the API HTTP server and the Daemon worker loop in the same process using two goroutines, and SHALL block until SIGINT or SIGTERM is received.
10. THE root command (`prowiki`) SHALL register all flags with Viper using the `PROWIKI_` environment variable prefix so that, for example, `PROWIKI_DIR=/tmp/repo` overrides the `--dir` flag default.
11. IF `prowiki init` is run in a directory that does not have an existing project record in the local `.prowiki.db`, THEN THE `init` command SHALL create the database, run migrations, and insert a `projects` row with `name = filepath.Base(dir)` and `fs_location = dir`.

---

### Requirement 9: REST API — Complete Knowledge and Admin Surface

**User Story:** As a developer, I want a complete REST API covering all knowledge and admin resources, so that the Web UI and external tools can query and manage the system without direct database access.

#### Acceptance Criteria

1. WHEN a `GET /api/entities` request is received, THE API_Server SHALL query all `Entity` records for the active project and return a JSON array. WHEN no entities exist, IT SHALL return an empty array with HTTP 200.
2. WHEN a `GET /api/entities/:id` request is received, THE API_Server SHALL return the matching `Entity` record along with the IDs of all `Feature` records linked to the same project. IF the entity ID does not exist, IT SHALL return HTTP 404.
3. WHEN a `GET /api/graph` request is received, THE API_Server SHALL return all `FeatureInteraction` records for the active project as a JSON array of objects with fields `id`, `from_feature_id`, `to_feature_id`, `description`, and `created_at`.
4. WHEN a `GET /api/macros` request is received, THE API_Server SHALL return all `MacroPipeline` records for the active project.
5. WHEN a `GET /api/dlq` request is received, THE API_Server SHALL return all `DeadLetterItem` records joined with their source `job_queue` row (including `stage`, `target_id`, `retry_count`).
6. WHEN a `POST /api/dlq/:id/retry` request is received and the DLQ item exists, THE API_Server SHALL atomically move the item back to `job_queue` with `status = pending` and `retry_count = 0` and delete the DLQ row, returning HTTP 200. IF the DLQ item does not exist, IT SHALL return HTTP 404.
7. WHEN a `GET /api/prompts` request is received, THE API_Server SHALL return all `PromptTemplate` records grouped by `stage` as a JSON object keyed by stage name.
8. WHEN a `PUT /api/prompts/:id` request is received with a JSON body containing a non-empty `"template"` string, THE API_Server SHALL update the template text and increment `version` by 1 and return HTTP 200 with the updated record. IF `"template"` is absent or empty, IT SHALL return HTTP 400 with `{"error": "template is required"}`.
9. WHEN a `GET /api/styles` request is received, THE API_Server SHALL return all `CodeStyle` records for the active project.
10. WHEN a `GET /api/styles/:id/anomalies` request is received and the style ID exists, THE API_Server SHALL return all `StyleAnomaly` records for that style. IF the style ID does not exist, IT SHALL return HTTP 404.
11. WHEN a `GET /metrics` request is received, THE API_Server SHALL respond with HTTP 200 and `Content-Type: text/plain; version=0.0.4; charset=utf-8` containing Prometheus text exposition format metrics including `prowiki_jobs_total{stage="<stage>",status="<status>"}`, `prowiki_llm_requests_total{model="<model>",status="<success|error>"}`, and `prowiki_queue_depth{status="<pending|processing|completed|failed|dead_lettered>"}`.
12. WHEN an API handler encounters a `domain.ErrNotFound` error, THE API_Server SHALL respond with HTTP 404 and `{"error": "not found"}`.
13. WHEN an API handler encounters any other internal error, THE API_Server SHALL respond with HTTP 500 and `{"error": "<message>"}` where `<message>` is the error's `.Error()` string, without including Go stack traces.

---

### Requirement 10: Structured Logging

**User Story:** As a developer, I want all log output to use structured JSON fields, so that log aggregation systems can index and query ProWiki logs without custom parsers.

#### Acceptance Criteria

1. THE System SHALL replace all `log.Printf` and `fmt.Printf` calls in `internal/worker`, `internal/queue`, `internal/app/extract`, `internal/app/ingest`, `internal/app/style`, and `internal/app/graph` with a structured logger that emits one JSON object per log line with at minimum the fields `level` (string), `ts` (RFC3339 timestamp with millisecond precision), and `msg` (string).
2. WHEN `prowiki run` is executed without `--progress`, THE Daemon SHALL emit one structured JSON log line to stdout for each job claimed (including `job_id` and `stage` fields), each job completed, and each job failed.
3. WHEN `prowiki run --progress` is executed, THE Daemon SHALL emit a human-readable progress line to stderr once per poll cycle showing the counts of `pending`, `processing`, `completed`, and `failed` jobs.
4. WHEN `PROWIKI_LOG_LEVEL` is set to one of `debug`, `info`, `warn`, or `error`, THE logger SHALL filter log entries to that level and above. IF the value is unrecognized or unset, THE logger SHALL default to `info`.
5. THE structured logger instance SHALL be created in `di.NewContainer` and injected into all components that need it; no component SHALL call a package-level global log function directly.

---

### Requirement 11: Prometheus Metrics Collection

**User Story:** As a developer, I want Prometheus metrics exported at `/metrics`, so that I can monitor job throughput, LLM error rates, and queue health from an external dashboard.

#### Acceptance Criteria

1. THE System SHALL implement a `metrics` package exposing a `Registry` struct that holds: a `CounterVec` named `prowiki_jobs_total` with labels `stage` and `status`; a `CounterVec` named `prowiki_llm_requests_total` with labels `model` and `status`; and a `GaugeVec` named `prowiki_queue_depth` with label `status`.
2. WHEN a job transitions to `completed`, `failed`, or `dead_lettered` state, THE Daemon SHALL call `metrics.IncJobsTotal(job.Stage, newStatus)` to increment `prowiki_jobs_total` with the job's stage string and the new status string.
3. WHEN the LiteLLM_Client receives a final response (after all internal retries are exhausted), IT SHALL call `metrics.IncLLMRequests(modelName, "success")` on HTTP 2xx or `metrics.IncLLMRequests(modelName, "error")` on any other outcome — once per outgoing call invocation, not once per retry attempt.
4. WHEN the Daemon completes a `ClaimBatch` call or a `Complete`/`Fail` call, IT SHALL call `metrics.SetQueueDepth(ctx)` which executes the aggregate status-count query and updates the `prowiki_queue_depth` gauge for each status value.
5. WHEN `GET /metrics` is served, THE API_Server SHALL invoke `promhttp.HandlerFor(metrics.Registry, ...)` and respond with `Content-Type: text/plain; version=0.0.4; charset=utf-8`.
6. THE `metrics.Registry` instance SHALL be constructed in `di.NewContainer` and injected into the Daemon and LiteLLM_Client so all components share one Prometheus registry.

---

### Requirement 12: Web UI — Full Three-Pane Layout

**User Story:** As a developer, I want the web dashboard to display entities, the feature interaction graph, style anomalies, and the DLQ management panel, so that I can understand the codebase and manage the system without using the CLI.

#### Acceptance Criteria

1. THE Web_UI SHALL render a three-column layout with a left navigation sidebar, a center content pane, and a right context/detail pane using the existing CSS custom properties (`--bg-primary`, `--accent-purple`, etc.) and glass-panel component styles already defined in `style.css`.
2. WHEN a user clicks a file entry in the navigation sidebar, THE Web_UI SHALL fetch `GET /api/files/:id` and display the file's `summary`, `features` list, and `entities` list in the center content pane without requiring a full page reload.
3. THE Web_UI SHALL include a "Feature Map" navigation entry that fetches `GET /api/graph` and renders the returned `FeatureInteraction` array as an interactive directed graph using Cytoscape.js, supporting pan, zoom, and node-click navigation to the feature detail view.
4. THE Web_UI SHALL include a "Code Explorer" view that displays raw file content as preformatted text and, for each `StyleAnomaly` associated with the file version, renders an inline indicator at the end of the content block with the anomaly's `rationale` text visible on hover.
5. THE Web_UI SHALL include an "Admin" navigation section with three sub-views: a DLQ list (fetching `GET /api/dlq`) with a Retry button per row that calls `POST /api/dlq/:id/retry`; a Prompts editor (fetching `GET /api/prompts`) with an inline `<textarea>` per stage and a Save button that calls `PUT /api/prompts/:id`; and a Project registration form that calls `POST /api/projects`. WHEN a Save action succeeds, THE UI SHALL display a brief "Saved" confirmation. WHEN it fails, THE UI SHALL display the error message from the response body.
6. IF a job status count (`pending`, `processing`, `completed`, or `failed`) changes between two consecutive polls, THEN THE Web_UI SHALL update the displayed number using a CSS transition of 300ms duration rather than a synchronous DOM text replacement.
7. IF two consecutive polls return identical job status counts, THEN THE Web_UI SHALL double the polling interval (starting from 2 seconds, capping at 30 seconds). WHEN any poll returns a changed count, THE Web_UI SHALL reset the interval to 2 seconds.
8. THE Web_UI SHALL NOT load fonts from external URLs (e.g., Google Fonts CDN). Any font referenced in `style.css` SHALL either use a system font stack or be served as a local asset from the `/web` directory.

---

### Requirement 13: Database Schema — Missing Indexes and Constraints

**User Story:** As a developer, I want the database schema to include all indexes required for efficient queue polling and knowledge queries, so that performance does not degrade as the knowledge base grows.

#### Acceptance Criteria

1. THE existing composite index `idx_jobq_status_priority` on `job_queue(status, priority DESC, id ASC)` SHALL be preserved unchanged in the schema.
2. A new migration SHALL add index `idx_fi_from_to` defined as `CREATE INDEX IF NOT EXISTS idx_fi_from_to ON feature_interactions(from_feature_id ASC, to_feature_id ASC)` to support efficient pair lookups.
3. A new migration SHALL add index `idx_sa_file_version` defined as `CREATE INDEX IF NOT EXISTS idx_sa_file_version ON style_anomalies(file_version_id ASC)` and index `idx_sa_code_style` defined as `CREATE INDEX IF NOT EXISTS idx_sa_code_style ON style_anomalies(code_style_id ASC)`.
4. A new migration SHALL add index `idx_entities_project` defined as `CREATE INDEX IF NOT EXISTS idx_entities_project ON entities(project_id ASC)` and unique constraint `uq_entities_project_name_type` defined as `CREATE UNIQUE INDEX IF NOT EXISTS uq_entities_project_name_type ON entities(project_id, name, type)`. WHEN an `EntityStore.Create` call would violate `uq_entities_project_name_type`, THE store SHALL return an error wrapping `domain.ErrConflict`.
5. A new migration SHALL add unique constraint `uq_features_project_name` defined as `CREATE UNIQUE INDEX IF NOT EXISTS uq_features_project_name ON features(project_id, name)`. WHEN a `FeatureStore.Create` call would violate `uq_features_project_name`, THE store SHALL return an error wrapping `domain.ErrConflict`.
6. Each new index or constraint SHALL be placed in a new numbered migration file (e.g., `000003_indexes.up.sql`). The corresponding `.down.sql` file SHALL drop each index or constraint created in the paired `.up.sql`. Existing migration files SHALL NOT be modified.

---

### Requirement 14: Key Rotation and Auth Suspension

**User Story:** As a developer, I want the LLM key provider to support atomic key rotation so that the daemon can switch API keys without dropping in-flight requests.

#### Acceptance Criteria

1. THE `domain.KeyProvider` interface SHALL include a `Rotate(ctx context.Context, newKey string) error` method. A successful `Rotate` call means all subsequent API requests will use `newKey`; in-flight requests that started before `Rotate` was called are allowed to complete using the old key.
2. WHEN the LiteLLM_Client returns `domain.ErrAuthRotation`, THE Daemon SHALL: stop claiming new jobs (by setting an internal rotation-in-progress flag); wait for all currently dispatched jobs to complete or fail; call `KeyProvider.Rotate` with a fresh key obtained by calling `KeyProvider.Refresh(ctx)`; re-queue the job that surfaced `ErrAuthRotation` back to `pending` status; and clear the rotation-in-progress flag to resume normal polling.
3. WHILE the rotation-in-progress flag is set, THE Daemon's `ClaimBatch` call SHALL NOT be invoked, and already-dispatched jobs SHALL be allowed to run to completion before the new key takes effect.
4. IF `KeyProvider.Rotate` or `KeyProvider.Refresh` returns an error on three consecutive invocations within a single rotation attempt, THEN THE Daemon SHALL cease polling permanently for the lifetime of the process, log a structured message at `error` level with field `reason` containing the last error string, and return from its main loop.
5. WHEN `EnvKeyProvider.Refresh(ctx)` is called, IT SHALL read the current value of the environment variable named by its configured key name at the time of the call (not a cached value from startup), so that updating the environment variable without restarting the process is sufficient to supply the new key.

---

### Requirement 15: AST Parser — Language-Aware Structural Hashing

**User Story:** As a developer, I want the AST parser to apply language-specific stripping rules, so that the structural hash reflects meaningful code changes and ignores comments and blank lines consistently across languages.

#### Acceptance Criteria

1. WHEN `domain.Language("go")` is passed to `Heuristic_Parser.Parse`, THE parser SHALL strip all single-line comments (lines or inline suffixes starting with `//`) and all block comment content between `/*` and `*/` including content spanning multiple lines, before computing the structural hash.
2. WHEN `domain.Language("python")` is passed, THE parser SHALL strip all lines starting with `#` and all content between opening `"""` or `'''` and the matching closing `"""` or `'''` delimiter (including multi-line docstrings), before computing the structural hash.
3. WHEN `domain.Language("javascript")` or `domain.Language("typescript")` is passed, THE parser SHALL apply the same stripping rules as for `"go"`: remove `//` line comments and `/* */` block comments including multi-line content.
4. WHEN `domain.Language("")` or any unrecognized language value is passed, THE Heuristic_Parser SHALL strip only blank lines (lines containing only whitespace), hash the remaining content, and return a valid non-empty hash string without returning an error.
5. IF the file extension (lowercased) is `.go`, THEN THE Ingest_Service SHALL pass `domain.Language("go")`. IF `.py`, THEN `domain.Language("python")`. IF `.js`, THEN `domain.Language("javascript")`. IF `.ts`, THEN `domain.Language("typescript")`. Otherwise `domain.Language("")`.
6. WHEN `Heuristic_Parser.Parse` is called twice on two byte slices that differ only in the placement or count of blank lines between non-blank structural lines, THEN `Heuristic_Parser.StructuralHash` SHALL return equal hash strings for both results.

---

### Requirement 16: Tokenizer — Real BPE Counter

**User Story:** As a developer, I want an accurate token counter rather than a character-length heuristic, so that context trimming and boundary discovery operate with token counts that match what the LLM actually sees.

#### Acceptance Criteria

1. THE Tokenizer package SHALL provide a `TikTokenCounter` struct that implements `domain.Counter` using a BPE tokenizer initialized with the `cl100k_base` encoding.
2. WHEN `TikTokenCounter.Count(text)` is called with a non-empty string, IT SHALL return the exact BPE token count for that string and a nil error. WHEN called with an empty string, IT SHALL return `0` and a nil error.
3. THE `HeuristicCounter` implementation SHALL remain in the package. IF the `TikTokenCounter` cannot be constructed (e.g., the encoding data cannot be loaded), IT SHALL return an error from its constructor function.
4. IF `TikTokenCounter` construction returns an error, THEN `di.NewContainer` SHALL log a warning at `warn` level and fall back to constructing a `HeuristicCounter` instead, without aborting container initialization.
5. IF `TikTokenCounter.Count(s)` returns value `T` and `HeuristicCounter.Count(s)` returns value `H` for the same non-empty string `s` with length between 10 and 10,000 characters, THEN `H/T` SHALL be between 0.5 and 2.0 (i.e., the heuristic is within a factor of 2 of the BPE count).
6. THE `domain.Counter` interface SHALL also include `CountMessages(msgs []domain.Message) (int, error)`. Both `TikTokenCounter` and `HeuristicCounter` SHALL implement this method by summing `Count(msg.Content)` for each message plus 4 tokens per message for role and formatting overhead.

---

### Requirement 17: Dual Entry Point Consolidation

**User Story:** As a developer, I want a single canonical entry point binary, so that `prowiki` refers to exactly one binary regardless of which `main.go` is built.

#### Acceptance Criteria

1. THE repository SHALL have exactly one `main` package at `cmd/prowiki/main.go` whose sole responsibility is to call `cli.Execute()` and handle the returned error.
2. THE `main.go` file at the repository root SHALL NOT contain a `main` package declaration. It SHALL either be deleted or converted to a non-`main` package (e.g., a build-tool or example file with a build constraint).
3. WHEN `go build ./cmd/prowiki` is executed, THE resulting binary SHALL expose all Cobra subcommands currently registered via `rootCmd.AddCommand` in `internal/cli`: `init`, `ingest`, `daemon`, and `server`.
4. THE Makefile `build` target SHALL set its build path to `./cmd/prowiki` and SHALL NOT reference the root `main.go`.
