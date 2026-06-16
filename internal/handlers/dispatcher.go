package handlers

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

type Dispatcher struct {
	parseHandler func(ctx context.Context, job domain.Job) (domain.TxFunc, error)
}

func NewDispatcher(parseHandler func(ctx context.Context, job domain.Job) (domain.TxFunc, error)) *Dispatcher {
	return &Dispatcher{
		parseHandler: parseHandler,
	}
}

func (d *Dispatcher) Dispatch(ctx context.Context, job domain.Job) (domain.TxFunc, error) {
	switch job.Stage {
	case "PARSE":
		if d.parseHandler == nil {
			return nil, fmt.Errorf("no handler registered for stage: %s", job.Stage)
		}
		return d.parseHandler(ctx, job)
	default:
		return nil, fmt.Errorf("unknown job stage: %s", job.Stage)
	}
}

func NoOpHandler(ctx context.Context, job domain.Job) (domain.TxFunc, error) {
	return func(tx *sql.Tx) error { return nil }, nil
}
