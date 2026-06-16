package graph

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
	"github.com/andrejsstepanovs/prowiki/internal/llm"
)

type InteractionSchema struct {
	Interactions []Interaction `json:"interactions"`
}

type Interaction struct {
	ToFeatureID int64  `json:"to_feature_id"`
	Description string `json:"description"`
}

type Registry interface {
	Active(ctx context.Context, stage domain.Stage) (domain.PromptTemplate, error)
	Render(tmpl domain.PromptTemplate, vars map[string]any) (string, error)
}

type GraphStore interface {
	CreateInteraction(ctx context.Context, interaction *domain.FeatureInteraction) error
}

type Synthesizer struct {
	completer domain.Completer
	registry  Registry
	store     GraphStore
}

func NewSynthesizer(c domain.Completer, r Registry, s GraphStore) *Synthesizer {
	return &Synthesizer{completer: c, registry: r, store: s}
}

func (s *Synthesizer) Synthesize(ctx context.Context, f1 domain.Feature, f2 domain.Feature) error {
	tmpl, err := s.registry.Active(ctx, domain.StageIntersectionSynthesis)
	if err != nil {
		return fmt.Errorf("missing prompt for interaction: %w", err)
	}

	promptStr, err := s.registry.Render(tmpl, map[string]any{
		"Feature1": f1.Description,
		"Feature2": f2.Description,
	})
	if err != nil {
		return err
	}

	schemaJSON, _ := json.Marshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"interactions": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"to_feature_id": map[string]any{"type": "integer"},
						"description": map[string]any{"type": "string"},
					},
					"required": []string{"to_feature_id", "description"},
				},
			},
		},
		"required": []string{"interactions"},
	})

	req := domain.CompletionRequest{
		Model: "gpt-4o", // using tier 2 for complex logic like graph connections
		Messages: []domain.Message{
			{Role: "user", Content: promptStr},
		},
		ResponseFormat: &domain.ResponseFormat{
			Type:   "json_schema",
			Schema: schemaJSON,
		},
	}

	resp, err := s.completer.Complete(ctx, req)
	if err != nil {
		return err
	}

	intResp, err := llm.ParseStrict[InteractionSchema](resp.Text)
	if err != nil {
		return err
	}

	for _, interaction := range intResp.Interactions {
		i := &domain.FeatureInteraction{
			FromFeatureID: f1.ID,
			ToFeatureID:   interaction.ToFeatureID,
			Description:   interaction.Description,
		}
		if err := s.store.CreateInteraction(ctx, i); err != nil {
			return err
		}
	}

	return nil
}
