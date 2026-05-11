package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/svetozarm/scry/internal/provider"
)

const (
	defaultEndpoint  = "https://openrouter.ai/api/v1"
	defaultMaxTokens = 128000
)

// OpenRouterProvider implements provider.Provider using OpenRouter's API.
type OpenRouterProvider struct {
	endpoint            string
	apiKey              string
	siteURL             string
	appName             string
	maxContextTokens    int
	maxCompletionTokens int
	reasoning           *reasoningConfig
	httpClient          *http.Client
}

// New creates an OpenRouterProvider. It reads optional providerConfig values:
// "endpoint", "api_key", "site_url", "app_name", "max_context_tokens",
// "max_completion_tokens", "reasoning_effort", "reasoning_max_tokens",
// "reasoning_exclude", and "reasoning_enabled". The API key defaults to
// OPENROUTER_API_KEY.
func New(providerConfig map[string]string) (provider.Provider, error) {
	endpoint := providerConfig["endpoint"]
	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	apiKey := providerConfig["api_key"]
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	if apiKey == "" {
		return nil, provider.ErrAuth
	}

	maxContextTokens := defaultMaxTokens
	if configured := providerConfig["max_context_tokens"]; configured != "" {
		parsed, err := strconv.Atoi(configured)
		if err != nil {
			return nil, fmt.Errorf("openrouter: invalid max_context_tokens %q: %w", configured, err)
		}
		maxContextTokens = parsed
	}

	maxCompletionTokens := 0
	if configured := providerConfig["max_completion_tokens"]; configured != "" {
		parsed, err := strconv.Atoi(configured)
		if err != nil {
			return nil, fmt.Errorf("openrouter: invalid max_completion_tokens %q: %w", configured, err)
		}
		maxCompletionTokens = parsed
	}

	reasoning, err := buildReasoningConfig(providerConfig)
	if err != nil {
		return nil, err
	}

	return &OpenRouterProvider{
		endpoint:            strings.TrimRight(endpoint, "/"),
		apiKey:              apiKey,
		siteURL:             providerConfig["site_url"],
		appName:             providerConfig["app_name"],
		maxContextTokens:    maxContextTokens,
		maxCompletionTokens: maxCompletionTokens,
		reasoning:           reasoning,
		httpClient:          &http.Client{},
	}, nil
}

type chatRequest struct {
	Model     string           `json:"model"`
	Messages  []chatMessage    `json:"messages"`
	Stream    bool             `json:"stream"`
	MaxTokens int              `json:"max_tokens,omitempty"`
	Reasoning *reasoningConfig `json:"reasoning,omitempty"`
}

type reasoningConfig struct {
	Effort    string `json:"effort,omitempty"`
	MaxTokens int    `json:"max_tokens,omitempty"`
	Exclude   *bool  `json:"exclude,omitempty"`
	Enabled   *bool  `json:"enabled,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Error   *apiError    `json:"error,omitempty"`
}

type chatChoice struct {
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type modelsResponse struct {
	Data  []modelData `json:"data"`
	Error *apiError   `json:"error,omitempty"`
}

type modelData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type apiError struct {
	Message string `json:"message"`
}

func (p *OpenRouterProvider) Invoke(ctx context.Context, modelID string, prompt string) (string, error) {
	body := chatRequest{
		Model:    modelID,
		Messages: []chatMessage{{Role: "user", Content: prompt}},
		Stream:   false,
	}
	if p.maxCompletionTokens > 0 {
		body.MaxTokens = p.maxCompletionTokens
	}
	if p.reasoning != nil {
		body.Reasoning = p.reasoning
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	p.addHeaders(req)

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

	if err := mapHTTPError(resp.StatusCode, respBody, modelID); err != nil {
		return "", err
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("openrouter: invalid response: %w", err)
	}
	if chatResp.Error != nil && chatResp.Error.Message != "" {
		return "", fmt.Errorf("openrouter: %s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("openrouter: no choices in response")
	}
	content := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	if content == "" {
		finishReason := chatResp.Choices[0].FinishReason
		if finishReason != "" {
			return "", fmt.Errorf("openrouter: empty response (finish_reason: %s)", finishReason)
		}
		return "", fmt.Errorf("openrouter: empty response")
	}

	return content, nil
}

func (p *OpenRouterProvider) ListModels(ctx context.Context) ([]provider.Model, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoint+"/models", nil)
	if err != nil {
		return nil, err
	}
	p.addHeaders(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %v", provider.ErrTimeout, err)
		}
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := mapHTTPError(resp.StatusCode, respBody, ""); err != nil {
		return nil, err
	}

	var modelsResp modelsResponse
	if err := json.Unmarshal(respBody, &modelsResp); err != nil {
		return nil, fmt.Errorf("openrouter: invalid response: %w", err)
	}
	if modelsResp.Error != nil && modelsResp.Error.Message != "" {
		return nil, fmt.Errorf("openrouter: %s", modelsResp.Error.Message)
	}

	models := make([]provider.Model, len(modelsResp.Data))
	for i, m := range modelsResp.Data {
		models[i] = provider.Model{ID: m.ID, Name: m.Name}
	}
	return models, nil
}

func (p *OpenRouterProvider) MaxTokens(_ string) int {
	return p.maxContextTokens
}

func buildReasoningConfig(providerConfig map[string]string) (*reasoningConfig, error) {
	reasoning := &reasoningConfig{}
	configured := false

	if effort := providerConfig["reasoning_effort"]; effort != "" {
		reasoning.Effort = effort
		configured = true
	}

	if raw := providerConfig["reasoning_max_tokens"]; raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("openrouter: invalid reasoning_max_tokens %q: %w", raw, err)
		}
		reasoning.MaxTokens = parsed
		configured = true
	}

	if raw := providerConfig["reasoning_exclude"]; raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("openrouter: invalid reasoning_exclude %q: %w", raw, err)
		}
		reasoning.Exclude = &parsed
		configured = true
	}

	if raw := providerConfig["reasoning_enabled"]; raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("openrouter: invalid reasoning_enabled %q: %w", raw, err)
		}
		reasoning.Enabled = &parsed
		configured = true
	}

	if !configured {
		return nil, nil
	}
	return reasoning, nil
}

func (p *OpenRouterProvider) addHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	if p.siteURL != "" {
		req.Header.Set("HTTP-Referer", p.siteURL)
	}
	if p.appName != "" {
		req.Header.Set("X-OpenRouter-Title", p.appName)
	}
}

func mapHTTPError(statusCode int, body []byte, modelID string) error {
	if statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices {
		return nil
	}

	message := strings.TrimSpace(string(body))
	if parsed := parseAPIError(body); parsed != "" {
		message = parsed
	}

	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusPaymentRequired:
		return fmt.Errorf("%w: %s", provider.ErrAuth, message)
	case http.StatusNotFound:
		if modelID != "" {
			return fmt.Errorf("%w: model %q", provider.ErrModelNotFound, modelID)
		}
		return fmt.Errorf("%w: %s", provider.ErrModelNotFound, message)
	case http.StatusRequestTimeout:
		return fmt.Errorf("%w: %s", provider.ErrTimeout, message)
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable:
		return fmt.Errorf("%w: %s", provider.ErrRateLimit, message)
	default:
		return fmt.Errorf("openrouter: HTTP %d: %s", statusCode, message)
	}
}

func parseAPIError(body []byte) string {
	var resp struct {
		Error *apiError `json:"error"`
	}
	if err := json.Unmarshal(body, &resp); err != nil || resp.Error == nil {
		return ""
	}
	return resp.Error.Message
}

func init() {
	provider.Register("openrouter", New)
}
