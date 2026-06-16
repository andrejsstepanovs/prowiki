package handlers

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
	"github.com/andrejsstepanovs/prowiki/internal/llm"
	"github.com/andrejsstepanovs/prowiki/internal/store"
)

type ParseResult struct {
	Summary  string `json:"summary"`
	Features []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"features"`
}

type ParseHandler struct {
	completer domain.Completer
	registry  domain.Registry
	vStore    *store.FileVersionStore
	fStore    *store.FeatureStore
	jStore    *store.JobStore
}

func NewParseHandler(c domain.Completer, r domain.Registry, vs *store.FileVersionStore, fs *store.FeatureStore, js *store.JobStore) *ParseHandler {
	return &ParseHandler{
		completer: c,
		registry:  r,
		vStore:    vs,
		fStore:    fs,
		jStore:    js,
	}
}

func (h *ParseHandler) Handle(ctx context.Context, job domain.Job) (domain.TxFunc, error) {
	// 1. Fetch FileVersion content
	version, err := h.vStore.GetByID(ctx, job.TargetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file version: %w", err)
	}

	// 2. Render Prompt
	tmpl, err := h.registry.Active(ctx, "PARSE")
	if err != nil {
		return nil, fmt.Errorf("failed to get parse prompt: %w", err)
	}

	rendered, err := h.registry.Render(tmpl, map[string]any{
		"Content": version.Content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to render parse prompt: %w", err)
	}

	// 3. Call LLM
	req := domain.CompletionRequest{
		Model: "gpt-4o-mini", // Assume a default or fetch from project config
		Messages: []domain.Message{
			{Role: "system", Content: rendered},
		},
		ResponseFormat: &domain.ResponseFormat{Type: "json_object"},
		Temperature:    0.1,
	}

	resp, err := h.completer.Complete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("llm completion failed: %w", err)
	}

	// 4. Parse Strict JSON
	result, err := llm.ParseStrict[ParseResult](resp.Text)
	if err != nil {
		return nil, fmt.Errorf("failed to parse json strict: %w", err)
	}

	// 5. Return atomic transaction block
	return func(tx *sql.Tx) error {
		// Update summary in file_versions
		vTx := h.vStore.WithTx(tx)
		if err := vTx.UpdateSummary(ctx, version.ID, result.Summary); err != nil {
			return err
		}

		// Insert features
		fTx := h.fStore.WithTx(tx)
		for _, feat := range result.Features {
			f := &domain.Feature{
				ProjectID:   job.ProjectID,
				Name:        feat.Name,
				Description: feat.Description,
			}
			if err := fTx.Create(ctx, f); err != nil {
				return err
			}
			if err := fTx.AddToFileVersion(ctx, version.ID, f.ID); err != nil {
				return err
			}
		}

		// Enqueue downstream job (e.g., ANALYZE_STYLE)
		jTx := h.jStore.WithTx(tx)
		downstreamJob := domain.Job{
			ProjectID:  job.ProjectID,
			TargetID:   version.ID,
			TargetType: "FILE_VERSION",
			Stage:      "ANALYZE_STYLE",
			Priority:   5, // lower priority
		}
		if err := jTx.EnqueueMany(ctx, []domain.Job{downstreamJob}); err != nil {
			return err
		}

		return nil
	}, nil
}
