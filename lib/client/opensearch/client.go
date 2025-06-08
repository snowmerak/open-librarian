package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/snowmerak/open-librarian/lib/util/language"
)

type Client struct {
	baseURL          string
	httpClient       *http.Client
	languageDetector *language.Detector
}

// Article represents an article document in OpenSearch
type Article struct {
	ID          string    `json:"id,omitempty"`
	Lang        string    `json:"lang"`
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	Content     string    `json:"content"`
	Tags        []string  `json:"tags"`
	OriginalURL string    `json:"original_url,omitempty"`
	Author      string    `json:"author,omitempty"`
	CreatedDate time.Time `json:"created_date"`
}

// SearchRequest represents a search query
type SearchRequest struct {
	Query string `json:"query"`
	Lang  string `json:"lang,omitempty"`
	Size  int    `json:"size,omitempty"`
	From  int    `json:"from,omitempty"`
}

// SearchResult represents a single search result with score
type SearchResult struct {
	Article Article `json:"article"`
	Score   float64 `json:"score"`
}

// SearchResponse represents the search results
type SearchResponse struct {
	Total   int            `json:"total"`
	Results []SearchResult `json:"results"`
	Took    int            `json:"took"`
}

// IndexResponse represents the response from indexing an article
type IndexResponse struct {
	ID      string `json:"_id"`
	Index   string `json:"_index"`
	Version int    `json:"_version"`
	Result  string `json:"result"`
}

const (
	DefaultIndexName = "open-librarian-articles"
	DefaultBaseURL   = "http://localhost:9200"
)

// NewClient creates a new OpenSearch client
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	return &Client{
		baseURL:          baseURL,
		languageDetector: language.NewDetector(),
		httpClient: &http.Client{
			Timeout: 3 * time.Minute,
		},
	}
}

// IndexArticle indexes a new article with embeddings
func (c *Client) IndexArticle(ctx context.Context, article *Article) (*IndexResponse, error) {
	// Auto-detect language if not provided
	if article.Lang == "" {
		article.Lang = c.languageDetector.DetectLanguage(article.Title + " " + article.Summary)
	}

	// Validate language code
	if !c.languageDetector.ValidateLanguageCode(article.Lang) {
		article.Lang = "en" // Default to English for unsupported languages
	}

	// Prepare the document for indexing
	doc := map[string]interface{}{
		"lang":         article.Lang,
		"title":        article.Title,
		"summary":      article.Summary,
		"content":      article.Content,
		"tags":         article.Tags,
		"original_url": article.OriginalURL,
		"author":       article.Author,
		"created_date": article.CreatedDate,
	}

	reqBody, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal document: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_doc", c.baseURL, DefaultIndexName)
	if article.ID != "" {
		url = fmt.Sprintf("%s/%s/_doc/%s", c.baseURL, DefaultIndexName, article.ID)
	}

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

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("indexing failed with status %d: %s", resp.StatusCode, string(body))
	}

	var indexResp IndexResponse
	if err := json.NewDecoder(resp.Body).Decode(&indexResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &indexResp, nil
}

// KeywordSearch performs traditional keyword-based search
func (c *Client) KeywordSearch(ctx context.Context, query, lang string, size, from int) (*SearchResponse, error) {
	if size == 0 {
		size = 10
	}

	log.Printf("=== OpenSearch KeywordSearch START ===")
	log.Printf("Query: '%s', Lang: '%s', Size: %d, From: %d", query, lang, size, from)

	searchQuery := c.buildKeywordQuery(query, lang, size, from)

	// Log the search query
	queryJSON, _ := json.MarshalIndent(searchQuery, "", "  ")
	log.Printf("OpenSearch Query:\n%s", string(queryJSON))

	reqBody, err := json.Marshal(searchQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_search", c.baseURL, DefaultIndexName)
	log.Printf("OpenSearch URL: %s", url)

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
		log.Printf("OpenSearch error response (status %d): %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("search failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Read response body for logging
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	log.Printf("OpenSearch raw response: %s", string(responseBody))

	var esResp struct {
		Took int `json:"took"`
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				ID     string  `json:"_id"`
				Score  float64 `json:"_score"`
				Source Article `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.Unmarshal(responseBody, &esResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	log.Printf("OpenSearch parsed response: Total=%d, Hits=%d, Took=%d",
		esResp.Hits.Total.Value, len(esResp.Hits.Hits), esResp.Took)

	results := make([]SearchResult, len(esResp.Hits.Hits))
	for i, hit := range esResp.Hits.Hits {
		results[i] = SearchResult{
			Article: hit.Source,
			Score:   hit.Score,
		}
		results[i].Article.ID = hit.ID
		log.Printf("OpenSearch result #%d: ID=%s, Score=%.4f, Title='%s'",
			i+1, hit.ID, hit.Score, hit.Source.Title)
	}

	response := &SearchResponse{
		Total:   esResp.Hits.Total.Value,
		Results: results,
		Took:    esResp.Took,
	}

	log.Printf("=== OpenSearch KeywordSearch END ===")
	return response, nil
}

// SimpleQueryStringSearch performs search using simple_query_string syntax
func (c *Client) SimpleQueryStringSearch(ctx context.Context, queryText, lang string, size, from int) (*SearchResponse, error) {
	// Redirect to KeywordSearch for consistency
	return c.KeywordSearch(ctx, queryText, lang, size, from)
}

// CreateIndexWithMapping creates the index with proper field mappings for keyword search
func (c *Client) CreateIndexWithMapping(ctx context.Context) error {
	mapping := map[string]interface{}{
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"lang": map[string]interface{}{
					"type": "keyword",
				},
				"title": map[string]interface{}{
					"type": "text",
					"fields": map[string]interface{}{
						"ko": map[string]interface{}{
							"type":     "text",
							"analyzer": "korean",
						},
						"en": map[string]interface{}{
							"type":     "text",
							"analyzer": "english",
						},
						"ja": map[string]interface{}{
							"type":     "text",
							"analyzer": "cjk",
						},
					},
				},
				"summary": map[string]interface{}{
					"type": "text",
					"fields": map[string]interface{}{
						"ko": map[string]interface{}{
							"type":     "text",
							"analyzer": "korean",
						},
						"en": map[string]interface{}{
							"type":     "text",
							"analyzer": "english",
						},
						"ja": map[string]interface{}{
							"type":     "text",
							"analyzer": "cjk",
						},
					},
				},
				"content": map[string]interface{}{
					"type": "text",
					"fields": map[string]interface{}{
						"ko": map[string]interface{}{
							"type":     "text",
							"analyzer": "korean",
						},
						"en": map[string]interface{}{
							"type":     "text",
							"analyzer": "english",
						},
						"ja": map[string]interface{}{
							"type":     "text",
							"analyzer": "cjk",
						},
					},
				},
				"tags": map[string]interface{}{
					"type": "keyword",
				},
				"original_url": map[string]interface{}{
					"type":  "keyword",
					"index": false,
				},
				"author": map[string]interface{}{
					"type": "keyword",
				},
				"created_date": map[string]interface{}{
					"type": "date",
				},
			},
		},
	}

	reqBody, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("failed to marshal mapping: %w", err)
	}

	url := fmt.Sprintf("%s/%s", c.baseURL, DefaultIndexName)
	httpReq, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("index creation failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// buildKeywordQuery builds a keyword-only search query using simple_query_string
func (c *Client) buildKeywordQuery(queryText, lang string, size, from int) map[string]interface{} {
	query := map[string]interface{}{
		"size":    size,
		"from":    from,
		"_source": []string{"title", "summary", "content", "original_url", "author", "lang", "tags", "created_date"},
	}

	// Use simple_query_string for natural language search across all fields
	var fields []string
	if lang != "" {
		// Language-specific fields with boost, plus fallback to default fields
		fields = []string{
			"title^4", // Always include default fields first
			"summary^2",
			"content",
			"tags^2",
			"author",
			fmt.Sprintf("title.%s^5", lang), // Higher boost for language-specific
			fmt.Sprintf("summary.%s^3", lang),
			fmt.Sprintf("content.%s^1.5", lang),
		}
	} else {
		// Default fields for all languages
		fields = []string{
			"title^4",
			"summary^2",
			"content",
			"tags^2",
			"author",
		}
	}

	query["query"] = map[string]interface{}{
		"simple_query_string": map[string]interface{}{
			"query":            queryText,
			"fields":           fields,
			"default_operator": "or", // Changed from "and" to "or" for better recall
			"flags":            "ALL",
			"analyze_wildcard": true,
			"lenient":          true,
		},
	}

	// Add highlighting
	query["highlight"] = map[string]interface{}{
		"fields": map[string]interface{}{
			"title":   map[string]interface{}{},
			"summary": map[string]interface{}{},
			"content": map[string]interface{}{
				"fragment_size":       150,
				"number_of_fragments": 3,
			},
		},
		"pre_tags":  []string{"<mark>"},
		"post_tags": []string{"</mark>"},
	}

	return query
}

// GetArticle retrieves an article by ID
func (c *Client) GetArticle(ctx context.Context, id string) (*Article, error) {
	url := fmt.Sprintf("%s/%s/_doc/%s", c.baseURL, DefaultIndexName, id)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("article not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get failed with status %d: %s", resp.StatusCode, string(body))
	}

	var esResp struct {
		ID     string  `json:"_id"`
		Source Article `json:"_source"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&esResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	article := esResp.Source
	article.ID = esResp.ID

	return &article, nil
}

// GetArticleByID retrieves an article by its ID
func (c *Client) GetArticleByID(ctx context.Context, articleID string) (*Article, error) {
	url := fmt.Sprintf("%s/%s/_doc/%s", c.baseURL, DefaultIndexName, articleID)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("article not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get article failed with status %d: %s", resp.StatusCode, string(body))
	}

	var esResp struct {
		ID     string  `json:"_id"`
		Source Article `json:"_source"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&esResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	article := esResp.Source
	article.ID = esResp.ID

	return &article, nil
}

// GetArticlesByIDs retrieves multiple articles by their IDs
func (c *Client) GetArticlesByIDs(ctx context.Context, articleIDs []string) ([]Article, error) {
	if len(articleIDs) == 0 {
		return []Article{}, nil
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"ids": map[string]interface{}{
				"values": articleIDs,
			},
		},
		"size": len(articleIDs),
	}

	reqBody, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_search", c.baseURL, DefaultIndexName)
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
		return nil, fmt.Errorf("search failed with status %d: %s", resp.StatusCode, string(body))
	}

	var esResp struct {
		Hits struct {
			Hits []struct {
				ID     string  `json:"_id"`
				Source Article `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&esResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	articles := make([]Article, len(esResp.Hits.Hits))
	for i, hit := range esResp.Hits.Hits {
		articles[i] = hit.Source
		articles[i].ID = hit.ID
	}

	return articles, nil
}

// HealthCheck checks if OpenSearch is accessible
func (c *Client) HealthCheck(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/_cluster/health", nil)
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

// DetectQueryLanguage detects the language of a search query
func (c *Client) DetectQueryLanguage(query string) string {
	return c.languageDetector.DetectLanguage(query)
}

// GetSupportedLanguages returns supported language codes
func (c *Client) GetSupportedLanguages() []string {
	return c.languageDetector.GetSupportedLanguages()
}
