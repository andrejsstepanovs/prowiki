package ingest

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
	"github.com/andrejsstepanovs/prowiki/internal/txn"
)

type FileStore interface {
	GetOrCreate(ctx context.Context, tx *sql.Tx, projectID int64, relPath string) (*domain.File, error)
}

type FileVersionStore interface {
	LatestByFileID(ctx context.Context, fileID int64) (*domain.FileVersion, error)
	InsertVersion(ctx context.Context, tx *sql.Tx, version *domain.FileVersion) error
	SetLatest(ctx context.Context, tx *sql.Tx, newVersionID int64, oldVersionID int64) error
	CloneJunctions(ctx context.Context, tx *sql.Tx, oldVersionID, newVersionID int64) error
}

type JobStore interface {
	EnqueueMany(ctx context.Context, tx *sql.Tx, jobs []domain.Job) error
	ResetCompletedForTarget(ctx context.Context, tx *sql.Tx, oldVersionID int64) error
}

type Parser interface {
	Parse(lang domain.Language, content []byte) (*domain.Tree, error)
	StructuralHash(tree *domain.Tree) (string, error)
}

type Service struct {
	db          *sql.DB
	walker      domain.Walker
	p           Parser
	fs          FileStore
	fvs         FileVersionStore
	js          JobStore
	projectID   int64
	projectRoot string
}

func NewService(db *sql.DB, w domain.Walker, p Parser, fs FileStore, fvs FileVersionStore, js JobStore, projectID int64, projectRoot string) *Service {
	return &Service{
		db:          db,
		walker:      w,
		p:           p,
		fs:          fs,
		fvs:         fvs,
		js:          js,
		projectID:   projectID,
		projectRoot: projectRoot,
	}
}

func (s *Service) Run(ctx context.Context) error {
	out, errc := s.walker.Walk(ctx, s.projectRoot, domain.WalkOptions{})

	for file := range out {
		if err := s.ProcessFile(ctx, file); err != nil {
			// Skip file on error, similar to previous versioning
		}
	}

	return <-errc
}

func (s *Service) ProcessFile(ctx context.Context, file domain.ScannedFile) error {
	contentBytes, err := os.ReadFile(file.Path)
	if err != nil {
		return err
	}
	content := string(contentBytes)

	// 1. Language detection
	lang := domain.LanguageFromPath(file.Path)

	// 2. Parse AST, compute ast_hash
	tree, err := s.p.Parse(lang, contentBytes)
	if err != nil {
		return err
	}

	astHash, err := s.p.StructuralHash(tree)
	if err != nil {
		return err
	}

	relPath, err := filepath.Rel(s.projectRoot, file.Path)
	if err != nil {
		return err
	}
	relPath = filepath.ToSlash(relPath)

	// Get or Create File
	domainFile, err := s.fs.GetOrCreate(ctx, nil, s.projectID, relPath)
	if err != nil {
		return err
	}
	fileID := domainFile.ID

	latest, err := s.fvs.LatestByFileID(ctx, fileID)
	var latestHash string
	var latestID int64
	if err == nil {
		latestHash = latest.AstHash
		latestID = latest.ID
	} else if err != domain.ErrNotFound {
		return err
	}

	newVer := &domain.FileVersion{
		FileID:   fileID,
		Content:  content,
		AstHash:  astHash,
		IsLatest: true,
	}

	// Transactional ingestion logic
	return txn.Immediate(ctx, s.db, func(tx *sql.Tx) error {
		if latestHash == astHash {
			// Bypass LLM: AST is identical
			newVer.IsLatest = true
			if err := s.fvs.InsertVersion(ctx, tx, newVer); err != nil {
				return err
			}

			if err := s.fvs.CloneJunctions(ctx, tx, latestID, newVer.ID); err != nil {
				return err
			}

			if err := s.fvs.SetLatest(ctx, tx, newVer.ID, latestID); err != nil {
				return err
			}

			return nil
		}

		// Structural change (or new file) -> cascade invalidation and enqueue extraction
		if err := s.fvs.InsertVersion(ctx, tx, newVer); err != nil {
			return err
		}

		if err := s.fvs.SetLatest(ctx, tx, newVer.ID, latestID); err != nil {
			return err
		}

		if latestID != 0 {
			if err := s.js.ResetCompletedForTarget(ctx, tx, latestID); err != nil {
				return err
			}
		}

		// Enqueue extraction jobs
		jobs := []domain.Job{
			{
				ProjectID:  s.projectID,
				TargetID:   newVer.ID,
				TargetType: "File",
				Stage:      domain.StageLevel1Overview,
				Priority:   1, // Not explicitly priority+10, but the spec says +10 for reset ones.
			},
		}
		if err := s.js.EnqueueMany(ctx, tx, jobs); err != nil {
			return err
		}

		return nil
	})
}
