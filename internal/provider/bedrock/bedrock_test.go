package bedrock

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	brtypes "github.com/aws/aws-sdk-go-v2/service/bedrock/types"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/svetozarm/scry/internal/provider"
)

// mockConverseAPI implements converseAPI for testing.
type mockConverseAPI struct {
	output *bedrockruntime.ConverseOutput
	err    error
}

func (m *mockConverseAPI) Converse(_ context.Context, _ *bedrockruntime.ConverseInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
	return m.output, m.err
}

// mockListModelsAPI implements listModelsAPI for testing.
type mockListModelsAPI struct {
	output *bedrock.ListFoundationModelsOutput
	err    error
}

func (m *mockListModelsAPI) ListFoundationModels(_ context.Context, _ *bedrock.ListFoundationModelsInput, _ ...func(*bedrock.Options)) (*bedrock.ListFoundationModelsOutput, error) {
	return m.output, m.err
}

func newTestProvider(mock *mockConverseAPI) *BedrockProvider {
	return &BedrockProvider{runtimeClient: mock, region: "us-east-1"}
}

func TestInvoke_Success(t *testing.T) {
	mock := &mockConverseAPI{
		output: &bedrockruntime.ConverseOutput{
			Output: &types.ConverseOutputMemberMessage{
				Value: types.Message{
					Role:    types.ConversationRoleAssistant,
					Content: []types.ContentBlock{&types.ContentBlockMemberText{Value: "fix: correct typo"}},
				},
			},
		},
	}

	p := newTestProvider(mock)
	result, err := p.Invoke(context.Background(), "amazon.nova-micro-v1:0", "generate a commit message")

	require.NoError(t, err)
	assert.Equal(t, "fix: correct typo", result)
}

func TestInvoke_Error(t *testing.T) {
	mock := &mockConverseAPI{err: fmt.Errorf("some AWS error")}

	p := newTestProvider(mock)
	_, err := p.Invoke(context.Background(), "amazon.nova-micro-v1:0", "prompt")

	require.Error(t, err)
}

func TestMapError_AccessDenied(t *testing.T) {
	mock := &mockConverseAPI{err: &types.AccessDeniedException{Message: aws.String("not allowed")}}
	p := newTestProvider(mock)
	_, err := p.Invoke(context.Background(), "m", "p")
	assert.ErrorIs(t, err, provider.ErrAuth)
}

func TestMapError_Throttling(t *testing.T) {
	mock := &mockConverseAPI{err: &types.ThrottlingException{Message: aws.String("slow down")}}
	p := newTestProvider(mock)
	_, err := p.Invoke(context.Background(), "m", "p")
	assert.ErrorIs(t, err, provider.ErrRateLimit)
}

func TestMapError_ServiceUnavailable(t *testing.T) {
	mock := &mockConverseAPI{err: &types.ServiceUnavailableException{Message: aws.String("unavailable")}}
	p := newTestProvider(mock)
	_, err := p.Invoke(context.Background(), "m", "p")
	assert.ErrorIs(t, err, provider.ErrRateLimit)
}

func TestMapError_Timeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	mock := &mockConverseAPI{err: context.DeadlineExceeded}
	p := newTestProvider(mock)
	_, err := p.Invoke(ctx, "m", "p")
	assert.ErrorIs(t, err, provider.ErrTimeout)
}

func TestMapError_ModelNotFound(t *testing.T) {
	mock := &mockConverseAPI{err: &types.ValidationException{Message: aws.String("Could not resolve the foundation model")}}
	p := newTestProvider(mock)
	_, err := p.Invoke(context.Background(), "m", "p")
	assert.ErrorIs(t, err, provider.ErrModelNotFound)
}

func TestMapError_ValidationNotModel(t *testing.T) {
	mock := &mockConverseAPI{err: &types.ValidationException{Message: aws.String("invalid parameter")}}
	p := newTestProvider(mock)
	_, err := p.Invoke(context.Background(), "m", "p")
	assert.NotErrorIs(t, err, provider.ErrModelNotFound)
}

func TestMapError_CredentialError(t *testing.T) {
	mock := &mockConverseAPI{err: fmt.Errorf("failed to retrieve credentials: no credential providers")}
	p := newTestProvider(mock)
	_, err := p.Invoke(context.Background(), "m", "p")
	assert.ErrorIs(t, err, provider.ErrAuth)
}

func TestMapError_UnknownError(t *testing.T) {
	orig := fmt.Errorf("something unexpected")
	mock := &mockConverseAPI{err: orig}
	p := newTestProvider(mock)
	_, err := p.Invoke(context.Background(), "m", "p")
	assert.NotErrorIs(t, err, provider.ErrAuth)
	assert.NotErrorIs(t, err, provider.ErrRateLimit)
	assert.NotErrorIs(t, err, provider.ErrTimeout)
	assert.NotErrorIs(t, err, provider.ErrModelNotFound)
	assert.EqualError(t, err, "something unexpected")
}

func TestNew_DefaultRegion(t *testing.T) {
	p, err := New(nil)
	require.NoError(t, err)

	bp := p.(*BedrockProvider)
	assert.Equal(t, "us-east-1", bp.region)
}

func TestNew_RegionFromConfig(t *testing.T) {
	p, err := New(map[string]string{"region": "eu-west-1"})
	require.NoError(t, err)

	bp := p.(*BedrockProvider)
	assert.Equal(t, "eu-west-1", bp.region)
}

func TestNew_ImplementsProvider(t *testing.T) {
	p, err := New(nil)
	require.NoError(t, err)

	var _ provider.Provider = p
}

func TestInit_RegistersBedrock(t *testing.T) {
	p, err := provider.New("bedrock", nil)
	require.NoError(t, err)
	assert.NotNil(t, p)
}

func TestListModels_Success(t *testing.T) {
	mock := &mockListModelsAPI{
		output: &bedrock.ListFoundationModelsOutput{
			ModelSummaries: []brtypes.FoundationModelSummary{
				{ModelId: aws.String("amazon.nova-micro-v1:0"), ModelName: aws.String("Nova Micro")},
				{ModelId: aws.String("amazon.nova-lite-v1:0"), ModelName: aws.String("Nova Lite")},
			},
		},
	}

	p := &BedrockProvider{bedrockClient: mock, region: "us-east-1"}
	models, err := p.ListModels(context.Background())

	require.NoError(t, err)
	require.Len(t, models, 2)
	assert.Equal(t, "amazon.nova-micro-v1:0", models[0].ID)
	assert.Equal(t, "Nova Micro", models[0].Name)
	assert.Equal(t, "amazon.nova-lite-v1:0", models[1].ID)
	assert.Equal(t, "Nova Lite", models[1].Name)
}

func TestListModels_Error(t *testing.T) {
	mock := &mockListModelsAPI{err: fmt.Errorf("access denied")}

	p := &BedrockProvider{bedrockClient: mock, region: "us-east-1"}
	_, err := p.ListModels(context.Background())

	require.Error(t, err)
}

func TestMaxTokens_KnownModels(t *testing.T) {
	p := &BedrockProvider{}

	assert.Equal(t, 300000, p.MaxTokens("amazon.nova-lite-v1:0"))
	assert.Equal(t, 128000, p.MaxTokens("amazon.nova-micro-v1:0"))
	assert.Equal(t, 300000, p.MaxTokens("amazon.nova-pro-v1:0"))
}

func TestMaxTokens_UnknownModel(t *testing.T) {
	p := &BedrockProvider{}

	assert.Equal(t, 128000, p.MaxTokens("some-unknown-model"))
}

func TestListModels_Empty(t *testing.T) {
	mock := &mockListModelsAPI{
		output: &bedrock.ListFoundationModelsOutput{
			ModelSummaries: []brtypes.FoundationModelSummary{},
		},
	}

	p := &BedrockProvider{bedrockClient: mock, region: "us-east-1"}
	models, err := p.ListModels(context.Background())

	require.NoError(t, err)
	assert.Empty(t, models)
}
