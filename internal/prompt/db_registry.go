package prompt

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

type PromptStore interface {
	Active(ctx context.Context, stage domain.Stage) (domain.PromptTemplate, error)
}

type DBRegistry struct {
	store PromptStore
}

func NewDBRegistry(store PromptStore) *DBRegistry {
	return &DBRegistry{store: store}
}

func (r *DBRegistry) Active(ctx context.Context, stage domain.Stage) (domain.PromptTemplate, error) {
	return r.store.Active(ctx, stage)
}

func (r *DBRegistry) Render(tmpl domain.PromptTemplate, vars map[string]any) (string, error) {
	t, err := template.New(string(tmpl.Stage)).Parse(tmpl.Template)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return buf.String(), nil
}
