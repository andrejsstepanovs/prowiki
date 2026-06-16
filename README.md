# ProWiki

ProWiki is an intelligent, pure-Go background daemon designed to continuously scan, parse, and summarize your source code using Large Language Models. 

It is completely self-contained. There are no external dependencies like Redis or RabbitMQ; the entire queue and orchestration engine runs on an embedded SQLite database (`.prowiki.db`) under a safe, single-threaded WAL connection. It is built strictly with Go (CGo-free).

## Core Architecture & Features

- **AST Structural Bypass**: An intelligent ingestion scanner that reads your Go codebase, hashes the AST structures, and bypasses LLM extraction entirely if only whitespace or comments have changed.
- **Secret Scrubber Engine**: A regex-based redaction pipeline that scrubs hardcoded API keys and secrets before code is sent to any external LLM provider.
- **Transactional SQLite Queue**: A panic-safe polling worker seamlessly pulls jobs using atomic SQLite pointer swaps. Includes a **Dead Letter Queue (DLQ)** to safely park poison pill jobs after 3 retries without crashing.
- **Multi-Pass Extraction**:
  - `Level 1`: File Overview and Summarization
  - `Level 2`: Entity Extraction (Structs, Types, Funcs)
  - `Level 3`: Feature Architecture Graph Synthesis
  - `Level 4`: Style Anomalies & Code Review
- **LLM Cost Controls**: Built with an `ExponentialBackoff` retry system for `5xx` errors, and a `DiscoverBoundary` function that dynamically binary searches for maximum safe token capacity before hitting Context Limits.

## Installation

Ensure you have Go installed (1.20+), then build the CLI binary:

```bash
go build -o prowiki cmd/prowiki/main.go
```

## Usage

ProWiki features a modern CLI built on `Cobra` and `Viper`. You can specify the target repository directory via the `-d` flag or the `PROWIKI_DIR` environment variable.

### 1. Initialize a Project

Initialize the ProWiki database within your target directory:

```bash
./prowiki init -d /path/to/your/project
```
This command bootstraps a local `.prowiki.db` SQLite file in that directory and runs all necessary database migrations (including seeding initial LLM prompts).

### 2. Ingest Code

Run the codebase scanner to find all files and track modifications:

```bash
./prowiki ingest -d /path/to/your/project
```
The ingestion process will structurally hash the files and safely queue `Level 1 Overview` jobs for any changed files.

### 3. Run the Intelligence Daemon

Start the background queue worker. Be sure to provide your API key (using the `PROWIKI_` prefix) to the daemon so it can run the LLM intelligence routines.

```bash
export PROWIKI_API_KEY="your-sk-key"
./prowiki daemon -d /path/to/your/project
```
The daemon will safely process background jobs, fetch file summaries, evaluate coding styles, map feature architectures, and automatically queue downstream feature extractions!

### 4. Dashboard Server (Upcoming)

Start the web dashboard and API (Implementation pending):

```bash
./prowiki server -d /path/to/your/project
```
