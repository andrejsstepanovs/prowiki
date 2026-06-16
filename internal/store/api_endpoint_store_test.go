package store

import (
	"context"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

func TestApiEndpointStore(t *testing.T) {
	database := setupTestDB(t)
	projectStore := NewProjectStore(database)
	apiStore := NewApiEndpointStore(database)
	ctx := context.Background()

	p := &domain.Project{Name: "API Test"}
	_ = projectStore.Create(ctx, p)

	endpoint := &domain.ApiEndpoint{
		ProjectID:   p.ID,
		Path:        "/users",
		Method:      "GET",
		Description: "Get users",
	}

	err := apiStore.Create(ctx, endpoint)
	if err != nil {
		t.Fatalf("failed to create api endpoint: %v", err)
	}
	if endpoint.ID == 0 {
		t.Fatalf("expected ID to be set")
	}
}
