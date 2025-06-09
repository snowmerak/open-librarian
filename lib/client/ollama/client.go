package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/snowmerak/open-librarian/lib/util/logger"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type GenerateResponse struct {
	Model     string    `json:"model"`
	Response  string    `json:"response"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
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
	DefaultTextModel      = "gemma3:12b"
	DefaultEmbeddingModel = "paraphrase-multilingual"
	DefaultBaseURL        = "http://localhost:11434"
)

// NewClient creates a new Ollama client
func NewClient(baseURL string) *Client {
	logger := logger.NewLogger("ollama-client")
	logger.StartWithMsg("Creating new Ollama client")

	if baseURL == "" {
		baseURL = DefaultBaseURL
		logger.Info().Str("base_url", baseURL).Msg("Using default base URL")
	} else {
		logger.Info().Str("base_url", baseURL).Msg("Using provided base URL")
	}

	client := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Minute, // Increased timeout significantly
		},
	}

	logger.EndWithMsg("Ollama client created successfully")
	return client
}

// GenerateText generates text using the specified model (default: gemma2:14b)
func (c *Client) GenerateText(ctx context.Context, prompt string) (string, error) {
	logger := logger.NewLogger("ollama-generate-text")
	logger.StartWithMsg("Generating text with default model")
	logger.Info().Str("model", DefaultTextModel).Msg("Using default text model")

	result, err := c.GenerateTextWithModel(ctx, DefaultTextModel, prompt)
	if err != nil {
		logger.EndWithError(err)
		return "", err
	}

	logger.EndWithMsg("Text generation completed")
	return result, nil
}

// GenerateTextWithModel generates text using a specific model
func (c *Client) GenerateTextWithModel(ctx context.Context, model, prompt string) (string, error) {
	logger := logger.NewLogger("ollama-generate-text-with-model")
	logger.StartWithMsg("Generating text with specific model")
	logger.Info().Str("model", model).Int("prompt_length", len(prompt)).Msg("Text generation request details")

	// Add strict output formatting instructions to prevent LLM from adding commentary
	strictPrompt := fmt.Sprintf(`You must respond ONLY with the requested content. Do not add any commentary, explanations, opinions, or meta-text. Do not prefix or suffix your response with any additional text.

%s

Remember: Output ONLY the requested content, nothing else.`, prompt)

	req := GenerateRequest{
		Model:  model,
		Prompt: strictPrompt,
		Stream: false,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		logger.EndWithError(fmt.Errorf("failed to marshal request: %w", err))
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/api/generate", bytes.NewBuffer(reqBody))
	if err != nil {
		logger.EndWithError(fmt.Errorf("failed to create request: %w", err))
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	logger.Info().Str("url", c.baseURL+"/api/generate").Msg("Sending request to Ollama API")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		logger.EndWithError(fmt.Errorf("failed to send request: %w", err))
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		logger.Error().Int("status_code", resp.StatusCode).Msg("API request failed")
		logger.EndWithError(err)
		return "", err
	}

	var genResp GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		logger.EndWithError(fmt.Errorf("failed to decode response: %w", err))
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	logger.Info().Int("response_length", len(genResp.Response)).Msg("Text generation successful")
	logger.EndWithMsg("Text generation completed successfully")
	return genResp.Response, nil
}

// GenerateTextStream generates text using streaming mode
func (c *Client) GenerateTextStream(ctx context.Context, prompt string, callback func(string) error) error {
	return c.GenerateTextStreamWithModel(ctx, DefaultTextModel, prompt, callback)
}

// GenerateTextStreamWithModel generates text using a specific model in streaming mode
func (c *Client) GenerateTextStreamWithModel(ctx context.Context, model, prompt string, callback func(string) error) error {
	// Add strict output formatting instructions to prevent LLM from adding commentary
	strictPrompt := fmt.Sprintf(`You must respond ONLY with the requested content. Do not add any commentary, explanations, opinions, or meta-text. Do not prefix or suffix your response with any additional text.

%s

Remember: Output ONLY the requested content, nothing else.`, prompt)

	req := GenerateRequest{
		Model:  model,
		Prompt: strictPrompt,
		Stream: true,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/api/generate", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	decoder := json.NewDecoder(resp.Body)
	for {
		var genResp GenerateResponse
		if err := decoder.Decode(&genResp); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode streaming response: %w", err)
		}

		// Call the callback with the response chunk
		if err := callback(genResp.Response); err != nil {
			return fmt.Errorf("callback error: %w", err)
		}

		// If done, break the loop
		if genResp.Done {
			break
		}
	}

	return nil
}

// GenerateEmbedding generates embeddings using the paraphrase-multilingual model
func (c *Client) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	logger := logger.NewLogger("ollama-generate-embedding")
	logger.StartWithMsg("Generating embedding with default model")
	logger.Info().Str("model", DefaultEmbeddingModel).Msg("Using default embedding model")

	result, err := c.GenerateEmbeddingWithModel(ctx, DefaultEmbeddingModel, text)
	if err != nil {
		logger.EndWithError(err)
		return nil, err
	}

	logger.EndWithMsg("Embedding generation completed")
	return result, nil
}

// GenerateEmbeddingWithModel generates embeddings using a specific model
func (c *Client) GenerateEmbeddingWithModel(ctx context.Context, model, text string) ([]float64, error) {
	logger := logger.NewLogger("ollama-generate-embedding-with-model")
	logger.StartWithMsg("Generating embedding with specific model")
	logger.Info().Str("model", model).Int("text_length", len(text)).Msg("Embedding generation request details")

	req := EmbedRequest{
		Model: model,
		Input: text,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		logger.EndWithError(fmt.Errorf("failed to marshal request: %w", err))
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/api/embed", bytes.NewBuffer(reqBody))
	if err != nil {
		logger.EndWithError(fmt.Errorf("failed to create request: %w", err))
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	logger.Info().Str("url", c.baseURL+"/api/embed").Msg("Sending embedding request to Ollama API")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		logger.EndWithError(fmt.Errorf("failed to send request: %w", err))
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		logger.Error().Int("status_code", resp.StatusCode).Msg("Embedding API request failed")
		logger.EndWithError(err)
		return nil, err
	}

	var embedResp EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		logger.EndWithError(fmt.Errorf("failed to decode response: %w", err))
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Ollama returns embeddings as array of arrays, we want the first one
	if len(embedResp.Embeddings) == 0 {
		err := fmt.Errorf("no embeddings returned from API")
		logger.EndWithError(err)
		return nil, err
	}

	embedding := embedResp.Embeddings[0]
	logger.Info().Int("embedding_dimensions", len(embedding)).Msg("Embedding generation successful")
	logger.EndWithMsg("Embedding generation completed successfully")
	return embedding, nil
}

// HealthCheck checks if Ollama is running and accessible
func (c *Client) HealthCheck(ctx context.Context) error {
	logger := logger.NewLogger("ollama-health-check")
	logger.StartWithMsg("Performing Ollama health check")
	logger.Info().Str("url", c.baseURL+"/api/tags").Msg("Checking Ollama availability")

	httpReq, err := http.NewRequest("GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		logger.EndWithError(fmt.Errorf("failed to create health check request: %w", err))
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		logger.EndWithError(fmt.Errorf("failed to send health check request: %w", err))
		return fmt.Errorf("failed to send health check request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("health check failed with status %d", resp.StatusCode)
		logger.Error().Int("status_code", resp.StatusCode).Msg("Health check failed")
		logger.EndWithError(err)
		return err
	}

	logger.Info().Msg("Ollama health check passed")
	logger.EndWithMsg("Health check completed successfully")
	return nil
}

// ListModels returns a list of available models
func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	httpReq, err := http.NewRequest("GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]string, len(result.Models))
	for i, model := range result.Models {
		models[i] = model.Name
	}

	return models, nil
}

// RetryConfig holds configuration for retry logic
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 2,               // Reduced from 3 to 2
		BaseDelay:  2 * time.Second, // Increased base delay
	}
}

// isRetryableError checks if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Only retry on specific timeout errors
	if err == context.DeadlineExceeded {
		return true
	}

	// Check for specific network timeout errors
	if strings.Contains(errStr, "timeout") && strings.Contains(errStr, "context deadline exceeded") {
		return true
	}

	// Check for connection refused (server temporarily unavailable)
	if strings.Contains(errStr, "connection refused") {
		return true
	}

	// Don't retry on other errors (like 4xx, 5xx HTTP errors, parsing errors, etc.)
	return false
}

// retryWithBackoff executes a function with exponential backoff retry
func retryWithBackoff(ctx context.Context, config RetryConfig, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff
			delay := time.Duration(float64(config.BaseDelay) * math.Pow(2, float64(attempt-1)))
			if delay > 30*time.Second {
				delay = 30 * time.Second // Cap at 30 seconds
			}

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		if !isRetryableError(lastErr) {
			return lastErr
		}
	}

	return fmt.Errorf("failed after %d retries: %w", config.MaxRetries, lastErr)
}
