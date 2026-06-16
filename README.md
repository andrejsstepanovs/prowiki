# ProWiki

ProWiki is an intelligent, pure-Go background daemon designed to continuously scan, parse, and summarize your source code using Large Language Models. 

It is completely self-contained. There are no external dependencies like Redis or RabbitMQ; the entire queue and orchestration engine runs on an embedded SQLite database (`.prowiki.db`) under a safe, single-threaded WAL connection. It is built strictly with Go (CGo-free).

## Features
- **Atomic Ingestion Pipeline**: Scans your codebase, honors `.gitignore`, and efficiently detects modifications via a structural AST hash.
- **Background Daemon**: A panic-safe polling worker seamlessly pulls intelligence jobs from the SQLite queue.
- **LLM Code Intelligence**: Natively calls out to the OpenAI API (via `go-litellm`) to summarize file versions and extract features dynamically into strongly-typed domains.
- **Dead Letter Queue (DLQ)**: Built-in fault tolerance. Jobs that fail repeatedly are parked safely without crashing the orchestrator.

## Installation

Ensure you have Go installed (1.20+), then build the CLI binary:

```bash
go build -o prowiki cmd/prowiki/main.go
```

## Usage

ProWiki uses a command-line interface with three primary commands.

### 1. Initialize a Project

Initialize the ProWiki database within your target directory:

```bash
./prowiki init /path/to/your/project
```
This command bootstraps a local `.prowiki.db` SQLite file in that directory and runs all necessary database migrations.

### 2. Ingest Code

Run the codebase scanner to find all files and track modifications:

```bash
./prowiki ingest /path/to/your/project
```
The ingestion process will structurally hash the files and safely queue `PARSE` jobs for any changed files.

### 3. Run the Intelligence Daemon

Start the background queue worker. Be sure to provide your API key to the daemon so it can run the LLM intelligence routines.

```bash
export OPENAI_API_KEY="your-sk-key"
./prowiki daemon /path/to/your/project
```
The daemon will safely process background jobs, fetch file summaries, and automatically queue downstream feature extractions!
