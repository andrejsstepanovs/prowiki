package extract

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
	"github.com/andrejsstepanovs/prowiki/internal/llm"
)

type Registry interface {
	Active(ctx context.Context, stage domain.Stage) (domain.PromptTemplate, error)
	Render(tmpl domain.PromptTemplate, vars map[string]any) (string, error)
}

type FeatureStore interface {
	Create(ctx context.Context, feature *domain.Feature) error
	AddToFileVersion(ctx context.Context, fileVersionID int64, featureID int64) error
}

type EntityStore interface {
	Create(ctx context.Context, entity *domain.Entity) error
}

type FileVersionStore interface {
	GetByID(ctx context.Context, id int64) (*domain.FileVersion, error)
	UpdateSummary(ctx context.Context, id int64, summary string) error
}

type JobStore interface {
	EnqueueMany(ctx context.Context, jobs []domain.Job) error
	UpdateStatus(ctx context.Context, jobID int64, status domain.JobStatus) error
}

type Scrubber interface {
	Scrub(content string, lang domain.Language) (string, int)
}

type Service struct {
	completer domain.Completer
	registry  Registry
	fvs       FileVersionStore
	fs        FeatureStore
	es        EntityStore
	js        JobStore
	scrub     Scrubber
}

func NewService(c domain.Completer, r Registry, fvs FileVersionStore, fs FeatureStore, es EntityStore, js JobStore, scrub Scrubber) *Service {
	return &Service{
		completer: c,
		registry:  r,
		fvs:       fvs,
		fs:        fs,
		es:        es,
		js:        js,
		scrub:     scrub,
	}
}

func (s *Service) ProcessOverview(ctx context.Context, job *domain.Job) error {
	fv, err := s.fvs.GetByID(ctx, job.TargetID)
	if err != nil {
		return fmt.Errorf("failed to get file version: %w", err)
	}

	content, _ := s.scrub.Scrub(fv.Content, domain.Language("go"))

	tmpl, err := s.registry.Active(ctx, domain.StageLevel1Overview)
	if err != nil {
		return fmt.Errorf("missing prompt: %w", err)
	}

	promptStr, err := s.registry.Render(tmpl, map[string]any{"Content": content})
	if err != nil {
		return err
	}

	schemaJSON, _ := json.Marshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"summary": map[string]any{"type": "string"},
			"features": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string"},
						"description": map[string]any{"type": "string"},
					},
					"required": []string{"name", "description"},
				},
			},
		},
		"required": []string{"summary", "features"},
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

	resp, err := s.completer.Complete(ctx, req)
	if err != nil {
		return fmt.Errorf("completion failed: %w", err)
	}

	overview, err := llm.ParseStrict[OverviewSchema](resp.Text)
	if err != nil {
		return fmt.Errorf("schema parsing failed: %w", err)
	}

	if err := s.fvs.UpdateSummary(ctx, fv.ID, overview.Summary); err != nil {
		return err
	}

	for _, feat := range overview.Features {
		f := &domain.Feature{
			ProjectID:   job.ProjectID,
			Name:        feat.Name,
			Description: feat.Description,
		}
		if err := s.fs.Create(ctx, f); err != nil {
			return err
		}
		if err := s.fs.AddToFileVersion(ctx, fv.ID, f.ID); err != nil {
			return err
		}
	}

	nextJob := domain.Job{
		ProjectID:  job.ProjectID,
		TargetID:   job.TargetID,
		TargetType: "File",
		Stage:      domain.StageLevel2Entity,
		Priority:   1,
	}
	if err := s.js.EnqueueMany(ctx, []domain.Job{nextJob}); err != nil {
		return err
	}

	return s.js.UpdateStatus(ctx, job.ID, domain.JobStatusCompleted)
}
