package store

import (
	"context"
	"database/sql"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

type GraphStore struct {
	db DBTx
}

func NewGraphStore(db DBTx) *GraphStore {
	return &GraphStore{db: db}
}

func (s *GraphStore) WithTx(tx *sql.Tx) *GraphStore {
	return NewGraphStore(tx)
}

func (s *GraphStore) CreateInteraction(ctx context.Context, interaction *domain.FeatureInteraction) error {
	query := `INSERT INTO feature_interactions (from_feature_id, to_feature_id, description) VALUES (?, ?, ?) RETURNING id, created_at`
	return s.db.QueryRowContext(ctx, query, interaction.FromFeatureID, interaction.ToFeatureID, interaction.Description).Scan(&interaction.ID, &interaction.CreatedAt)
}

// DiscoverMacroPipelines performs a recursive CTE traversal
func (s *GraphStore) DiscoverMacroPipelines(ctx context.Context, projectID int64) ([]int64, error) {
	// Example CTE that traverses interactions up to depth 7
	query := `
		WITH RECURSIVE feature_paths AS (
			SELECT from_feature_id AS current_id, 1 AS depth
			FROM feature_interactions fi
			JOIN features f ON fi.from_feature_id = f.id
			WHERE f.project_id = ?
			
			UNION ALL
			
			SELECT fi.to_feature_id, fp.depth + 1
			FROM feature_interactions fi
			JOIN feature_paths fp ON fi.from_feature_id = fp.current_id
			WHERE fp.depth < 7
		)
		SELECT DISTINCT current_id FROM feature_paths;
	`
	rows, err := s.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
