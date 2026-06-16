package versioning

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
	"github.com/andrejsstepanovs/prowiki/internal/store"
	"github.com/andrejsstepanovs/prowiki/internal/txn"
)

type IngestionService struct {
	db          *sql.DB
	walker      domain.Walker
	parser      domain.Parser
	fileStore   *store.FileStore
	verStore    *store.FileVersionStore
	jobStore    *store.JobStore
	projectID   int64
	projectRoot string
}

func NewIngestionService(db *sql.DB, w domain.Walker, p domain.Parser, fs *store.FileStore, vs *store.FileVersionStore, js *store.JobStore, projectID int64, projectRoot string) *IngestionService {
	return &IngestionService{
		db:          db,
		walker:      w,
		parser:      p,
		fileStore:   fs,
		verStore:    vs,
		jobStore:    js,
		projectID:   projectID,
		projectRoot: projectRoot,
	}
}

func (s *IngestionService) Run(ctx context.Context) error {
	out, errc := s.walker.Walk(ctx, s.projectRoot, domain.WalkOptions{})

	for file := range out {
		// Read content
		contentBytes, err := os.ReadFile(file.Path)
		if err != nil {
			continue // Skip unreadable files silently for now
		}
		content := string(contentBytes)

		// Parse AST hash
		// For simplicity, passing empty language. We'd infer it normally.
		tree, err := s.parser.Parse(domain.Language(""), contentBytes)
		if err != nil {
			continue
		}

		astHash, _ := s.parser.StructuralHash(tree)

		// Transactional insertion
		err = txn.Immediate(ctx, s.db, func(tx *sql.Tx) error {
			fTx := s.fileStore.WithTx(tx)
			vTx := s.verStore.WithTx(tx)
			jTx := s.jobStore.WithTx(tx)

			relPath, err := filepath.Rel(s.projectRoot, file.Path)
			if err != nil {
				return err
			}
			relPath = filepath.ToSlash(relPath)

			// 1. Get or Create File
			var fileID int64
			err = tx.QueryRowContext(ctx, `SELECT id FROM files WHERE project_id = ? AND path = ?`, s.projectID, relPath).Scan(&fileID)
			if err != nil {
				if err == sql.ErrNoRows {
					f := &domain.File{
						ProjectID: s.projectID,
						Path:      relPath,
					}
					if err := fTx.Create(ctx, f); err != nil {
						return fmt.Errorf("failed to create file: %w", err)
					}
					fileID = f.ID
				} else {
					return err
				}
			}

			// 2. Check Latest Version
			latest, err := vTx.LatestByFileID(ctx, fileID)
			var latestHash string
			var latestID int64
			if err == nil {
				latestHash = latest.AstHash
				latestID = latest.ID
			} else if err != domain.ErrNotFound {
				return err
			}

			// 3. If hash changed (or new file), insert new version
			if latestHash != astHash {
				newVer := &domain.FileVersion{
					FileID:   fileID,
					Content:  content,
					AstHash:  astHash,
					IsLatest: true,
				}
				if err := vTx.InsertVersion(ctx, newVer); err != nil {
					return fmt.Errorf("failed to insert version: %w", err)
				}

				if err := vTx.SetLatest(ctx, newVer.ID, latestID); err != nil {
					return fmt.Errorf("failed to set latest: %w", err)
				}

				// 4. Enqueue Parse Job
				job := domain.Job{
					ProjectID:  s.projectID,
					TargetID:   newVer.ID,
					TargetType: "FILE_VERSION", 
					Stage:      "PARSE",        
					Priority:   10,
				}
				if err := jTx.EnqueueMany(ctx, []domain.Job{job}); err != nil {
					return fmt.Errorf("failed to enqueue job: %w", err)
				}
			}

			return nil
		})
		
		if err != nil {
			// we log but don't halt the whole walker
		}
	}

	return <-errc
}
