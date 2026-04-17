package ollama

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
	p, err := New(map[string]string{})
	require.NoError(t, err)
	assert.Equal(t, defaultEndpoint, p.(*OllamaProvider).endpoint)
}

func TestNew_CustomEndpoint(t *testing.T) {
	p, err := New(map[string]string{"endpoint": "http://myhost:1234"})
	require.NoError(t, err)
	assert.Equal(t, "http://myhost:1234", p.(*OllamaProvider).endpoint)
}

func TestNew_TrailingSlashTrimmed(t *testing.T) {
	p, err := New(map[string]string{"endpoint": "http://myhost:1234/"})
	require.NoError(t, err)
	assert.Equal(t, "http://myhost:1234", p.(*OllamaProvider).endpoint)
}

func TestInvoke_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/chat", r.URL.Path)
		var req chatRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "llama3", req.Model)
		assert.False(t, req.Stream)
		json.NewEncoder(w).Encode(chatResponse{Message: chatMessage{Content: "fix: typo"}})
	}))
	defer srv.Close()

	p, _ := New(map[string]string{"endpoint": srv.URL})
	result, err := p.Invoke(context.Background(), "llama3", "generate commit")
	require.NoError(t, err)
	assert.Equal(t, "fix: typo", result)
}

func TestInvoke_ModelNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p, _ := New(map[string]string{"endpoint": srv.URL})
	_, err := p.Invoke(context.Background(), "nonexistent", "prompt")
	assert.True(t, errors.Is(err, provider.ErrModelNotFound))
}

func TestInvoke_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// never respond
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	p, _ := New(map[string]string{"endpoint": srv.URL})
	_, err := p.Invoke(ctx, "llama3", "prompt")
	assert.True(t, errors.Is(err, provider.ErrTimeout))
}

func TestInvoke_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	p, _ := New(map[string]string{"endpoint": srv.URL})
	_, err := p.Invoke(context.Background(), "llama3", "prompt")
	assert.ErrorContains(t, err, "HTTP 500")
}

func TestListModels_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/tags", r.URL.Path)
		json.NewEncoder(w).Encode(tagResponse{
			Models: []tagModel{{Name: "llama3"}, {Name: "mistral"}},
		})
	}))
	defer srv.Close()

	p, _ := New(map[string]string{"endpoint": srv.URL})
	models, err := p.ListModels(context.Background())
	require.NoError(t, err)
	assert.Len(t, models, 2)
	assert.Equal(t, "llama3", models[0].ID)
	assert.Equal(t, "mistral", models[1].ID)
}

func TestListModels_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	p, _ := New(map[string]string{"endpoint": srv.URL})
	_, err := p.ListModels(context.Background())
	assert.Error(t, err)
}

func TestMaxTokens(t *testing.T) {
	p, _ := New(map[string]string{})
	assert.Equal(t, 128000, p.MaxTokens("anything"))
}

func TestInit_RegistersOllama(t *testing.T) {
	p, err := provider.New("ollama", map[string]string{})
	require.NoError(t, err)
	assert.NotNil(t, p)
}
