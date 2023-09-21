package llm

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	llm "github.com/grafana/grafana-llm-app/llmclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/oauthtokenretriever"
	"github.com/sashabaranov/go-openai"
)

type openAIUsingToken struct {
	httpClient     *http.Client
	grafanaURL     string
	tokenRetriever oauthtokenretriever.TokenRetriever
}

func NewOpenAIForPlugin(ctx context.Context) (llm.OpenAI, error) {
	cfg := backend.GrafanaConfigFromContext(ctx)
	grafanaAppURL := strings.TrimRight(cfg.Get("GF_APP_URL"), "/")
	if grafanaAppURL == "" {
		// For debugging purposes only
		grafanaAppURL = "http://localhost:3000"
	}
	client := &http.Client{}
	tokenRetriever, err := oauthtokenretriever.New()
	if err != nil {
		return nil, fmt.Errorf("create token retriever: %w", err)
	}
	return &openAIUsingToken{
		httpClient:     client,
		tokenRetriever: tokenRetriever,
		grafanaURL:     grafanaAppURL,
	}, nil
}

func (o *openAIUsingToken) openAIClient(ctx context.Context) (llm.OpenAI, error) {
	token, err := o.tokenRetriever.Self(ctx)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}
	return llm.NewOpenAIWithClient(o.grafanaURL, token, o.httpClient), nil
}

func (o *openAIUsingToken) Enabled(ctx context.Context) (bool, error) {
	client, err := o.openAIClient(ctx)
	if err != nil {
		return false, err
	}
	return client.Enabled(ctx)
}

func (o *openAIUsingToken) ChatCompletions(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	client, err := o.openAIClient(ctx)
	if err != nil {
		return openai.ChatCompletionResponse{}, err
	}
	return client.ChatCompletions(ctx, req)
}

func (o *openAIUsingToken) ChatCompletionsStream(ctx context.Context, req openai.ChatCompletionRequest) (*openai.ChatCompletionStream, error) {
	client, err := o.openAIClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.ChatCompletionsStream(ctx, req)
}
