package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/svetozarm/scry/internal/provider"
)

const defaultEndpoint = "http://localhost:11434"

// OllamaProvider implements provider.Provider using a local Ollama instance.
type OllamaProvider struct {
	endpoint   string
	httpClient *http.Client
}

// New creates an OllamaProvider. It reads "endpoint" from providerConfig,
// defaulting to "http://localhost:11434".
func New(providerConfig map[string]string) (provider.Provider, error) {
	endpoint := providerConfig["endpoint"]
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	return &OllamaProvider{
		endpoint:   strings.TrimRight(endpoint, "/"),
		httpClient: &http.Client{},
	}, nil
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Message chatMessage `json:"message"`
	Error   string      `json:"error,omitempty"`
}

type tagResponse struct {
	Models []tagModel `json:"models"`
}

type tagModel struct {
	Name string `json:"name"`
}

func (p *OllamaProvider) Invoke(ctx context.Context, modelID string, prompt string) (string, error) {
	body, err := json.Marshal(chatRequest{
		Model:    modelID,
		Messages: []chatMessage{{Role: "user", Content: prompt}},
		Stream:   false,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("%w: %v", provider.ErrTimeout, err)
		}
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("%w: model %q", provider.ErrModelNotFound, modelID)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama: HTTP %d: %s", resp.StatusCode, respBody)
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("ollama: invalid response: %w", err)
	}
	if chatResp.Error != "" {
		return "", fmt.Errorf("ollama: %s", chatResp.Error)
	}

	return chatResp.Message.Content, nil
}

func (p *OllamaProvider) ListModels(ctx context.Context) ([]provider.Model, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoint+"/api/tags", nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %v", provider.ErrTimeout, err)
		}
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama: HTTP %d listing models", resp.StatusCode)
	}

	var tags tagResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, fmt.Errorf("ollama: invalid response: %w", err)
	}

	models := make([]provider.Model, len(tags.Models))
	for i, m := range tags.Models {
		models[i] = provider.Model{ID: m.Name, Name: m.Name}
	}
	return models, nil
}

func (p *OllamaProvider) MaxTokens(_ string) int {
	return 128000
}

func init() {
	provider.Register("ollama", New)
}
