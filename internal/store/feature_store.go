package store

import (
	"context"
	"database/sql"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

type FeatureStore struct {
	db DBTx
}

func NewFeatureStore(db DBTx) *FeatureStore {
	return &FeatureStore{db: db}
}

func (s *FeatureStore) WithTx(tx *sql.Tx) *FeatureStore {
	return NewFeatureStore(tx)
}

func (s *FeatureStore) Create(ctx context.Context, feature *domain.Feature) error {
	query := `INSERT INTO features (project_id, name, description) VALUES (?, ?, ?) RETURNING id, created_at, updated_at`
	return s.db.QueryRowContext(ctx, query, feature.ProjectID, feature.Name, feature.Description).Scan(&feature.ID, &feature.CreatedAt, &feature.UpdatedAt)
}

func (s *FeatureStore) AddToFileVersion(ctx context.Context, fileVersionID int64, featureID int64) error {
	query := `INSERT INTO file_features (file_version_id, feature_id) VALUES (?, ?) ON CONFLICT DO NOTHING`
	_, err := s.db.ExecContext(ctx, query, fileVersionID, featureID)
	return err
}

func (s *FeatureStore) GetByProjectID(ctx context.Context, projectID int64) ([]domain.Feature, error) {
	query := `SELECT id, project_id, name, description, created_at, updated_at FROM features WHERE project_id = ? ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var features []domain.Feature
	for rows.Next() {
		var f domain.Feature
		if err := rows.Scan(&f.ID, &f.ProjectID, &f.Name, &f.Description, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		features = append(features, f)
	}
	return features, rows.Err()
}
