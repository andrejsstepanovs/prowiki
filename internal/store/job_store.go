package store

import (
	"context"
	"database/sql"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

type JobStore struct {
	db DBTx
}

func NewJobStore(db DBTx) *JobStore {
	return &JobStore{db: db}
}

func (s *JobStore) WithTx(tx *sql.Tx) *JobStore {
	return NewJobStore(tx)
}

func (s *JobStore) ClaimBatch(ctx context.Context, limit int) ([]domain.Job, error) {
	// Atomic claim: update status to processing and return the rows.
	query := `
		UPDATE job_queue 
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id IN (
			SELECT id FROM job_queue 
			WHERE status = ? 
			ORDER BY priority DESC, id ASC 
			LIMIT ?
		)
		RETURNING id, project_id, target_id, target_type, stage, status, priority, retry_count, created_at, updated_at
	`
	rows, err := s.db.QueryContext(ctx, query, domain.JobStatusProcessing, domain.JobStatusPending, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []domain.Job
	for rows.Next() {
		var j domain.Job
		if err := rows.Scan(&j.ID, &j.ProjectID, &j.TargetID, &j.TargetType, &j.Stage, &j.Status, &j.Priority, &j.RetryCount, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func (s *JobStore) UpdateStatus(ctx context.Context, id int64, status domain.JobStatus) error {
	query := `UPDATE job_queue SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, status, id)
	return err
}

func (s *JobStore) EnqueueMany(ctx context.Context, jobs []domain.Job) error {
	if len(jobs) == 0 {
		return nil
	}
	
	query := `INSERT INTO job_queue (project_id, target_id, target_type, stage, status, priority) VALUES `
	args := make([]any, 0, len(jobs)*6)
	
	for i, j := range jobs {
		if i > 0 {
			query += `, `
		}
		query += `(?, ?, ?, ?, ?, ?)`
		args = append(args, j.ProjectID, j.TargetID, j.TargetType, j.Stage, domain.JobStatusPending, j.Priority)
	}

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}
