package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
	"github.com/andrejsstepanovs/prowiki/internal/store"
)

type Server struct {
	port      int
	server    *http.Server
	project   *domain.Project
	pStore    *store.ProjectStore
	fStore    *store.FileStore
	vStore    *store.FileVersionStore
	featStore *store.FeatureStore
	jStore    *store.JobStore
}

func NewServer(port int, project *domain.Project, pStore *store.ProjectStore, fStore *store.FileStore, vStore *store.FileVersionStore, featStore *store.FeatureStore, jStore *store.JobStore) *Server {
	return &Server{
		port:      port,
		project:   project,
		pStore:    pStore,
		fStore:    fStore,
		vStore:    vStore,
		featStore: featStore,
		jStore:    jStore,
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// API Routes
	mux.HandleFunc("/api/project", s.handleGetProject)
	mux.HandleFunc("/api/files", s.handleGetFiles)
	mux.HandleFunc("/api/files/", s.handleGetFile)
	mux.HandleFunc("/api/features", s.handleGetFeatures)
	mux.HandleFunc("/api/jobs", s.handleGetJobs)

	// Serve the embedded/static web assets
	mux.Handle("/", http.FileServer(http.Dir("web")))

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: corsMiddleware(mux),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.server.Shutdown(shutdownCtx)
	}()

	fmt.Printf("API Server listening on http://localhost:%d\n", s.port)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
