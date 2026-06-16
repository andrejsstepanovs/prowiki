package worker

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

type ExtractService interface {
	ProcessOverview(ctx context.Context, job *domain.Job) error
	// ProcessEntity(ctx context.Context, job *domain.Job) error
	// ProcessFeature(ctx context.Context, job *domain.Job) error
}

type StyleEvaluator interface {
	// Evaluate(ctx context.Context, job *domain.Job) error
}

type GraphSynthesizer interface {
	// Synthesize(ctx context.Context, job *domain.Job) error
}

type ServiceDispatcher struct {
	extractor ExtractService
	style     StyleEvaluator
	graph     GraphSynthesizer
}

func NewServiceDispatcher(e ExtractService, s StyleEvaluator, g GraphSynthesizer) *ServiceDispatcher {
	return &ServiceDispatcher{
		extractor: e,
		style:     s,
		graph:     g,
	}
}

func (d *ServiceDispatcher) Dispatch(ctx context.Context, job domain.Job) (domain.TxFunc, error) {
	switch job.Stage {
	case domain.StageLevel1Overview:
		if err := d.extractor.ProcessOverview(ctx, &job); err != nil {
			return nil, err
		}
		// Return empty tx func for now since extraction currently handles its own transactions
		// If ExtractService starts returning TxFunc, we'd pass it along here.
		return func(tx *sql.Tx) error { return nil }, nil

	case domain.StageLevel2Entity:
		// return d.extractor.ProcessEntity(ctx, &job)
		return nil, fmt.Errorf("StageLevel2Entity not yet fully implemented")

	case domain.StageLevel3Feature:
		// return d.extractor.ProcessFeature(ctx, &job)
		return nil, fmt.Errorf("StageLevel3Feature not yet fully implemented")

	case domain.StageLevel4EdgeCase:
		return nil, fmt.Errorf("StageLevel4EdgeCase not yet fully implemented")

	case domain.StageStyleEvaluation:
		// return d.style.Evaluate(ctx, &job)
		return nil, fmt.Errorf("StageStyleEvaluation not yet fully implemented")

	case domain.StageIntersectionSynthesis:
		// return d.graph.Synthesize(ctx, &job)
		return nil, fmt.Errorf("StageIntersectionSynthesis not yet fully implemented")

	default:
		return nil, fmt.Errorf("unknown stage: %s", job.Stage)
	}
}
