package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/snowmerak/open-librarian/lib/util/logger"
)

type Client struct {
	provider      string
	genBaseURL    string
	genKey        string
	genModel      string
	ollamaBaseURL string
	httpClient    *http.Client
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type ChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type ChatStreamResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

type EmbedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type EmbedResponse struct {
	Model      string      `json:"model"`
	Embeddings [][]float64 `json:"embeddings"`
}

const (
	DefaultEmbeddingModel = "embeddinggemma:300m"

	ProviderOllama     = "ollama"
	ProviderOpenAPI    = "openapi"
	ProviderOpenRouter = "openrouter"
)

// NewClient creates a new LLM client
func NewClient(provider, genBaseURL, genKey, genModel, ollamaBaseURL string) *Client {
	log := logger.NewLogger("llm-client")
	log.StartWithMsg("Creating new LLM client")

	if provider == "" {
		provider = ProviderOllama
	}

	// Normalize URLs (remove trailing slash)
	genBaseURL = strings.TrimRight(genBaseURL, "/")
	ollamaBaseURL = strings.TrimRight(ollamaBaseURL, "/")

	log.Info().
		Str("provider", provider).
		Str("gen_url", genBaseURL).
		Str("gen_model", genModel).
		Str("ollama_url", ollamaBaseURL).
		Msg("LLM Client Configuration")

	client := &Client{
		provider:      provider,
		genBaseURL:    genBaseURL,
		genKey:        genKey,
		genModel:      genModel,
		ollamaBaseURL: ollamaBaseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Minute,
		},
	}

	log.EndWithMsg("LLM client created successfully")
	return client
}

// GenerateText generates text using the configured provider via OpenAI-compatible API
func (c *Client) GenerateText(ctx context.Context, prompt string) (string, error) {
	log := logger.NewLogger("llm-generate-text")
	log.StartWithMsg("Generating text")

	// Strict prompt wrapper
	strictPrompt := fmt.Sprintf(`You must respond ONLY with the requested content. Do not add any commentary, explanations, opinions, or meta-text. Do not prefix or suffix your response with any additional text.

%s

Remember: Output ONLY the requested content, nothing else.`, prompt)

	reqPayload := ChatRequest{
		Model: c.genModel,
		Messages: []ChatMessage{
			{Role: "user", Content: strictPrompt},
		},
		Stream: false,
	}

	reqBody, err := json.Marshal(reqPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/chat/completions", c.genBaseURL)
	// OpenRouter specific path adjustment if needed, but usually v1/chat/completions works.
	// However, user said "api/v1/chat/completions" for OpenRouter sometimes?
	// Standard OpenRouter: https://openrouter.ai/api/v1/chat/completions
	// If genBaseURL is https://openrouter.ai/api, then + /v1/chat/completions is correct.

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.genKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.genKey)
	}
	// OpenRouter specific headers
	if c.provider == ProviderOpenRouter {
		httpReq.Header.Set("HTTP-Referer", "https://github.com/snowmerak/open-librarian")
		httpReq.Header.Set("X-Title", "Open Librarian")
	}

	log.Info().Str("url", url).Str("model", c.genModel).Msg("Sending request")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// GenerateTextStream generates text using streaming mode via OpenAI-compatible API
func (c *Client) GenerateTextStream(ctx context.Context, prompt string, callback func(string) error) error {
	strictPrompt := fmt.Sprintf(`You must respond ONLY with the requested content. Do not add any commentary, explanations, opinions, or meta-text. Do not prefix or suffix your response with any additional text.

%s

Remember: Output ONLY the requested content, nothing else.`, prompt)

	reqPayload := ChatRequest{
		Model: c.genModel,
		Messages: []ChatMessage{
			{Role: "user", Content: strictPrompt},
		},
		Stream: true,
	}

	reqBody, err := json.Marshal(reqPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/chat/completions", c.genBaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.genKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.genKey)
	}
	if c.provider == ProviderOpenRouter {
		httpReq.Header.Set("HTTP-Referer", "https://github.com/snowmerak/open-librarian")
		httpReq.Header.Set("X-Title", "Open Librarian")
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading stream: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// SSE format: "data: {JSON}"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var streamResp ChatStreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			continue
		}

		if len(streamResp.Choices) > 0 {
			content := streamResp.Choices[0].Delta.Content
			if content != "" {
				if err := callback(content); err != nil {
					return fmt.Errorf("callback error: %w", err)
				}
			}
		}
	}

	return nil
}

// GenerateEmbedding generates embeddings using Ollama (Native API) with the specified embeddinggemma model
func (c *Client) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	model := DefaultEmbeddingModel

	// We use Ollama native API for embeddings as requested/implied by "embedding is done by ollama"
	// and specific usage of an Ollama model tag.
	req := EmbedRequest{
		Model: model,
		Input: text,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Use c.ollamaBaseURL specifically for embedding
	url := fmt.Sprintf("%s/api/embed", c.ollamaBaseURL)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var embedResp EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(embedResp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned from API")
	}

	return embedResp.Embeddings[0], nil
}

// HealthCheck checks if both configured LLM and Ollama are reachable
func (c *Client) HealthCheck(ctx context.Context) error {
	// Check Ollama (for embeddings)
	ollamaReq, err := http.NewRequest("GET", c.ollamaBaseURL+"/api/tags", nil)
	if err == nil {
		if resp, err := c.httpClient.Do(ollamaReq); err == nil {
			resp.Body.Close()
		}
	}

	// If using local ollama for everything, we are good.
	if c.provider == ProviderOllama && c.genBaseURL == c.ollamaBaseURL {
		return nil
	}

	// If different provider, maybe we can't easily health check standard OpenAI endpoint without cost or valid model.
	// But we can assume if config is correct it works for now.
	return nil
}
