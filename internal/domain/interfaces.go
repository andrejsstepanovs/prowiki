package domain

import (
	"context"
	"database/sql"
)

// llm

type CompletionRequest struct {
	Model          string
	Messages       []Message
	ResponseFormat *ResponseFormat
	MaxTokens      int
	Temperature    float32
}

type Message struct {
	Role    string
	Content string
}

type ResponseFormat struct {
	Type   string
	Schema []byte
}

type CompletionResponse struct {
	Text      string
	ToolCalls []any // Placeholder if needed
}

type Completer interface {
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
}

// tokenizer

type Counter interface {
	Count(text string) (int, error)
	CountMessages(msgs []Message) int
}

// ast

type Language string

type Tree struct {
	Hash string // opaque hash representing structural identity
}

type Parser interface {
	Parse(lang Language, content []byte) (*Tree, error)
	StructuralHash(tree *Tree) (string, error)
}

// scanner

type WalkOptions struct {
	ExcludePatterns []string
}

type ScannedFile struct {
	Path     string
	Size     int64
	MTime    int64
	Hash     string
	IsBypass bool
}

type Walker interface {
	Walk(ctx context.Context, root string, opts WalkOptions) (<-chan ScannedFile, <-chan error)
}

// queue

type TxFunc func(*sql.Tx) error

type Queue interface {
	ClaimBatch(ctx context.Context, limit int) ([]Job, error)
	Complete(ctx context.Context, jobID int64, fn TxFunc) error
	Fail(ctx context.Context, jobID int64, errPayload string) error
	Enqueue(ctx context.Context, jobs ...Job) error
}

// prompt

type Registry interface {
	Active(ctx context.Context, stage Stage) (PromptTemplate, error)
	Render(tmpl PromptTemplate, vars map[string]any) (string, error)
}

// scrub

type Scrubber interface {
	Scrub(content string, lang Language) (redacted string, hits int)
}
