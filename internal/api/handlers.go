package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.jsonResponse(w, http.StatusOK, s.project)
}

func (s *Server) handleGetFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	files, err := s.fStore.GetByProjectID(r.Context(), s.project.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.jsonResponse(w, http.StatusOK, files)
}

func (s *Server) handleGetFeatures(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	feats, err := s.featStore.GetByProjectID(r.Context(), s.project.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.jsonResponse(w, http.StatusOK, feats)
}

func (s *Server) handleGetJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	stats, err := s.jStore.GetStats(r.Context(), s.project.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.jsonResponse(w, http.StatusOK, stats)
}

func (s *Server) handleGetFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Path[len("/api/files/"):]
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		http.Error(w, "Invalid file ID", http.StatusBadRequest)
		return
	}

	// Fetch latest file version
	version, err := s.vStore.LatestByFileID(r.Context(), id)
	if err != nil {
		http.Error(w, "File version not found", http.StatusNotFound)
		return
	}

	// Fetch features for this file
	// We'll need a way to fetch features by file_version_id
	// Or we can just return what we have right now
	// To keep it simple without adding more store methods, we can just return the summary for now
	// Let's implement a quick payload
	type FileResponse struct {
		Summary  string           `json:"summary"`
		Features []domain.Feature `json:"features"`
	}

	resp := FileResponse{
		Summary:  version.Summary,
		Features: []domain.Feature{}, // We'll skip per-file features for now and use global ones or add the query
	}

	s.jsonResponse(w, http.StatusOK, resp)
}
