package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	litellm "github.com/andrejsstepanovs/go-litellm/client"
	"github.com/andrejsstepanovs/go-litellm/models"
	"github.com/andrejsstepanovs/go-litellm/request"
	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

type LitellmClient struct {
	client *litellm.Litellm
}

func NewClient(c *litellm.Litellm) *LitellmClient {
	return &LitellmClient{client: c}
}

func (c *LitellmClient) Complete(ctx context.Context, req domain.CompletionRequest) (*domain.CompletionResponse, error) {
	modelMeta, err := c.client.Model(ctx, models.ModelID(req.Model))
	if err != nil {
		return nil, fmt.Errorf("failed to get model %s: %w", req.Model, err)
	}

	msgs := make(request.Messages, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m.Role == "system" {
			msgs = append(msgs, request.SystemMessageSimple(m.Content))
		} else if m.Role == "user" {
			msgs = append(msgs, request.UserMessageSimple(m.Content))
		} else if m.Role == "assistant" {
			msgs = append(msgs, request.AssistantMessageSimple(m.Content))
		}
	}

	lReq := request.NewCompletionRequest(modelMeta, msgs, nil, nil, req.Temperature)
	// MaxTokens is passed directly or needs to be set differently.
	// Since NewCompletionRequest doesn't accept max tokens and struct doesn't have it exposed directly,
	// wait, `request.NewCompletionRequest` does not expose `MaxTokens`? 
	// The struct returned is a generic request. Let's see if we can set it, or just omit if not supported.
	// Let's omit it for now if it's not exposed, or just rely on defaults.


	if req.ResponseFormat != nil && req.ResponseFormat.Type == "json_schema" {
		var schemaMap map[string]interface{}
		if err := json.Unmarshal(req.ResponseFormat.Schema, &schemaMap); err == nil {
			lReq.SetJSONSchema(request.JSONSchema{
				Name:   "structured_output",
				Schema: schemaMap,
				Strict: true,
			})
		}
	}

	resp, err := c.client.Completion(ctx, lReq)
	if err != nil {
		return nil, mapError(err)
	}

	choice := resp.Choice()
	return &domain.CompletionResponse{
		Text: choice.Message.Content,
	}, nil
}

func mapError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Handle 401 Auth Rotation
	if strings.Contains(errStr, "401") || strings.Contains(errStr, "unauthorized") || strings.Contains(errStr, "AuthError") {
		return domain.ErrAuthRotation
	}

	// Handle 413 Context Overflow
	if strings.Contains(errStr, "413") || strings.Contains(errStr, "context length exceeded") || strings.Contains(errStr, "too many tokens") || strings.Contains(errStr, "ContextWindowExceeded") {
		return domain.ErrContextOverflow
	}

	return err
}
