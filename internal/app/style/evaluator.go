package style

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
	"github.com/andrejsstepanovs/prowiki/internal/llm"
)

type StyleSchema struct {
	Anomalies []AnomalySchema `json:"anomalies"`
}

type AnomalySchema struct {
	Rationale string `json:"rationale"`
}

type Registry interface {
	Active(ctx context.Context, stage domain.Stage) (domain.PromptTemplate, error)
	Render(tmpl domain.PromptTemplate, vars map[string]any) (string, error)
}

type StyleStore interface {
	CreateStyleAnomaly(ctx context.Context, anomaly *domain.StyleAnomaly) error
}

type Evaluator struct {
	completer domain.Completer
	registry  Registry
	store     StyleStore
}

func NewEvaluator(c domain.Completer, r Registry, s StyleStore) *Evaluator {
	return &Evaluator{completer: c, registry: r, store: s}
}

func (e *Evaluator) Evaluate(ctx context.Context, rule domain.CodeStyle, fv domain.FileVersion) error {
	tmpl, err := e.registry.Active(ctx, domain.StageStyleEvaluation)
	if err != nil {
		return fmt.Errorf("missing prompt for style: %w", err)
	}

	promptStr, err := e.registry.Render(tmpl, map[string]any{
		"Rule":    rule.Rule,
		"Content": fv.Content,
	})
	if err != nil {
		return err
	}

	schemaJSON, _ := json.Marshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"anomalies": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"rationale": map[string]any{"type": "string"},
					},
					"required": []string{"rationale"},
				},
			},
		},
		"required": []string{"anomalies"},
	})

	req := domain.CompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []domain.Message{
			{Role: "user", Content: promptStr},
		},
		ResponseFormat: &domain.ResponseFormat{
			Type:   "json_schema",
			Schema: schemaJSON,
		},
	}

	resp, err := e.completer.Complete(ctx, req)
	if err != nil {
		return err
	}

	styleResp, err := llm.ParseStrict[StyleSchema](resp.Text)
	if err != nil {
		return err
	}

	for _, anom := range styleResp.Anomalies {
		a := &domain.StyleAnomaly{
			FileVersionID: fv.ID,
			CodeStyleID:   rule.ID,
			Rationale:     anom.Rationale,
		}
		if err := e.store.CreateStyleAnomaly(ctx, a); err != nil {
			return err
		}
	}

	return nil
}
