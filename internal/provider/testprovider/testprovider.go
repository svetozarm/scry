//go:build integration

package testprovider

import (
	"context"
	"fmt"
	"os"

	"github.com/svetozarm/scry/internal/provider"
)

func init() {
	provider.Register("test", func(_ map[string]string) (provider.Provider, error) {
		return &testProvider{}, nil
	})
}

type testProvider struct{}

func (p *testProvider) Invoke(_ context.Context, modelID string, _ string) (string, error) {
	if errType := os.Getenv("TEST_PROVIDER_ERROR"); errType != "" {
		switch errType {
		case "auth":
			return "", provider.ErrAuth
		case "ratelimit":
			return "", provider.ErrRateLimit
		case "timeout":
			return "", provider.ErrTimeout
		case "model":
			return "", fmt.Errorf("model %q: %w", modelID, provider.ErrModelNotFound)
		}
	}
	if msg := os.Getenv("TEST_PROVIDER_RESPONSE"); msg != "" {
		return msg, nil
	}
	return "feat: add new feature", nil
}

func (p *testProvider) ListModels(_ context.Context) ([]provider.Model, error) {
	return []provider.Model{
		{ID: "test-model-1", Name: "Test Model 1"},
		{ID: "test-model-2", Name: "Test Model 2"},
	}, nil
}

func (p *testProvider) MaxTokens(_ string) int { return 128000 }
