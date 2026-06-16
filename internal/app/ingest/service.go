package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

type FileVersionStore interface {
	LatestByFileID(ctx context.Context, fileID int64) (*domain.FileVersion, error)
	InsertVersion(ctx context.Context, version *domain.FileVersion) error
	SetLatest(ctx context.Context, newVersionID int64, oldVersionID int64) error
}

type JobStore interface {
	EnqueueMany(ctx context.Context, jobs []domain.Job) error
}

type Parser interface {
	Parse(lang domain.Language, content []byte) (*domain.Tree, error)
	StructuralHash(tree *domain.Tree) (string, error)
}

type Service struct {
	fvs FileVersionStore
	js  JobStore
	p   Parser
}

func NewService(fvs FileVersionStore, js JobStore, p Parser) *Service {
	return &Service{fvs: fvs, js: js, p: p}
}

func (s *Service) ProcessFile(ctx context.Context, projectID int64, file *domain.File, content []byte) error {
	// 1. Compute content SHA-256
	h := sha256.Sum256(content)
	contentHash := hex.EncodeToString(h[:])
	_ = contentHash // Reserved for exact match skipping

	latest, err := s.fvs.LatestByFileID(ctx, file.ID)
	if err != nil && err != domain.ErrNotFound {
		return err
	}

	// 2. Parse AST, compute ast_hash
	tree, err := s.p.Parse(domain.Language("go"), content)
	if err != nil {
		return err
	}
	astHash, err := s.p.StructuralHash(tree)
	if err != nil {
		return err
	}
	
	newVer := &domain.FileVersion{
		FileID:   file.ID,
		Content:  string(content),
		AstHash:  astHash,
		IsLatest: false,
	}
	
	if err := s.fvs.InsertVersion(ctx, newVer); err != nil {
		return err
	}

	if latest != nil && latest.AstHash == astHash {
		// Bypass LLM: AST is identical
		if err := s.fvs.SetLatest(ctx, newVer.ID, latest.ID); err != nil {
			return err
		}
		// Clone feature junctions here in future
		return nil
	}

	// Structural change -> cascade invalidation and enqueue extraction
	oldID := int64(0)
	if latest != nil {
		oldID = latest.ID
	}
	if err := s.fvs.SetLatest(ctx, newVer.ID, oldID); err != nil {
		return err
	}

	// Enqueue extraction jobs
	jobs := []domain.Job{
		{
			ProjectID:  projectID,
			TargetID:   newVer.ID,
			TargetType: "File",
			Stage:      domain.StageLevel1Overview,
			Priority:   1,
		},
	}
	if err := s.js.EnqueueMany(ctx, jobs); err != nil {
		return err
	}

	return nil
}
