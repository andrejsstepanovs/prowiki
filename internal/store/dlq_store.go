package store

import (
	"context"
	"database/sql"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

type DLQStore struct {
	db DBTx
}

func NewDLQStore(db DBTx) *DLQStore {
	return &DLQStore{db: db}
}

func (s *DLQStore) WithTx(tx *sql.Tx) *DLQStore {
	return NewDLQStore(tx)
}

func (s *DLQStore) Create(ctx context.Context, item *domain.DeadLetterItem) error {
	query := `INSERT INTO dead_letter_queue (job_id, payload, reason) VALUES (?, ?, ?) RETURNING id, created_at`
	return s.db.QueryRowContext(ctx, query, item.JobID, item.Payload, item.Reason).Scan(&item.ID, &item.CreatedAt)
}
