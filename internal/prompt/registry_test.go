package prompt

import (
	"context"
	"strings"
	"testing"
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
