# Module: Provider Abstraction (`internal/provider/`)

## Responsibility

Define the common interface for LLM backends, classify provider errors into typed errors, and provide a factory to instantiate the correct provider from config.

## Package Structure

```
internal/provider/
  provider.go           # Provider interface, Model struct, factory function
  errors.go             # Typed error definitions
  bedrock/
    bedrock.go          # Bedrock implementation of Provider
    bedrock_test.go     # Unit tests (mocked AWS SDK clients)
  ollama/
    ollama.go           # Ollama implementation of Provider
    ollama_test.go      # Unit tests (httptest server)
  testprovider/
    testprovider.go     # Test helper provider for integration tests
```

## Provider Interface

```go
type Provider interface {
    // Invoke sends a prompt to the LLM and returns the generated text.
    Invoke(ctx context.Context, modelID string, prompt string) (string, error)

    // ListModels returns the models available on this provider.
    ListModels(ctx context.Context) ([]Model, error)

    // MaxTokens returns the context window size for the given model.
    MaxTokens(modelID string) int
}

type Model struct {
    ID   string
    Name string
}
```

## Factory

```go
// New creates a Provider by looking up the named factory in the registry.
func New(name string, config map[string]string) (Provider, error)
```

Internally uses a registry map populated via `Register()` calls from provider `init()` functions:

```go
var registry = map[string]FactoryFunc{}

// Register adds a provider factory under the given name.
func Register(name string, factory FactoryFunc)
```

Both the Bedrock and Ollama providers register themselves in their `init()` functions. New providers are added by calling `Register` — no modification to existing code (Open/Closed Principle).

## Error Types

```go
var (
    ErrAuth            // Authentication or authorisation failure
    ErrRateLimit       // Throttling / rate limit
    ErrTimeout         // Request timed out
    ErrModelNotFound   // Model ID not recognised by provider
    ErrUnknownProvider // Provider name not in registry
)
```

Each provider implementation is responsible for mapping SDK-specific errors to these types using `fmt.Errorf("%w: %v", ...)` wrapping.

## Bedrock Implementation

- Uses `github.com/aws/aws-sdk-go-v2` with `config.LoadDefaultConfig()`.
- `region` is read from `providerConfig["region"]`, defaulting to `"us-east-1"`.
- `Invoke` calls the Bedrock **Converse** API (`bedrockruntime.Converse`) with the prompt as a user message.
- `ListModels` calls `bedrock.ListFoundationModels`.
- Uses internal interfaces (`converseAPI`, `listModelsAPI`) for testability.
- Error mapping:
  - `AccessDeniedException`, credential errors → `ErrAuth`
  - `ThrottlingException` → `ErrRateLimit`
  - `ServiceUnavailableException` → `ErrRateLimit`
  - Context deadline exceeded → `ErrTimeout`
  - `ValidationException` with "model" in message → `ErrModelNotFound`
  - Error message containing "credential" → `ErrAuth`

## MaxTokens

Hardcoded map of known model context sizes:

```go
var contextWindows = map[string]int{
    "amazon.nova-lite-v1:0":  300000,
    "amazon.nova-micro-v1:0": 128000,
    "amazon.nova-pro-v1:0":   300000,
}
```

Falls back to 128,000 for unknown models.

## Ollama Implementation

- Uses the Ollama REST API over HTTP.
- `endpoint` is read from `providerConfig["endpoint"]`, defaulting to `http://localhost:11434`.
- `Invoke` calls `/api/chat` with `stream: false` and returns the assistant message content.
- `ListModels` calls `/api/tags` and returns locally available models.
- Error mapping:
  - HTTP 404 → `ErrModelNotFound`
  - Cancelled context → `ErrTimeout`
- `MaxTokens` returns 128,000 for all models.

## Testing Strategy

- Mock the AWS SDK clients using interfaces (`converseAPI`, `listModelsAPI`).
- Test error classification for each SDK error type.
- Test that `providerConfig` values are correctly applied.
- Ollama tests use `net/http/httptest` server.

## Dependencies

- `github.com/aws/aws-sdk-go-v2/config`
- `github.com/aws/aws-sdk-go-v2/service/bedrockruntime` (Converse API)
- `github.com/aws/aws-sdk-go-v2/service/bedrock` (ListFoundationModels)
