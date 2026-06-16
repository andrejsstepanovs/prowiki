package llm

import (
	"context"
	"fmt"
	"os"
)

type KeyProvider interface {
	Key(ctx context.Context, model string) (string, error)
	Refresh(ctx context.Context) error
}

type EnvKeyProvider struct {
	envVar string
}

func NewEnvKeyProvider(envVar string) *EnvKeyProvider {
	if envVar == "" {
		envVar = "LITELLM_API_KEY"
	}
	return &EnvKeyProvider{envVar: envVar}
}

func (p *EnvKeyProvider) Key(ctx context.Context, model string) (string, error) {
	key := os.Getenv(p.envVar)
	if key == "" {
		return "", fmt.Errorf("api key not found in env var %s", p.envVar)
	}
	return key, nil
}

func (p *EnvKeyProvider) Refresh(ctx context.Context) error {
	// For simple env vars, there's no dynamic refresh possible.
	// A Vault provider would fetch a new short-lived token here.
	return nil
}
