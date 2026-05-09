package bedrock

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/svetozarm/scry/internal/provider"
)

// converseAPI abstracts the Converse call for testing.
type converseAPI interface {
	Converse(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error)
}

// listModelsAPI abstracts the ListFoundationModels call for testing.
type listModelsAPI interface {
	ListFoundationModels(ctx context.Context, params *bedrock.ListFoundationModelsInput, optFns ...func(*bedrock.Options)) (*bedrock.ListFoundationModelsOutput, error)
}

// BedrockProvider implements provider.Provider using Amazon Bedrock.
type BedrockProvider struct {
	runtimeClient converseAPI
	bedrockClient listModelsAPI
	region        string
}

// New creates a BedrockProvider. It reads "region" from providerConfig,
// defaulting to "us-east-1".
func New(providerConfig map[string]string) (provider.Provider, error) {
	region := providerConfig["region"]
	if region == "" {
		region = "us-east-1"
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion(region))
	if err != nil {
		return nil, err
	}

	return &BedrockProvider{
		runtimeClient: bedrockruntime.NewFromConfig(cfg),
		bedrockClient: bedrock.NewFromConfig(cfg),
		region:        region,
	}, nil
}

// Invoke calls the Bedrock Converse API and returns the model's text response.
func (p *BedrockProvider) Invoke(ctx context.Context, modelID string, prompt string) (string, error) {
	out, err := p.runtimeClient.Converse(ctx, &bedrockruntime.ConverseInput{
		ModelId: &modelID,
		Messages: []types.Message{
			{
				Role:    types.ConversationRoleUser,
				Content: []types.ContentBlock{&types.ContentBlockMemberText{Value: prompt}},
			},
		},
	})
	if err != nil {
		return "", mapError(err)
	}

	msg, ok := out.Output.(*types.ConverseOutputMemberMessage)
	if !ok || len(msg.Value.Content) == 0 {
		return "", fmt.Errorf("unexpected response format")
	}

	for _, block := range msg.Value.Content {
		if text, ok := block.(*types.ContentBlockMemberText); ok {
			return text.Value, nil
		}
	}

	return "", fmt.Errorf("no text content block in response")
}

// mapError converts AWS SDK errors to typed provider errors.
func mapError(err error) error {
	var accessDenied *types.AccessDeniedException
	if errors.As(err, &accessDenied) {
		return fmt.Errorf("%w: %v", provider.ErrAuth, err)
	}

	var throttling *types.ThrottlingException
	if errors.As(err, &throttling) {
		return fmt.Errorf("%w: %v", provider.ErrRateLimit, err)
	}

	var unavailable *types.ServiceUnavailableException
	if errors.As(err, &unavailable) {
		return fmt.Errorf("%w: %v", provider.ErrRateLimit, err)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%w: %v", provider.ErrTimeout, err)
	}

	var validation *types.ValidationException
	if errors.As(err, &validation) && validation.Message != nil && strings.Contains(strings.ToLower(*validation.Message), "model") {
		return fmt.Errorf("%w: %v", provider.ErrModelNotFound, err)
	}

	if strings.Contains(err.Error(), "credential") {
		return fmt.Errorf("%w: %v", provider.ErrAuth, err)
	}

	return err
}

func (p *BedrockProvider) ListModels(ctx context.Context) ([]provider.Model, error) {
	out, err := p.bedrockClient.ListFoundationModels(ctx, &bedrock.ListFoundationModelsInput{})
	if err != nil {
		return nil, mapError(err)
	}

	models := make([]provider.Model, len(out.ModelSummaries))
	for i, s := range out.ModelSummaries {
		models[i] = provider.Model{
			ID:   aws.ToString(s.ModelId),
			Name: aws.ToString(s.ModelName),
		}
	}
	return models, nil
}

var contextWindows = map[string]int{
	"amazon.nova-lite-v1:0":  300000,
	"amazon.nova-micro-v1:0": 128000,
	"amazon.nova-pro-v1:0":   300000,
}

const defaultMaxTokens = 128000

func (p *BedrockProvider) MaxTokens(modelID string) int {
	if v, ok := contextWindows[modelID]; ok {
		return v
	}
	return defaultMaxTokens
}

func init() {
	provider.Register("bedrock", New)
}
