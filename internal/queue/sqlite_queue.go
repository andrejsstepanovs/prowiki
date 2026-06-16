package queue

import (
	"context"
	"database/sql"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
	"github.com/andrejsstepanovs/prowiki/internal/store"
	"github.com/andrejsstepanovs/prowiki/internal/txn"
)

type SQLiteQueue struct {
	db       *sql.DB
	jobStore *store.JobStore
	dlqStore *store.DLQStore
}

func NewSQLiteQueue(db *sql.DB, js *store.JobStore, dlqs *store.DLQStore) *SQLiteQueue {
	return &SQLiteQueue{
		db:       db,
		jobStore: js,
		dlqStore: dlqs,
	}
}

func (q *SQLiteQueue) ClaimBatch(ctx context.Context, limit int) ([]domain.Job, error) {
	var jobs []domain.Job
	err := txn.Immediate(ctx, q.db, func(tx *sql.Tx) error {
		jTx := q.jobStore.WithTx(tx)
		var innerErr error
		jobs, innerErr = jTx.ClaimBatch(ctx, limit)
		return innerErr
	})
	return jobs, err
}

func (q *SQLiteQueue) Complete(ctx context.Context, jobID int64, fn domain.TxFunc) error {
	return txn.Immediate(ctx, q.db, func(tx *sql.Tx) error {
		if fn != nil {
			if err := fn(tx); err != nil {
				return err
			}
		}

		query := `DELETE FROM job_queue WHERE id = ?`
		_, err := tx.ExecContext(ctx, query, jobID)
		return err
	})
}

func (q *SQLiteQueue) Fail(ctx context.Context, jobID int64, errPayload string) error {
	return txn.Immediate(ctx, q.db, func(tx *sql.Tx) error {
		var retryCount int
		err := tx.QueryRowContext(ctx, `SELECT retry_count FROM job_queue WHERE id = ?`, jobID).Scan(&retryCount)
		if err != nil {
			return err
		}

		if retryCount >= 3 {
			dlqTx := q.dlqStore.WithTx(tx)
			err = dlqTx.Create(ctx, &domain.DeadLetterItem{
				JobID:   jobID,
				Payload: errPayload,
				Reason:  "max retries exceeded",
			})
			if err != nil {
				return err
			}

			jTx := q.jobStore.WithTx(tx)
			return jTx.UpdateStatus(ctx, jobID, domain.JobStatusFailed)
		}

		query := `UPDATE job_queue SET status = ?, retry_count = retry_count + 1 WHERE id = ?`
		_, err = tx.ExecContext(ctx, query, domain.JobStatusPending, jobID)
		return err
	})
}

func (q *SQLiteQueue) Enqueue(ctx context.Context, jobs ...domain.Job) error {
	return txn.Immediate(ctx, q.db, func(tx *sql.Tx) error {
		jTx := q.jobStore.WithTx(tx)
		return jTx.EnqueueMany(ctx, jobs)
	})
}
