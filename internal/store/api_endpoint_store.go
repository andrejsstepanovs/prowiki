package store

import (
	"context"
	"database/sql"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

type ApiEndpointStore struct {
	db DBTx
}

func NewApiEndpointStore(db DBTx) *ApiEndpointStore {
	return &ApiEndpointStore{db: db}
}

func (s *ApiEndpointStore) WithTx(tx *sql.Tx) *ApiEndpointStore {
	return NewApiEndpointStore(tx)
}

func (s *ApiEndpointStore) Create(ctx context.Context, endpoint *domain.ApiEndpoint) error {
	query := `INSERT INTO api_endpoints (project_id, path, method, description) VALUES (?, ?, ?, ?) RETURNING id, created_at, updated_at`
	return s.db.QueryRowContext(ctx, query, endpoint.ProjectID, endpoint.Path, endpoint.Method, endpoint.Description).Scan(&endpoint.ID, &endpoint.CreatedAt, &endpoint.UpdatedAt)
}
