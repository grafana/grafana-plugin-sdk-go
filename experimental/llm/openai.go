package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sashabaranov/go-openai"
)

type OpenAI struct {
	httpClient *http.Client
	client     *openai.Client

	grafanaURL, grafanaAPIKey string
}

func NewOpenAI(grafanaURL, grafanaAPIKey string) *OpenAI {
	httpClient := &http.Client{}
	url := strings.TrimRight(grafanaURL, "/") + "/api/plugins/grafana-llm-app/resources/openai/v1"
	cfg := openai.DefaultConfig(grafanaAPIKey)
	cfg.BaseURL = url
	cfg.HTTPClient = httpClient
	client := openai.NewClientWithConfig(cfg)
	return &OpenAI{
		httpClient:    httpClient,
		client:        client,
		grafanaURL:    grafanaURL,
		grafanaAPIKey: grafanaAPIKey,
	}
}

type pluginSettings struct {
	Enabled          bool `json:"enabled"`
	SecureJSONFields struct {
		OpenAIKey bool `json:"openAIKey"`
	} `json:"secureJsonFields"`
}

func (o *OpenAI) Enabled(ctx context.Context) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", o.grafanaURL+"/api/plugins/grafana-llm-app/settings", nil)
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+o.grafanaAPIKey)
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("make request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return false, nil
	}
	var settings pluginSettings
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return false, fmt.Errorf("unmarshal response: %w", err)
	}
	return settings.Enabled && settings.SecureJSONFields.OpenAIKey, nil
}

func (o *OpenAI) ChatCompletions(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	return o.client.CreateChatCompletion(ctx, req)
}

func (o *OpenAI) StreamChatCompletions(ctx context.Context, req openai.ChatCompletionRequest) (*openai.ChatCompletionStream, error) {
	return o.client.CreateChatCompletionStream(ctx, req)
}
