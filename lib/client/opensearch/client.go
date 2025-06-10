package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/snowmerak/open-librarian/lib/util/language"
	"github.com/snowmerak/open-librarian/lib/util/logger"
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
	Registrar   string    `json:"registrar,omitempty"`
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
	logger := logger.NewLogger("opensearch-client")
	logger.StartWithMsg("Creating new OpenSearch client")

	if baseURL == "" {
		baseURL = DefaultBaseURL
		logger.Info().Str("base_url", baseURL).Msg("Using default base URL")
	} else {
		logger.Info().Str("base_url", baseURL).Msg("Using provided base URL")
	}

	client := &Client{
		baseURL:          baseURL,
		languageDetector: language.NewDetector(),
		httpClient: &http.Client{
			Timeout: 3 * time.Minute,
		},
	}

	logger.EndWithMsg("OpenSearch client created successfully")
	return client
}

// IndexArticle indexes a new article with embeddings
func (c *Client) IndexArticle(ctx context.Context, article *Article) (*IndexResponse, error) {
	logger := logger.NewLogger("opensearch-index-article")
	logger.StartWithMsg("Indexing article in OpenSearch")
	logger.Info().Str("article_id", article.ID).Str("title", article.Title).Msg("Article indexing request")

	// Auto-detect language if not provided
	if article.Lang == "" {
		article.Lang = c.languageDetector.DetectLanguage(article.Title + " " + article.Summary)
		logger.Info().Str("detected_language", article.Lang).Msg("Language auto-detected")
	}

	// Validate language code
	if !c.languageDetector.ValidateLanguageCode(article.Lang) {
		originalLang := article.Lang
		article.Lang = "en" // Default to English for unsupported languages
		logger.Warn().Str("original_lang", originalLang).Str("default_lang", article.Lang).Msg("Invalid language code, using default")
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
		"registrar":    article.Registrar,
	}

	reqBody, err := json.Marshal(doc)
	if err != nil {
		logger.EndWithError(fmt.Errorf("failed to marshal document: %w", err))
		return nil, fmt.Errorf("failed to marshal document: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_doc", c.baseURL, DefaultIndexName)
	if article.ID != "" {
		url = fmt.Sprintf("%s/%s/_doc/%s", c.baseURL, DefaultIndexName, article.ID)
	}
	logger.Info().Str("url", url).Msg("Sending indexing request")

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		logger.EndWithError(fmt.Errorf("failed to create request: %w", err))
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		logger.EndWithError(fmt.Errorf("failed to send request: %w", err))
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("indexing failed with status %d: %s", resp.StatusCode, string(body))
		logger.Error().Int("status_code", resp.StatusCode).Msg("Indexing failed")
		logger.EndWithError(err)
		return nil, err
	}

	var indexResp IndexResponse
	if err := json.NewDecoder(resp.Body).Decode(&indexResp); err != nil {
		logger.EndWithError(fmt.Errorf("failed to decode response: %w", err))
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	logger.DataCreated("article", indexResp.ID, map[string]interface{}{
		"index":   indexResp.Index,
		"version": indexResp.Version,
		"result":  indexResp.Result,
		"lang":    article.Lang,
	})
	logger.EndWithMsg("Article indexed successfully")
	return &indexResp, nil
}

// KeywordSearch performs traditional keyword-based search
func (c *Client) KeywordSearch(ctx context.Context, query, lang string, size, from int) (*SearchResponse, error) {
	searchLogger := logger.NewLogger("opensearch-keyword-search")
	searchLogger.StartWithMsg("Starting OpenSearch keyword search")

	if size == 0 {
		size = 10
	}

	searchLogger.Info().
		Str("query", query).
		Str("language", lang).
		Int("size", size).
		Int("from", from).
		Msg("Start keyword search")

	searchQuery := c.buildKeywordQuery(query, lang, size, from)

	// Log the search query
	queryJSON, _ := json.MarshalIndent(searchQuery, "", "  ")
	searchLogger.Debug().Str("search_query", string(queryJSON)).Msg("Generated search query")

	reqBody, err := json.Marshal(searchQuery)
	if err != nil {
		searchLogger.EndWithError(err)
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_search", c.baseURL, DefaultIndexName)
	searchLogger.Info().Str("url", url).Msg("Search request URL")

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		searchLogger.EndWithError(err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		searchLogger.EndWithError(err)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		searchLogger.Error().
			Int("status_code", resp.StatusCode).
			Str("response_body", string(body)).
			Msg("OpenSearch error response")
		searchLogger.EndWithError(fmt.Errorf("search failed with status %d", resp.StatusCode))
		return nil, fmt.Errorf("search failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Read response body for logging
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		searchLogger.EndWithError(err)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	searchLogger.Debug().Str("raw_response", string(responseBody)).Msg("OpenSearch raw response")

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
		searchLogger.EndWithError(err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	searchLogger.Info().
		Int("total_results", esResp.Hits.Total.Value).
		Int("returned_hits", len(esResp.Hits.Hits)).
		Int("took_ms", esResp.Took).
		Msg("OpenSearch search completed")

	results := make([]SearchResult, len(esResp.Hits.Hits))
	for i, hit := range esResp.Hits.Hits {
		results[i] = SearchResult{
			Article: hit.Source,
			Score:   hit.Score,
		}
		results[i].Article.ID = hit.ID
		searchLogger.Debug().
			Int("result_index", i+1).
			Str("article_id", hit.ID).
			Float64("score", hit.Score).
			Str("title", hit.Source.Title).
			Msg("Search result")
	}

	response := &SearchResponse{
		Total:   esResp.Hits.Total.Value,
		Results: results,
		Took:    esResp.Took,
	}

	searchLogger.EndWithMsg("OpenSearch keyword search completed successfully")
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
				"registrar": map[string]interface{}{
					"type": "keyword",
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

// DeleteArticle deletes an article from the index
func (c *Client) DeleteArticle(ctx context.Context, id string) error {
	deleteLogger := logger.NewLogger("opensearch-delete-article")
	deleteLogger.StartWithMsg("Deleting article from OpenSearch")
	deleteLogger.Info().Str("article_id", id).Msg("Article deletion request")

	url := fmt.Sprintf("%s/%s/_doc/%s", c.baseURL, DefaultIndexName, id)
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		deleteLogger.EndWithError(err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		deleteLogger.EndWithError(err)
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		deleteLogger.EndWithError(fmt.Errorf("delete failed with status %d", resp.StatusCode))
		return fmt.Errorf("delete failed with status %d: %s", resp.StatusCode, string(body))
	}

	deleteLogger.Info().Str("article_id", id).Msg("Article deleted successfully")
	deleteLogger.EndWithMsg("Article deletion completed")
	return nil
}

// GetUserArticlesByDateRange retrieves articles registered by a specific user within a date range
func (c *Client) GetUserArticlesByDateRange(ctx context.Context, username, dateFrom, dateTo string, size, from int) (*SearchResponse, error) {
	userSearchLogger := logger.NewLogger("opensearch-user-articles-by-date")
	userSearchLogger.StartWithMsg("Getting user articles by date range")

	userSearchLogger.Info().
		Str("username", username).
		Str("date_from", dateFrom).
		Str("date_to", dateTo).
		Int("size", size).
		Int("from", from).
		Msg("User articles search parameters")

	// Build the query for user articles with date range
	query := map[string]interface{}{
		"size":    size,
		"from":    from,
		"_source": []string{"title", "summary", "content", "original_url", "author", "lang", "tags", "created_date", "registrar"},
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"term": map[string]interface{}{
							"registrar": username,
						},
					},
				},
			},
		},
		"sort": []map[string]interface{}{
			{
				"created_date": map[string]interface{}{
					"order": "desc",
				},
			},
		},
	}

	// Add date range filter if provided
	if dateFrom != "" || dateTo != "" {
		dateRange := map[string]interface{}{}
		if dateFrom != "" {
			dateRange["gte"] = dateFrom
		}
		if dateTo != "" {
			dateRange["lte"] = dateTo
		}

		boolQuery := query["query"].(map[string]interface{})["bool"].(map[string]interface{})
		boolQuery["must"] = append(boolQuery["must"].([]map[string]interface{}), map[string]interface{}{
			"range": map[string]interface{}{
				"created_date": dateRange,
			},
		})
	}

	// Log the search query
	queryJSON, _ := json.MarshalIndent(query, "", "  ")
	userSearchLogger.Debug().Str("search_query", string(queryJSON)).Msg("Generated user articles query")

	reqBody, err := json.Marshal(query)
	if err != nil {
		userSearchLogger.EndWithError(err)
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_search", c.baseURL, DefaultIndexName)
	userSearchLogger.Info().Str("url", url).Msg("User articles search request URL")

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		userSearchLogger.EndWithError(err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		userSearchLogger.EndWithError(err)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		userSearchLogger.Error().
			Int("status_code", resp.StatusCode).
			Str("response_body", string(body)).
			Msg("OpenSearch error response")
		userSearchLogger.EndWithError(fmt.Errorf("search failed with status %d", resp.StatusCode))
		return nil, fmt.Errorf("search failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		userSearchLogger.EndWithError(err)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	userSearchLogger.Debug().Str("raw_response", string(responseBody)).Msg("OpenSearch raw response")

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
		userSearchLogger.EndWithError(err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	userSearchLogger.Info().
		Int("total_results", esResp.Hits.Total.Value).
		Int("returned_hits", len(esResp.Hits.Hits)).
		Int("took_ms", esResp.Took).
		Msg("User articles search completed")

	results := make([]SearchResult, len(esResp.Hits.Hits))
	for i, hit := range esResp.Hits.Hits {
		results[i] = SearchResult{
			Article: hit.Source,
			Score:   hit.Score,
		}
		results[i].Article.ID = hit.ID
		userSearchLogger.Debug().
			Int("result_index", i+1).
			Str("article_id", hit.ID).
			Str("title", hit.Source.Title).
			Str("created_date", hit.Source.CreatedDate.Format("2006-01-02T15:04:05Z")).
			Msg("User article result")
	}

	response := &SearchResponse{
		Total:   esResp.Hits.Total.Value,
		Results: results,
		Took:    esResp.Took,
	}

	userSearchLogger.EndWithMsg("User articles search completed successfully")
	return response, nil
}
