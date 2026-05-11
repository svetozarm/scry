package openrouter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/svetozarm/scry/internal/provider"
)

func TestNew_DefaultEndpoint(t *testing.T) {
	p, err := New(map[string]string{"api_key": "test-key"})
	require.NoError(t, err)
	assert.Equal(t, defaultEndpoint, p.(*OpenRouterProvider).endpoint)
}

func TestNew_CustomEndpoint(t *testing.T) {
	p, err := New(map[string]string{"api_key": "test-key", "endpoint": "http://myhost:1234"})
	require.NoError(t, err)
	assert.Equal(t, "http://myhost:1234", p.(*OpenRouterProvider).endpoint)
}

func TestNew_TrailingSlashTrimmed(t *testing.T) {
	p, err := New(map[string]string{"api_key": "test-key", "endpoint": "http://myhost:1234/"})
	require.NoError(t, err)
	assert.Equal(t, "http://myhost:1234", p.(*OpenRouterProvider).endpoint)
}

func TestNew_APIKeyFromEnvironment(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "env-key")

	p, err := New(map[string]string{})
	require.NoError(t, err)
	assert.Equal(t, "env-key", p.(*OpenRouterProvider).apiKey)
}

func TestNew_MissingAPIKey(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "")

	_, err := New(map[string]string{})
	assert.ErrorIs(t, err, provider.ErrAuth)
}

func TestInvoke_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "https://example.com", r.Header.Get("HTTP-Referer"))
		assert.Equal(t, "Scry", r.Header.Get("X-OpenRouter-Title"))

		var req chatRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "openai/gpt-5.2", req.Model)
		assert.False(t, req.Stream)
		require.Len(t, req.Messages, 1)
		assert.Equal(t, "user", req.Messages[0].Role)
		assert.Equal(t, "generate commit", req.Messages[0].Content)
		assert.Equal(t, 512, req.MaxTokens)

		json.NewEncoder(w).Encode(chatResponse{
			Choices: []chatChoice{{Message: chatMessage{Content: "feat: add openrouter"}}},
		})
	}))
	defer srv.Close()

	p, err := New(map[string]string{
		"api_key":               "test-key",
		"endpoint":              srv.URL,
		"site_url":              "https://example.com",
		"app_name":              "Scry",
		"max_completion_tokens": "512",
	})
	require.NoError(t, err)

	result, err := p.Invoke(context.Background(), "openai/gpt-5.2", "generate commit")
	require.NoError(t, err)
	assert.Equal(t, "feat: add openrouter", result)
}

func TestInvoke_ReasoningConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req chatRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.NotNil(t, req.Reasoning)
		assert.Equal(t, "none", req.Reasoning.Effort)
		assert.Equal(t, 256, req.Reasoning.MaxTokens)
		require.NotNil(t, req.Reasoning.Exclude)
		assert.True(t, *req.Reasoning.Exclude)
		require.NotNil(t, req.Reasoning.Enabled)
		assert.False(t, *req.Reasoning.Enabled)

		json.NewEncoder(w).Encode(chatResponse{
			Choices: []chatChoice{{Message: chatMessage{Content: "fix: handle thinking models"}}},
		})
	}))
	defer srv.Close()

	p, err := New(map[string]string{
		"api_key":              "test-key",
		"endpoint":             srv.URL,
		"reasoning_effort":     "none",
		"reasoning_max_tokens": "256",
		"reasoning_exclude":    "true",
		"reasoning_enabled":    "false",
	})
	require.NoError(t, err)

	result, err := p.Invoke(context.Background(), "moonshotai/kimi-k2-thinking", "prompt")
	require.NoError(t, err)
	assert.Equal(t, "fix: handle thinking models", result)
}

func TestInvoke_AuthError(t *testing.T) {
	p := newHTTPErrorProvider(t, http.StatusUnauthorized)

	_, err := p.Invoke(context.Background(), "openai/gpt-5.2", "prompt")
	assert.ErrorIs(t, err, provider.ErrAuth)
}

func TestInvoke_RateLimit(t *testing.T) {
	p := newHTTPErrorProvider(t, http.StatusTooManyRequests)

	_, err := p.Invoke(context.Background(), "openai/gpt-5.2", "prompt")
	assert.ErrorIs(t, err, provider.ErrRateLimit)
}

func TestInvoke_ModelNotFound(t *testing.T) {
	p := newHTTPErrorProvider(t, http.StatusNotFound)

	_, err := p.Invoke(context.Background(), "missing-model", "prompt")
	assert.ErrorIs(t, err, provider.ErrModelNotFound)
}

func TestInvoke_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p, err := New(map[string]string{"api_key": "test-key", "endpoint": srv.URL})
	require.NoError(t, err)

	_, err = p.Invoke(ctx, "openai/gpt-5.2", "prompt")
	assert.True(t, errors.Is(err, provider.ErrTimeout))
}

func TestInvoke_ServerError(t *testing.T) {
	p := newHTTPErrorProvider(t, http.StatusInternalServerError)

	_, err := p.Invoke(context.Background(), "openai/gpt-5.2", "prompt")
	assert.ErrorContains(t, err, "HTTP 500")
}

func TestInvoke_EmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(chatResponse{
			Choices: []chatChoice{{
				Message:      chatMessage{Content: ""},
				FinishReason: "stop",
			}},
		})
	}))
	defer srv.Close()

	p, err := New(map[string]string{"api_key": "test-key", "endpoint": srv.URL})
	require.NoError(t, err)

	_, err = p.Invoke(context.Background(), "moonshotai/kimi-k2", "prompt")
	assert.ErrorContains(t, err, "empty response")
	assert.ErrorContains(t, err, "finish_reason: stop")
}

func TestInvoke_WhitespaceResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(chatResponse{
			Choices: []chatChoice{{Message: chatMessage{Content: "\n\t "}}},
		})
	}))
	defer srv.Close()

	p, err := New(map[string]string{"api_key": "test-key", "endpoint": srv.URL})
	require.NoError(t, err)

	_, err = p.Invoke(context.Background(), "moonshotai/kimi-k2", "prompt")
	assert.ErrorContains(t, err, "empty response")
}

func TestListModels_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/models", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		json.NewEncoder(w).Encode(modelsResponse{
			Data: []modelData{
				{ID: "openai/gpt-5.2", Name: "GPT-5.2"},
				{ID: "anthropic/claude-sonnet-4", Name: "Claude Sonnet 4"},
			},
		})
	}))
	defer srv.Close()

	p, err := New(map[string]string{"api_key": "test-key", "endpoint": srv.URL})
	require.NoError(t, err)

	models, err := p.ListModels(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []provider.Model{
		{ID: "openai/gpt-5.2", Name: "GPT-5.2"},
		{ID: "anthropic/claude-sonnet-4", Name: "Claude Sonnet 4"},
	}, models)
}

func TestListModels_Error(t *testing.T) {
	p := newHTTPErrorProvider(t, http.StatusServiceUnavailable)

	_, err := p.ListModels(context.Background())
	assert.ErrorIs(t, err, provider.ErrRateLimit)
}

func TestMaxTokens_Default(t *testing.T) {
	p, err := New(map[string]string{"api_key": "test-key"})
	require.NoError(t, err)

	assert.Equal(t, 128000, p.MaxTokens("anything"))
}

func TestMaxTokens_FromConfig(t *testing.T) {
	p, err := New(map[string]string{"api_key": "test-key", "max_context_tokens": "200000"})
	require.NoError(t, err)

	assert.Equal(t, 200000, p.MaxTokens("anything"))
}

func TestNew_InvalidMaxContextTokens(t *testing.T) {
	_, err := New(map[string]string{"api_key": "test-key", "max_context_tokens": "nope"})
	assert.ErrorContains(t, err, "invalid max_context_tokens")
}

func TestNew_InvalidReasoningMaxTokens(t *testing.T) {
	_, err := New(map[string]string{"api_key": "test-key", "reasoning_max_tokens": "nope"})
	assert.ErrorContains(t, err, "invalid reasoning_max_tokens")
}

func TestNew_InvalidReasoningExclude(t *testing.T) {
	_, err := New(map[string]string{"api_key": "test-key", "reasoning_exclude": "nope"})
	assert.ErrorContains(t, err, "invalid reasoning_exclude")
}

func TestNew_InvalidReasoningEnabled(t *testing.T) {
	_, err := New(map[string]string{"api_key": "test-key", "reasoning_enabled": "nope"})
	assert.ErrorContains(t, err, "invalid reasoning_enabled")
}

func TestInit_RegistersOpenRouter(t *testing.T) {
	p, err := provider.New("openrouter", map[string]string{"api_key": "test-key"})
	require.NoError(t, err)
	assert.NotNil(t, p)
}

func newHTTPErrorProvider(t *testing.T, statusCode int) provider.Provider {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(struct {
			Error apiError `json:"error"`
		}{Error: apiError{Message: "request failed"}})
	}))
	t.Cleanup(srv.Close)

	p, err := New(map[string]string{"api_key": "test-key", "endpoint": srv.URL})
	require.NoError(t, err)
	return p
}
