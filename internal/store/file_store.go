package store

import (
	"context"
	"database/sql"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

type FileStore struct {
	db DBTx
}

func NewFileStore(db DBTx) *FileStore {
	return &FileStore{db: db}
}

func (s *FileStore) WithTx(tx *sql.Tx) *FileStore {
	return NewFileStore(tx)
}

func (s *FileStore) Create(ctx context.Context, file *domain.File) error {
	query := `INSERT INTO files (project_id, folder_id, path) VALUES (?, ?, ?) RETURNING id, created_at, updated_at`
	err := s.db.QueryRowContext(ctx, query, file.ProjectID, file.FolderID, file.Path).Scan(&file.ID, &file.CreatedAt, &file.UpdatedAt)
	return err
}

type FileVersionStore struct {
	db DBTx
}

func NewFileVersionStore(db DBTx) *FileVersionStore {
	return &FileVersionStore{db: db}
}

func (s *FileVersionStore) WithTx(tx *sql.Tx) *FileVersionStore {
	return NewFileVersionStore(tx)
}

func (s *FileVersionStore) InsertVersion(ctx context.Context, version *domain.FileVersion) error {
	query := `INSERT INTO file_versions (file_id, content, ast_hash, is_latest) VALUES (?, ?, ?, ?) RETURNING id, created_at`
	err := s.db.QueryRowContext(ctx, query, version.FileID, version.Content, version.AstHash, version.IsLatest).Scan(&version.ID, &version.CreatedAt)
	return err
}

func (s *FileVersionStore) LatestByFileID(ctx context.Context, fileID int64) (*domain.FileVersion, error) {
	query := `SELECT id, file_id, content, ast_hash, is_latest, created_at FROM file_versions WHERE file_id = ? AND is_latest = 1`
	var fv domain.FileVersion
	err := s.db.QueryRowContext(ctx, query, fileID).Scan(&fv.ID, &fv.FileID, &fv.Content, &fv.AstHash, &fv.IsLatest, &fv.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &fv, nil
}

// SetLatest atomically swaps the is_latest flag
func (s *FileVersionStore) SetLatest(ctx context.Context, newVersionID int64, oldVersionID int64) error {
	if oldVersionID != 0 {
		queryOld := `UPDATE file_versions SET is_latest = 0 WHERE id = ?`
		_, err := s.db.ExecContext(ctx, queryOld, oldVersionID)
		if err != nil {
			return err
		}
	}

	queryNew := `UPDATE file_versions SET is_latest = 1 WHERE id = ?`
	_, err := s.db.ExecContext(ctx, queryNew, newVersionID)
	return err
}

func (s *FileVersionStore) GetByID(ctx context.Context, id int64) (*domain.FileVersion, error) {
	query := `SELECT id, file_id, content, ast_hash, is_latest, summary, created_at FROM file_versions WHERE id = ?`
	var fv domain.FileVersion
	var summary sql.NullString
	err := s.db.QueryRowContext(ctx, query, id).Scan(&fv.ID, &fv.FileID, &fv.Content, &fv.AstHash, &fv.IsLatest, &summary, &fv.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if summary.Valid {
		fv.Summary = summary.String
	}
	return &fv, nil
}

func (s *FileVersionStore) UpdateSummary(ctx context.Context, id int64, summary string) error {
	query := `UPDATE file_versions SET summary = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, summary, id)
	return err
}
