package provider

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockProvider implements Provider for testing.
type mockProvider struct{}

func (m *mockProvider) Invoke(_ context.Context, _ string, _ string) (string, error) {
	return "response", nil
}

func (m *mockProvider) ListModels(_ context.Context) ([]Model, error) {
	return []Model{{ID: "m1", Name: "Model One"}}, nil
}

func (m *mockProvider) MaxTokens(_ string) int {
	return 128000
}

func TestProviderInterfaceSatisfied(t *testing.T) {
	var p Provider = &mockProvider{}
	resp, err := p.Invoke(context.Background(), "model", "prompt")
	assert.NoError(t, err)
	assert.Equal(t, "response", resp)

	models, err := p.ListModels(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, []Model{{ID: "m1", Name: "Model One"}}, models)

	assert.Equal(t, 128000, p.MaxTokens("any"))
}

func TestErrorWrapping(t *testing.T) {
	tests := []struct {
		name     string
		sentinel error
	}{
		{"ErrAuth", ErrAuth},
		{"ErrRateLimit", ErrRateLimit},
		{"ErrTimeout", ErrTimeout},
		{"ErrModelNotFound", ErrModelNotFound},
		{"ErrUnknownProvider", ErrUnknownProvider},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := fmt.Errorf("bedrock: %w", tt.sentinel)
			assert.True(t, errors.Is(wrapped, tt.sentinel))
		})
	}
}

func TestFactoryKnownProvider(t *testing.T) {
	Register("mock", func(_ map[string]string) (Provider, error) {
		return &mockProvider{}, nil
	})
	defer func() { delete(registry, "mock") }()

	p, err := New("mock", nil)
	assert.NoError(t, err)
	assert.NotNil(t, p)
}

func TestFactoryUnknownProvider(t *testing.T) {
	_, err := New("nonexistent", nil)
	assert.ErrorIs(t, err, ErrUnknownProvider)
}
