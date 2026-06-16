package llm

import (
	"context"
	"fmt"
	"os"
)

type EnvKeyProvider struct {
	envVar string
}

func NewEnvKeyProvider(envVar string) *EnvKeyProvider {
	if envVar == "" {
		envVar = "LITELLM_API_KEY"
	}
	return &EnvKeyProvider{envVar: envVar}
}

func (p *EnvKeyProvider) APIKey(ctx context.Context, model string) (string, error) {
	key := os.Getenv(p.envVar)
	if key == "" {
		return "", fmt.Errorf("api key not found in env var %s", p.envVar)
	}
	return key, nil
}

func (p *EnvKeyProvider) Refresh(ctx context.Context) error {
	return nil
}

func (p *EnvKeyProvider) Rotate(ctx context.Context) error {
	return nil
}
