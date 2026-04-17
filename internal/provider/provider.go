package provider

import (
	"context"
	"fmt"
)

// Provider defines the interface for LLM backends.
type Provider interface {
	Invoke(ctx context.Context, modelID string, prompt string) (string, error)
	ListModels(ctx context.Context) ([]Model, error)
	MaxTokens(modelID string) int
}

// Model represents an available LLM model.
type Model struct {
	ID   string
	Name string
}

// FactoryFunc creates a Provider from config values.
type FactoryFunc func(config map[string]string) (Provider, error)

var registry = map[string]FactoryFunc{}

// Register adds a provider factory under the given name.
func Register(name string, factory FactoryFunc) {
	registry[name] = factory
}

// New creates a Provider by looking up the named factory in the registry.
func New(name string, config map[string]string) (Provider, error) {
	factory, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("%q: %w", name, ErrUnknownProvider)
	}
	return factory(config)
}
