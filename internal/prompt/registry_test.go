package prompt

import (
	"context"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

func TestHardcodedRegistry(t *testing.T) {
	r := NewHardcodedRegistry()
	ctx := context.Background()

	tmpl, err := r.Active(ctx, "PARSE")
	if err != nil {
		t.Fatalf("expected to find parse prompt, got %v", err)
	}

	vars := map[string]any{"Content": "func main() {}"}
	out, err := r.Render(tmpl, vars)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	if !strings.Contains(out, "func main() {}") {
		t.Fatalf("expected output to contain rendered content, got: %s", out)
	}
}

func TestPropertyRenderErrorIncludesStageName(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		stage := rapid.StringMatching(`^[A-Z0-9_]{1,20}$`).Draw(t, "stage")
		
		r := NewHardcodedRegistry()
		
		tmpl := domain.PromptTemplate{
			Stage: domain.Stage(stage),
			// This parses correctly but fails during execution
			Template: `{{ .NilStruct.NonExistentField }}`,
		}
		
		_, err := r.Render(tmpl, map[string]any{"NilStruct": nil})
		
		if err == nil {
			t.Fatalf("expected error during render, got nil")
		}
		
		if !strings.Contains(err.Error(), stage) {
			t.Fatalf("expected error to contain stage name %q, got: %v", stage, err)
		}
	})
}
