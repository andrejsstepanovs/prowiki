package prompt

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

type HardcodedRegistry struct {
	prompts map[domain.Stage]domain.PromptTemplate
}

func NewHardcodedRegistry() *HardcodedRegistry {
	r := &HardcodedRegistry{
		prompts: make(map[domain.Stage]domain.PromptTemplate),
	}

	r.prompts["PARSE"] = domain.PromptTemplate{
		Stage:    "PARSE",
		Template: `You are an expert software engineer analyzing a source code file.
Please extract a concise summary of the file's purpose, and a list of distinct features or capabilities implemented within it.

File Content:
{{.Content}}

Return ONLY a JSON object matching this schema:
{
  "summary": "Brief 1-2 sentence description",
  "features": [
    {"name": "FeatureName", "description": "What it does"}
  ]
}`,
	}

	return r
}

func (r *HardcodedRegistry) Active(ctx context.Context, stage domain.Stage) (domain.PromptTemplate, error) {
	if tmpl, ok := r.prompts[stage]; ok {
		return tmpl, nil
	}
	return domain.PromptTemplate{}, fmt.Errorf("no prompt found for stage: %s", stage)
}

func (r *HardcodedRegistry) Render(tmpl domain.PromptTemplate, vars map[string]any) (string, error) {
	t, err := template.New(string(tmpl.Stage)).Parse(tmpl.Template)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return buf.String(), nil
}
