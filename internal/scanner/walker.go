package scanner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
	ignore "github.com/sabhiram/go-gitignore"
)

type DefaultWalker struct{}

func NewDefaultWalker() *DefaultWalker {
	return &DefaultWalker{}
}

func (w *DefaultWalker) Walk(ctx context.Context, root string, opts domain.WalkOptions) (<-chan domain.ScannedFile, <-chan error) {
	out := make(chan domain.ScannedFile)
	errc := make(chan error, 1)

	go func() {
		defer close(out)
		defer close(errc)

		gitIgnorePath := filepath.Join(root, ".gitignore")
		ignorer, err := ignore.CompileIgnoreFile(gitIgnorePath)
		if err != nil || ignorer == nil {
			ignorer = ignore.CompileIgnoreLines(".git", "node_modules", "vendor")
		}

		err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				if info != nil && info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			if ctx.Err() != nil {
				return ctx.Err()
			}

			relPath, err := filepath.Rel(root, path)
			if err != nil {
				return nil
			}
			if relPath == "." {
				return nil
			}

			relPath = filepath.ToSlash(relPath)

			// Fast ignore logic
			if relPath == ".git" || strings.HasPrefix(relPath, ".git/") {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			if ignorer != nil && ignorer.MatchesPath(relPath) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			for _, p := range opts.ExcludePatterns {
				if strings.Contains(relPath, p) {
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}

			// Ignore large files
			if !info.IsDir() && info.Size() > 1024*1024 {
				return nil
			}

			if !info.IsDir() {
				hashStr := ""
				f, err := os.Open(path)
				if err == nil {
					h := sha256.New()
					io.Copy(h, f)
					f.Close()
					hashStr = hex.EncodeToString(h.Sum(nil))
				}

				select {
				case <-ctx.Done():
					return ctx.Err()
				case out <- domain.ScannedFile{
					Path:  path,
					Size:  info.Size(),
					MTime: info.ModTime().UnixNano(),
					Hash:  hashStr,
				}:
				}
			}
			return nil
		})
		if err != nil {
			errc <- err
		}
	}()

	return out, errc
}
