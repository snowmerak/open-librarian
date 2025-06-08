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
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Minute, // Increased timeout significantly
		},
	}
}

// GenerateText generates text using the specified model (default: gemma2:14b)
func (c *Client) GenerateText(ctx context.Context, prompt string) (string, error) {
	return c.GenerateTextWithModel(ctx, DefaultTextModel, prompt)
}

// GenerateTextWithModel generates text using a specific model
func (c *Client) GenerateTextWithModel(ctx context.Context, model, prompt string) (string, error) {
	req := GenerateRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx = context.WithoutCancel(ctx)

	httpReq, err := http.NewRequest("POST", c.baseURL+"/api/generate", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var genResp GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return genResp.Response, nil
}

// GenerateTextStream generates text using streaming mode
func (c *Client) GenerateTextStream(ctx context.Context, prompt string, callback func(string) error) error {
	return c.GenerateTextStreamWithModel(ctx, DefaultTextModel, prompt, callback)
}

// GenerateTextStreamWithModel generates text using a specific model in streaming mode
func (c *Client) GenerateTextStreamWithModel(ctx context.Context, model, prompt string, callback func(string) error) error {
	req := GenerateRequest{
		Model:  model,
		Prompt: prompt,
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
	return c.GenerateEmbeddingWithModel(ctx, DefaultEmbeddingModel, text)
}

// GenerateEmbeddingWithModel generates embeddings using a specific model
func (c *Client) GenerateEmbeddingWithModel(ctx context.Context, model, text string) ([]float64, error) {
	req := EmbedRequest{
		Model: model,
		Input: text,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/api/embed", bytes.NewBuffer(reqBody))
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

	// Ollama returns embeddings as array of arrays, we want the first one
	if len(embedResp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned from API")
	}
	return embedResp.Embeddings[0], nil
}

// HealthCheck checks if Ollama is running and accessible
func (c *Client) HealthCheck(ctx context.Context) error {
	httpReq, err := http.NewRequest("GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send health check request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

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
