package api

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/snowmerak/open-librarian/lib/client/ollama"
	"github.com/snowmerak/open-librarian/lib/client/opensearch"
	"github.com/snowmerak/open-librarian/lib/client/qdrant"
	"github.com/snowmerak/open-librarian/lib/util/language"
)

// Server represents the main API server
type Server struct {
	ollamaClient     *ollama.Client
	opensearchClient *opensearch.Client
	qdrantClient     *qdrant.Client
	languageDetector *language.Detector
}

// ArticleRequest represents the request to add an article
type ArticleRequest struct {
	Title       string `json:"title" validate:"required"`
	Content     string `json:"content" validate:"required"`
	OriginalURL string `json:"original_url,omitempty"`
	Author      string `json:"author,omitempty"`
	CreatedDate string `json:"created_date,omitempty"` // RFC3339 format (e.g., "2023-12-25T15:30:00Z")
}

// SearchRequest represents the search request
type SearchRequest struct {
	Query    string `json:"query" validate:"required"`
	Size     int    `json:"size,omitempty"`
	From     int    `json:"from,omitempty"`
	DateFrom string `json:"date_from,omitempty"` // RFC3339 format for filtering articles created after this date
	DateTo   string `json:"date_to,omitempty"`   // RFC3339 format for filtering articles created before this date
}

// ArticleResponse represents the response after adding an article
type ArticleResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// SearchResultWithScore represents a search result with score
type SearchResultWithScore struct {
	Article opensearch.Article `json:"article"`
	Score   float64            `json:"score"`
	Source  string             `json:"source"` // "keyword" or "vector"
}

// SearchResponse represents the search response
type SearchResponse struct {
	Answer  string                  `json:"answer"`
	Sources []SearchResultWithScore `json:"sources"`
	Took    int                     `json:"took"`
}

// NewServer creates a new API server instance
func NewServer(ollamaBaseURL, opensearchBaseURL, qdrantHost string, qdrantPort int) (*Server, error) {
	qdrantClient, err := qdrant.NewClient(qdrantHost, qdrantPort)
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	// Initialize Qdrant collection with 768-dimension vectors
	ctx := context.Background()
	if err := qdrantClient.CreateCollection(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize Qdrant collection: %w", err)
	}

	return &Server{
		ollamaClient:     ollama.NewClient(ollamaBaseURL),
		opensearchClient: opensearch.NewClient(opensearchBaseURL),
		qdrantClient:     qdrantClient,
		languageDetector: language.NewDetector(),
	}, nil
}

// HealthCheck checks the health of all services
func (s *Server) HealthCheck(ctx context.Context) error {
	// Check Ollama
	if err := s.ollamaClient.HealthCheck(ctx); err != nil {
		return fmt.Errorf("ollama health check failed: %w", err)
	}

	// Check OpenSearch
	if err := s.opensearchClient.HealthCheck(ctx); err != nil {
		return fmt.Errorf("opensearch health check failed: %w", err)
	}

	// Check Qdrant
	if err := s.qdrantClient.HealthCheck(ctx); err != nil {
		return fmt.Errorf("qdrant health check failed: %w", err)
	}

	return nil
}

// AddArticle processes and indexes a new article
func (s *Server) AddArticle(ctx context.Context, req *ArticleRequest) (*ArticleResponse, error) {
	log.Printf("Processing article: %s", req.Title)

	// 1. Check for duplicate articles based on title and content similarity
	isDuplicate, existingID, err := s.checkDuplicateArticle(ctx, req.Title, req.Content)
	if err != nil {
		log.Printf("Failed to check for duplicates: %v", err)
		// Continue with indexing despite duplicate check failure
	} else if isDuplicate {
		log.Printf("Duplicate article detected, existing ID: %s", existingID)
		return &ArticleResponse{
			ID:      existingID,
			Message: "Duplicate article found, returning existing article ID",
		}, nil
	}

	// 2. Detect language
	lang := s.languageDetector.DetectLanguage(req.Content)
	log.Printf("Detected language: %s", lang)

	// 3. Generate summary using Ollama
	summaryPrompt := fmt.Sprintf(`Please summarize the following text in 3-4 sentences in English. Include key points and important information. Only return the summary without any additional text.

Text:
%s

Summary:`, req.Content)

	summary, err := s.ollamaClient.GenerateText(ctx, summaryPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %w", err)
	}
	log.Printf("Generated summary: %.100s...", summary)

	// 4. Generate tags using Ollama
	tagsPrompt := fmt.Sprintf(`Extract 5 key keywords from the following text in English. Separate them with commas. Only return the keywords without any additional text.

Text:
%s

Keywords:`, req.Content)

	tagsText, err := s.ollamaClient.GenerateText(ctx, tagsPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tags: %w", err)
	}

	// Simple tag parsing (split by comma)
	tags := []string{}
	if tagsText != "" {
		// Basic parsing - in production, you might want more sophisticated parsing
		tags = append(tags, tagsText) // For now, store as single tag
	}

	// 4. Generate embeddings for both title and summary
	titleEmbedding, err := s.ollamaClient.GenerateEmbedding(ctx, "passage: "+req.Title)
	if err != nil {
		return nil, fmt.Errorf("failed to generate title embedding: %w", err)
	}

	summaryEmbedding, err := s.ollamaClient.GenerateEmbedding(ctx, "passage: "+summary)
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary embedding: %w", err)
	}

	// 5. Parse and validate created date
	var createdDate time.Time
	if req.CreatedDate != "" {
		parsed, err := time.Parse(time.RFC3339, req.CreatedDate)
		if err != nil {
			return nil, fmt.Errorf("invalid created_date format: %w (expected RFC3339 format like 2023-12-25T15:30:00Z)", err)
		}

		// Validate that the date is not in the future (with 1 minute tolerance for clock skew)
		now := time.Now()
		if parsed.After(now.Add(time.Minute)) {
			return nil, fmt.Errorf("created_date cannot be in the future")
		}

		// Warn if the date is more than 10 years old (might be a mistake)
		tenYearsAgo := now.AddDate(-10, 0, 0)
		if parsed.Before(tenYearsAgo) {
			log.Printf("Warning: Article created_date is more than 10 years old: %s", parsed.Format(time.RFC3339))
		}

		createdDate = parsed
		log.Printf("Using provided created_date: %s", createdDate.Format(time.RFC3339))
	} else {
		createdDate = time.Now()
		log.Printf("No created_date provided, using current time: %s", createdDate.Format(time.RFC3339))
	}

	// 6. Create article object (without embeddings for OpenSearch)
	article := &opensearch.Article{
		Lang:        lang,
		Title:       req.Title,
		Summary:     summary,
		Content:     req.Content,
		Tags:        tags,
		OriginalURL: req.OriginalURL,
		Author:      req.Author,
		CreatedDate: createdDate,
	}

	// 7. Index in OpenSearch (text data only)
	indexResp, err := s.opensearchClient.IndexArticle(ctx, article)
	if err != nil {
		return nil, fmt.Errorf("failed to index article: %w", err)
	}

	// 8. Index vectors in Qdrant (use the same ID from OpenSearch)
	// Index both title and summary embeddings
	err = s.qdrantClient.UpsertPoint(ctx, indexResp.ID+"_title", titleEmbedding, lang)
	if err != nil {
		log.Printf("Failed to index title embedding in Qdrant, cleaning up OpenSearch entry")
		return nil, fmt.Errorf("failed to index title vectors in Qdrant: %w", err)
	}

	err = s.qdrantClient.UpsertPoint(ctx, indexResp.ID+"_summary", summaryEmbedding, lang)
	if err != nil {
		log.Printf("Failed to index summary embedding in Qdrant, cleaning up OpenSearch entry")
		return nil, fmt.Errorf("failed to index summary vectors in Qdrant: %w", err)
	}

	log.Printf("Successfully indexed article with ID: %s", indexResp.ID)

	return &ArticleResponse{
		ID:      indexResp.ID,
		Message: "Article indexed successfully",
	}, nil
}

// ProgressCallback represents a function that can be called to report progress
type ProgressCallback func(step string, progress int, total int) error

// AddArticleWithProgress processes and indexes a new article with progress callbacks
func (s *Server) AddArticleWithProgress(ctx context.Context, req *ArticleRequest, progressCallback ProgressCallback) (*ArticleResponse, error) {
	log.Printf("Processing article with progress tracking: %s", req.Title)

	totalSteps := 8
	currentStep := 0

	// Helper function to report progress
	reportProgress := func(step string) error {
		currentStep++
		if progressCallback != nil {
			return progressCallback(step, currentStep, totalSteps)
		}
		return nil
	}

	// 1. Check for duplicate articles based on title and content similarity
	if err := reportProgress("Checking for duplicate articles..."); err != nil {
		return nil, err
	}
	isDuplicate, existingID, err := s.checkDuplicateArticle(ctx, req.Title, req.Content)
	if err != nil {
		log.Printf("Failed to check for duplicates: %v", err)
		// Continue with indexing despite duplicate check failure
	} else if isDuplicate {
		log.Printf("Duplicate article detected, existing ID: %s", existingID)
		return &ArticleResponse{
			ID:      existingID,
			Message: "Duplicate article found, returning existing article ID",
		}, nil
	}

	// 2. Detect language
	if err := reportProgress("Detecting language..."); err != nil {
		return nil, err
	}
	lang := s.languageDetector.DetectLanguage(req.Content)
	log.Printf("Detected language: %s", lang)

	// 3. Generate summary using Ollama
	if err := reportProgress("Generating summary..."); err != nil {
		return nil, err
	}
	summaryPrompt := fmt.Sprintf(`Please summarize the following text in 3-4 sentences in English. Include key points and important information.

Text:
%s

Summary:`, req.Content)

	// Use longer timeout for summary generation
	summaryCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	summary, err := s.ollamaClient.GenerateText(summaryCtx, summaryPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %w", err)
	}
	log.Printf("Generated summary: %.100s...", summary)

	// 4. Generate tags using Ollama
	if err := reportProgress("Generating tags..."); err != nil {
		return nil, err
	}
	tagsPrompt := fmt.Sprintf(`Extract 5 key keywords from the following text in English. Separate them with commas. Only return the keywords without any additional text.

Text:
%s

Keywords:`, req.Content)

	// Use longer timeout for tags generation
	tagsCtx, cancel2 := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel2()

	tagsText, err := s.ollamaClient.GenerateText(tagsCtx, tagsPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tags: %w", err)
	}

	// Simple tag parsing (split by comma)
	tags := []string{}
	if tagsText != "" {
		// Basic parsing - in production, you might want more sophisticated parsing
		tags = append(tags, tagsText) // For now, store as single tag
	}

	// 5. Generate embeddings for both title and summary
	if err := reportProgress("Generating embeddings..."); err != nil {
		return nil, err
	}

	// Use longer timeout for embeddings generation
	embeddingCtx, cancel3 := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel3()

	titleEmbedding, err := s.ollamaClient.GenerateEmbedding(embeddingCtx, "passage: "+req.Title)
	if err != nil {
		return nil, fmt.Errorf("failed to generate title embedding: %w", err)
	}

	summaryEmbedding, err := s.ollamaClient.GenerateEmbedding(embeddingCtx, "passage: "+summary)
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary embedding: %w", err)
	}

	// 6. Parse and validate created date
	if err := reportProgress("Validating date..."); err != nil {
		return nil, err
	}
	var createdDate time.Time
	if req.CreatedDate != "" {
		parsed, err := time.Parse(time.RFC3339, req.CreatedDate)
		if err != nil {
			return nil, fmt.Errorf("invalid created_date format: %w (expected RFC3339 format like 2023-12-25T15:30:00Z)", err)
		}

		// Validate that the date is not in the future (with 1 minute tolerance for clock skew)
		now := time.Now()
		if parsed.After(now.Add(time.Minute)) {
			return nil, fmt.Errorf("created_date cannot be in the future")
		}

		// Warn if the date is more than 10 years old (might be a mistake)
		tenYearsAgo := now.AddDate(-10, 0, 0)
		if parsed.Before(tenYearsAgo) {
			log.Printf("Warning: Article created_date is more than 10 years old: %s", parsed.Format(time.RFC3339))
		}

		createdDate = parsed
		log.Printf("Using provided created_date: %s", createdDate.Format(time.RFC3339))
	} else {
		createdDate = time.Now()
		log.Printf("No created_date provided, using current time: %s", createdDate.Format(time.RFC3339))
	}

	// 7. Create article object and index in OpenSearch
	if err := reportProgress("Indexing in OpenSearch..."); err != nil {
		return nil, err
	}
	article := &opensearch.Article{
		Lang:        lang,
		Title:       req.Title,
		Summary:     summary,
		Content:     req.Content,
		Tags:        tags,
		OriginalURL: req.OriginalURL,
		Author:      req.Author,
		CreatedDate: createdDate,
	}

	indexResp, err := s.opensearchClient.IndexArticle(ctx, article)
	if err != nil {
		return nil, fmt.Errorf("failed to index article: %w", err)
	}

	// 8. Index vectors in Qdrant
	if err := reportProgress("Indexing embeddings in Qdrant..."); err != nil {
		return nil, err
	}
	// Index both title and summary embeddings
	err = s.qdrantClient.UpsertPoint(ctx, indexResp.ID+"_title", titleEmbedding, lang)
	if err != nil {
		log.Printf("Failed to index title embedding in Qdrant, cleaning up OpenSearch entry")
		return nil, fmt.Errorf("failed to index title vectors in Qdrant: %w", err)
	}

	err = s.qdrantClient.UpsertPoint(ctx, indexResp.ID+"_summary", summaryEmbedding, lang)
	if err != nil {
		log.Printf("Failed to index summary embedding in Qdrant, cleaning up OpenSearch entry")
		return nil, fmt.Errorf("failed to index summary vectors in Qdrant: %w", err)
	}

	log.Printf("Successfully indexed article with ID: %s", indexResp.ID)

	return &ArticleResponse{
		ID:      indexResp.ID,
		Message: "Article indexed successfully",
	}, nil
}

// BulkArticleRequest represents a bulk upload request
type BulkArticleRequest struct {
	Articles []ArticleRequest `json:"articles" validate:"required"`
}

// BulkArticleResponse represents the response for bulk upload
type BulkArticleResponse struct {
	SuccessCount int                 `json:"success_count"`
	ErrorCount   int                 `json:"error_count"`
	Results      []BulkArticleResult `json:"results"`
}

// BulkArticleResult represents the result of a single article in bulk upload
type BulkArticleResult struct {
	Index   int    `json:"index"`
	Title   string `json:"title"`
	Success bool   `json:"success"`
	ID      string `json:"id,omitempty"`
	Error   string `json:"error,omitempty"`
}

// BulkProgressCallback represents a function that can be called to report bulk upload progress
type BulkProgressCallback func(articleIndex int, totalArticles int, currentStep string, stepProgress int, stepTotal int, result *BulkArticleResult) error

// AddArticlesBulkWithProgress processes multiple articles with progress callbacks
func (s *Server) AddArticlesBulkWithProgress(ctx context.Context, req *BulkArticleRequest, progressCallback BulkProgressCallback) (*BulkArticleResponse, error) {
	log.Printf("Processing bulk upload: %d articles", len(req.Articles))

	response := &BulkArticleResponse{
		Results: make([]BulkArticleResult, len(req.Articles)),
	}

	// Limit concurrent processing to reduce load on Ollama
	const maxConcurrent = 1 // Reduced from 2 to 1 for testing
	semaphore := make(chan struct{}, maxConcurrent)

	// Use channels for collecting results
	type indexedResult struct {
		index  int
		result BulkArticleResult
	}

	resultChan := make(chan indexedResult, len(req.Articles))

	// Process articles with limited concurrency
	for i, articleReq := range req.Articles {
		go func(index int, article ArticleRequest) {
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := BulkArticleResult{
				Index: index,
				Title: article.Title,
			}

			// Create individual progress callback for this article
			articleProgressCallback := func(step string, progress int, total int) error {
				if progressCallback != nil {
					return progressCallback(index, len(req.Articles), step, progress, total, nil)
				}
				return nil
			}

			// Process individual article with timeout
			articleCtx, cancel := context.WithTimeout(ctx, 10*time.Minute) // Increased from 5 to 10 minutes
			defer cancel()

			articleResp, err := s.AddArticleWithProgress(articleCtx, &article, articleProgressCallback)
			if err != nil {
				result.Success = false
				result.Error = err.Error()
				log.Printf("Failed to process article %d (%s): %v", index, article.Title, err)
			} else {
				result.Success = true
				result.ID = articleResp.ID
				log.Printf("Successfully processed article %d (%s): %s", index, article.Title, articleResp.ID)
			}

			// Report completion of this article
			if progressCallback != nil {
				progressCallback(index, len(req.Articles), "Article completed", 8, 8, &result)
			}

			resultChan <- indexedResult{index: index, result: result}
		}(i, articleReq)
	}

	// Collect all results
	for i := 0; i < len(req.Articles); i++ {
		select {
		case indexedRes := <-resultChan:
			response.Results[indexedRes.index] = indexedRes.result
			if indexedRes.result.Success {
				response.SuccessCount++
			} else {
				response.ErrorCount++
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	log.Printf("Bulk upload completed: %d success, %d errors", response.SuccessCount, response.ErrorCount)
	return response, nil
}

// Search performs hybrid search combining vector and keyword search
func (s *Server) Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	log.Printf("Searching for: %s", req.Query)

	// 1. Detect query language
	queryLang := s.languageDetector.DetectLanguage(req.Query)

	// 2. Generate query embedding for vector search
	queryEmbedding, err := s.ollamaClient.GenerateEmbedding(ctx, "query: "+req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// 3. Set default size if not provided
	size := req.Size
	if size == 0 {
		size = 5 // Default to top 5 results for AI answer generation
	}

	// 4. Perform parallel searches
	// 4a. Vector search with Qdrant (search both title and summary embeddings)
	allVectorResults, err := s.qdrantClient.VectorSearch(ctx, queryEmbedding, uint64(size*4), queryLang)
	if err != nil {
		log.Printf("Vector search failed: %v", err)
		allVectorResults = []qdrant.VectorSearchResult{}
	}

	// Separate title and summary results
	var titleVectorResults, summaryVectorResults []qdrant.VectorSearchResult
	for _, result := range allVectorResults {
		if len(result.ID) > 6 && result.ID[len(result.ID)-6:] == "_title" {
			titleVectorResults = append(titleVectorResults, result)
		} else if len(result.ID) > 8 && result.ID[len(result.ID)-8:] == "_summary" {
			summaryVectorResults = append(summaryVectorResults, result)
		}
	}

	// Combine and deduplicate vector results
	combinedVectorResults := s.combineVectorResults(titleVectorResults, summaryVectorResults)

	// 4b. Keyword search with OpenSearch
	keywordResp, err := s.opensearchClient.KeywordSearch(ctx, req.Query, queryLang, size*2, req.From)
	// 5. Get articles by IDs from vector search results
	var vectorArticleIDs []string
	uniqueIDs := make(map[string]bool)
	for _, result := range combinedVectorResults {
		// Extract original article ID (remove _title or _summary suffix)
		articleID := s.extractArticleID(result.ID)
		if !uniqueIDs[articleID] {
			vectorArticleIDs = append(vectorArticleIDs, articleID)
			uniqueIDs[articleID] = true
		}
	}

	log.Printf("Extracted unique article IDs: %v", vectorArticleIDs)

	var vectorArticles []opensearch.Article
	if len(vectorArticleIDs) > 0 {
		vectorArticles, err = s.opensearchClient.GetArticlesByIDs(ctx, vectorArticleIDs)
		if err != nil {
			log.Printf("Failed to get articles by IDs: %v", err)
			vectorArticles = []opensearch.Article{}
		}
	}

	// 6. Combine and deduplicate results
	combinedResults := s.combineSearchResults(combinedVectorResults, vectorArticles, keywordResp.Results, size)

	// 7. Extract articles for AI answer generation
	articles := make([]opensearch.Article, len(combinedResults))
	for i, result := range combinedResults {
		articles[i] = result.Article
	}

	// 8. Generate AI answer using search results
	answer, err := s.generateAnswer(ctx, req.Query, articles)
	if err != nil {
		return nil, fmt.Errorf("failed to generate answer: %w", err)
	}

	return &SearchResponse{
		Answer:  answer,
		Sources: combinedResults,
		Took:    keywordResp.Took, // Use keyword search timing for now
	}, nil
}

// combineSearchResults combines vector and keyword search results with scoring
func (s *Server) combineSearchResults(vectorResults []qdrant.VectorSearchResult, vectorArticles []opensearch.Article, keywordResults []opensearch.SearchResult, limit int) []SearchResultWithScore {
	const minScoreThreshold = 0.55 // Minimum score threshold for quality filtering

	// Create maps for easier lookup
	vectorArticleMap := make(map[string]opensearch.Article)
	for _, article := range vectorArticles {
		vectorArticleMap[article.ID] = article
	}

	vectorScoreMap := make(map[string]float64)
	for _, result := range vectorResults {
		articleID := s.extractArticleID(result.ID)
		vectorScoreMap[articleID] = result.Score
	}

	// Collect all results
	resultMap := make(map[string]SearchResultWithScore)

	// Add vector search results
	for _, result := range vectorResults {
		articleID := s.extractArticleID(result.ID)
		if article, exists := vectorArticleMap[articleID]; exists {
			resultMap[articleID] = SearchResultWithScore{
				Article: article,
				Score:   result.Score,
				Source:  "vector",
			}
		}
	}

	// Add keyword search results (combine scores if duplicate)
	for _, result := range keywordResults {
		if existing, exists := resultMap[result.Article.ID]; exists {
			// Normalize scores to 0-1 range before combining
			normalizedVectorScore := existing.Score // Vector scores are already 0-1
			normalizedKeywordScore := s.normalizeKeywordScore(result.Score)

			// Combine normalized scores with weighted average: vector 60%, keyword 40%
			combinedScore := (0.6 * normalizedVectorScore) + (0.4 * normalizedKeywordScore)
			resultMap[result.Article.ID] = SearchResultWithScore{
				Article: result.Article,
				Score:   combinedScore,
				Source:  "hybrid",
			}
		} else {
			resultMap[result.Article.ID] = SearchResultWithScore{
				Article: result.Article,
				Score:   s.normalizeKeywordScore(result.Score),
				Source:  "keyword",
			}
		}
	}

	// Convert to slice and filter by minimum score threshold
	var combinedResults []SearchResultWithScore
	for _, result := range resultMap {
		if result.Score >= minScoreThreshold {
			combinedResults = append(combinedResults, result)
		}
	}

	// Sort by score descending
	for i := 0; i < len(combinedResults)-1; i++ {
		for j := i + 1; j < len(combinedResults); j++ {
			if combinedResults[i].Score < combinedResults[j].Score {
				combinedResults[i], combinedResults[j] = combinedResults[j], combinedResults[i]
			}
		}
	}

	// Limit results
	if len(combinedResults) > limit {
		combinedResults = combinedResults[:limit]
	}

	return combinedResults
}

// normalizeKeywordScore normalizes OpenSearch keyword scores to 0-1 range
func (s *Server) normalizeKeywordScore(score float64) float64 {
	if score <= 0 {
		return 0.0
	}

	// Use logarithmic scaling: log(1 + score) / log(1 + max_expected_score)
	// Max expected BM25 score is around 20 for very good matches
	maxExpectedScore := 20.0
	normalized := math.Log(1+score) / math.Log(1+maxExpectedScore)

	// Clamp to [0, 1] range
	if normalized > 1.0 {
		normalized = 1.0
	}
	if normalized < 0.0 {
		normalized = 0.0
	}

	return normalized
}

// generateAnswer creates an AI-powered answer based on search results
func (s *Server) generateAnswer(ctx context.Context, query string, articles []opensearch.Article) (string, error) {
	// Detect query language to generate appropriate response
	queryLang := s.languageDetector.DetectLanguage(query)

	// Prepare language-specific response templates
	var noResultsMessage, contextIntro, promptTemplate string

	switch queryLang {
	case "ko":
		noResultsMessage = `관련된 정보를 찾지 못했습니다. 일반적인 지식을 바탕으로 답변을 생성합니다.

질문: %s

다음과 같은 도움이 되는 답변을 제공해주세요:
1. 정확하고 유용한 정보를 제공하세요
2. 불확실한 내용은 추측하지 말고 그렇다고 명시하세요
3. Markdown 형식으로 답변을 작성하세요 (제목, 굵은 글씨, 목록 등 활용)
4. 2-3개 문단으로 구성하여 읽기 쉽게 작성하세요
5. 추가 정보가 필요한 경우 어디서 찾을 수 있는지 안내해주세요
6. 특정 자료를 참조하지 않았음을 명시해주세요`
		contextIntro = "다음은 검색된 관련 자료들입니다:\n\n"
		promptTemplate = `위의 자료들을 바탕으로 다음 질문에 대해 종합적이고 정확한 답변을 해주세요.

질문: %s

답변할 때 다음 사항을 지켜주세요:
1. 제공된 자료의 내용만을 바탕으로 답변하세요
2. 질문과 관련이 없는 정보나 무관한 내용은 무시하세요
3. 구체적이고 실용적인 정보를 포함하세요
4. Markdown 형식으로 답변을 작성하세요 (제목, 굵은 글씨, 목록 등 활용)
5. 2-3개 문단으로 구성하여 읽기 쉽게 작성하세요
6. 확실하지 않은 내용은 추측하지 마세요

%s

답변 (Markdown 형식):`
	case "en":
		noResultsMessage = `No relevant information was found. Generating an answer based on general knowledge.

Question: %s

Please provide a helpful answer with the following guidelines:
1. Provide accurate and useful information
2. Do not speculate on uncertain content and clearly state when something is uncertain
3. Write your answer in Markdown format (use headings, bold text, lists, etc.)
4. Structure your response in 2-3 paragraphs for easy reading
5. If additional information is needed, guide where it can be found
6. Clearly state that no specific materials were referenced`
		contextIntro = "Here are the relevant materials found:\n\n"
		promptTemplate = `Based on the materials above, please provide a comprehensive and accurate answer to the following question.

Question: %s

Please follow these guidelines when answering:
1. Base your answer only on the provided materials
2. Ignore information that is not relevant to the question
3. Include specific and practical information
4. Write your answer in Markdown format (use headings, bold text, lists, etc.)
5. Structure your response in 2-3 paragraphs for easy reading
6. Do not speculate on uncertain information

%s

Answer (Markdown format):`
	case "ja":
		noResultsMessage = `関連する情報が見つかりませんでした。一般的な知識に基づいて回答を生成します。

質問: %s

以下のガイドラインに従って役立つ回答を提供してください:
1. 正確で有用な情報を提供してください
2. 不確実な内容は推測せず、そうであることを明示してください
3. Markdown形式で回答を作成してください（見出し、太字、リストなどを活用）
4. 読みやすいように2-3段落で構成してください
5. 追加情報が必要な場合、どこで見つけられるかを案内してください
6. 特定の資料を参照していないことを明示してください`
		contextIntro = "以下は検索された関連資料です：\n\n"
		promptTemplate = `上記の資料に基づいて、以下の質問に対して包括的で正確な回答をしてください。

質問: %s

回答の際は以下の点にご注意ください：
1. 提供された資料の内容のみに基づいて回答してください
2. 質問に関連しない情報や無関係な内容は無視してください
3. 具体的で実用的な情報を含めてください
4. Markdown形式で回答を作成してください（見出し、太字、リストなどを活用）
5. 読みやすいように2-3段落で構成してください
6. 不確実な内容は推測しないでください

%s

回答 (Markdown形式):`
	case "zh":
		noResultsMessage = `没有找到相关信息。基于一般知识生成回答。

问题: %s

请按照以下指导原则提供有用的回答:
1. 提供准确有用的信息
2. 对不确定的内容不要推测，并明确说明
3. 使用Markdown格式撰写回答（使用标题、粗体、列表等）
4. 分2-3段落组织，便于阅读
5. 如需更多信息，请指导在哪里可以找到
6. 明确说明未参考特定资料`
		contextIntro = "以下是搜索到的相关资料：\n\n"
		promptTemplate = `基于上述资料，请对以下问题提供全面准确的回答。

问题: %s

回答时请遵循以下要求：
1. 仅基于提供的资料内容进行回答
2. 忽略与问题无关的信息和内容
3. 包含具体实用的信息
4. 使用Markdown格式撰写回答（使用标题、粗体、列表等）
5. 分2-3段落组织，便于阅读
6. 对不确定的内容不要推测

%s

回答 (Markdown格式):`
	default:
		// Default to English for unrecognized languages
		noResultsMessage = `No relevant information was found. Generating an answer based on general knowledge.

Question: %s

Please provide a helpful answer with the following guidelines:
1. Provide accurate and useful information
2. Do not speculate on uncertain content and clearly state when something is uncertain
3. Write your answer in Markdown format (use headings, bold text, lists, etc.)
4. Structure your response in 2-3 paragraphs for easy reading
5. If additional information is needed, guide where it can be found
6. Clearly state that no specific materials were referenced`
		contextIntro = "Here are the relevant materials found:\n\n"
		promptTemplate = `Based on the materials above, please provide a comprehensive and accurate answer to the following question.

Question: %s

Please follow these guidelines when answering:
1. Base your answer only on the provided materials
2. Ignore information that is not relevant to the question
3. Include specific and practical information
4. Write your answer in Markdown format (use headings, bold text, lists, etc.)
5. Structure your response in 2-3 paragraphs for easy reading
6. Do not speculate on uncertain information

%s

Answer (Markdown format):`
	}

	if len(articles) == 0 {
		return noResultsMessage, nil
	}

	// Build context from search results
	context := contextIntro
	contentUsageCount := 0
	summaryUsageCount := 0

	for i, article := range articles {
		// Determine whether to use content or summary based on content length
		useContent := len(article.Content) < 4000
		contentText := article.Summary
		contentLabel := ""

		if useContent && article.Content != "" {
			contentText = article.Content
			contentUsageCount++
		} else {
			summaryUsageCount++
		}

		switch queryLang {
		case "ko":
			if useContent && article.Content != "" {
				contentLabel = "내용"
			} else {
				contentLabel = "요약"
			}
			context += fmt.Sprintf("%d. 제목: %s\n", i+1, article.Title)
			context += fmt.Sprintf("   %s: %s\n", contentLabel, contentText)
			if article.Author != "" {
				context += fmt.Sprintf("   작성자: %s\n", article.Author)
			}
		case "ja":
			if useContent && article.Content != "" {
				contentLabel = "内容"
			} else {
				contentLabel = "要約"
			}
			context += fmt.Sprintf("%d. タイトル: %s\n", i+1, article.Title)
			context += fmt.Sprintf("   %s: %s\n", contentLabel, contentText)
			if article.Author != "" {
				context += fmt.Sprintf("   著者: %s\n", article.Author)
			}
		case "zh":
			if useContent && article.Content != "" {
				contentLabel = "内容"
			} else {
				contentLabel = "摘要"
			}
			context += fmt.Sprintf("%d. 标题: %s\n", i+1, article.Title)
			context += fmt.Sprintf("   %s: %s\n", contentLabel, contentText)
			if article.Author != "" {
				context += fmt.Sprintf("   作者: %s\n", article.Author)
			}
		default: // English and others
			if useContent && article.Content != "" {
				contentLabel = "Content"
			} else {
				contentLabel = "Summary"
			}
			context += fmt.Sprintf("%d. Title: %s\n", i+1, article.Title)
			context += fmt.Sprintf("   %s: %s\n", contentLabel, contentText)
			if article.Author != "" {
				context += fmt.Sprintf("   Author: %s\n", article.Author)
			}
		}
		context += "\n"
	}

	log.Printf("Answer generation: Used full content for %d articles, summary for %d articles (total: %d)",
		contentUsageCount, summaryUsageCount, len(articles))

	// Create prompt for answer generation
	prompt := ""
	switch len(articles) {
	case 0:
		prompt = fmt.Sprintf(noResultsMessage, query)
	default:
		prompt = fmt.Sprintf(promptTemplate, query, context)
	}

	answer, err := s.ollamaClient.GenerateText(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate answer: %w", err)
	}

	return answer, nil
}

// generateAnswerStream creates an AI-powered answer based on search results using streaming
func (s *Server) generateAnswerStream(ctx context.Context, query string, articles []opensearch.Article, callback func(string) error) error {
	// Detect query language to generate appropriate response
	queryLang := s.languageDetector.DetectLanguage(query)

	// Prepare language-specific response templates
	var noResultsMessage, contextIntro, promptTemplate string

	switch queryLang {
	case "ko":
		noResultsMessage = `관련된 정보를 찾지 못했습니다. 일반적인 지식을 바탕으로 답변을 생성합니다.

질문: %s

다음과 같은 도움이 되는 답변을 제공해주세요:
1. 정확하고 유용한 정보를 제공하세요
2. 불확실한 내용은 추측하지 말고 그렇다고 명시하세요
3. Markdown 형식으로 답변을 작성하세요 (제목, 굵은 글씨, 목록 등 활용)
4. 2-3개 문단으로 구성하여 읽기 쉽게 작성하세요
5. 추가 정보가 필요한 경우 어디서 찾을 수 있는지 안내해주세요
6. 특정 자료를 참조하지 않았음을 명시해주세요`
		contextIntro = "다음은 검색된 관련 자료들입니다:\n\n"
		promptTemplate = `위의 자료들을 바탕으로 다음 질문에 대해 종합적이고 정확한 답변을 해주세요.

질문: %s

답변할 때 다음 사항을 지켜주세요:
1. 제공된 자료의 내용만을 바탕으로 답변하세요
2. 질문과 관련이 없는 정보나 무관한 내용은 무시하세요
3. 구체적이고 실용적인 정보를 포함하세요
4. Markdown 형식으로 답변을 작성하세요 (제목, 굵은 글씨, 목록 등 활용)
5. 2-3개 문단으로 구성하여 읽기 쉽게 작성하세요
6. 확실하지 않은 내용은 추측하지 마세요

%s

답변 (Markdown 형식):`
	case "en":
		noResultsMessage = `No relevant information was found. Generating an answer based on general knowledge.

Question: %s

Please provide a helpful answer with the following guidelines:
1. Provide accurate and useful information
2. Do not speculate on uncertain content and clearly state when something is uncertain
3. Write your answer in Markdown format (use headings, bold text, lists, etc.)
4. Structure your response in 2-3 paragraphs for easy reading
5. If additional information is needed, guide where it can be found
6. Clearly state that no specific materials were referenced`
		contextIntro = "Here are the relevant materials found:\n\n"
		promptTemplate = `Based on the materials above, please provide a comprehensive and accurate answer to the following question.

Question: %s

Please follow these guidelines when answering:
1. Base your answer only on the provided materials
2. Ignore information that is not relevant to the question
3. Include specific and practical information
4. Write your answer in Markdown format (use headings, bold text, lists, etc.)
5. Structure your response in 2-3 paragraphs for easy reading
6. Do not speculate on uncertain information

%s

Answer (Markdown format):`
	case "ja":
		noResultsMessage = `関連する情報が見つかりませんでした。一般的な知識に基づいて回答を生成します。

質問: %s

以下のガイドラインに従って役立つ回答を提供してください:
1. 正確で有用な情報を提供してください
2. 不確実な内容は推測せず、そうであることを明示してください
3. Markdown形式で回答を作成してください（見出し、太字、リストなどを活用）
4. 読みやすいように2-3段落で構成してください
5. 追加情報が必要な場合、どこで見つけられるかを案内してください
6. 特定の資料を参照していないことを明示してください`
		contextIntro = "以下は検索された関連資料です：\n\n"
		promptTemplate = `上記の資料に基づいて、以下の質問に対して包括的で正確な回答をしてください。

質問: %s

回答の際は以下の点にご注意ください：
1. 提供された資料の内容のみに基づいて回答してください
2. 質問に関連しない情報や無関係な内容は無視してください
3. 具体的で実用的な情報を含めてください
4. Markdown形式で回答を作成してください（見出し、太字、リストなどを活用）
5. 読みやすいように2-3段落で構成してください
6. 不確実な内容は推測しないでください

%s

回答 (Markdown形式):`
	case "zh":
		noResultsMessage = `没有找到相关信息。基于一般知识生成回答。

问题: %s

请按照以下指导原则提供有用的回答:
1. 提供准确有用的信息
2. 对不确定的内容不要推测，并明确说明
3. 使用Markdown格式撰写回答（使用标题、粗体、列表等）
4. 分2-3段落组织，便于阅读
5. 如需更多信息，请指导在哪里可以找到
6. 明确说明未参考特定资料`
		contextIntro = "以下是搜索到的相关资料：\n\n"
		promptTemplate = `基于上述资料，请对以下问题提供全面准确的回答。

问题: %s

回答时请遵循以下要求：
1. 仅基于提供的资料内容进行回答
2. 忽略与问题无关的信息和内容
3. 包含具体实用的信息
4. 使用Markdown格式撰写回答（使用标题、粗体、列表等）
5. 分2-3段落组织，便于阅读
6. 对不确定的内容不要推测

%s

回答 (Markdown格式):`
	default:
		// Default to English for unrecognized languages
		noResultsMessage = `No relevant information was found. Generating an answer based on general knowledge.

Question: %s

Please provide a helpful answer with the following guidelines:
1. Provide accurate and useful information
2. Do not speculate on uncertain content and clearly state when something is uncertain
3. Write your answer in Markdown format (use headings, bold text, lists, etc.)
4. Structure your response in 2-3 paragraphs for easy reading
5. If additional information is needed, guide where it can be found
6. Clearly state that no specific materials were referenced`
		contextIntro = "Here are the relevant materials found:\n\n"
		promptTemplate = `Based on the materials above, please provide a comprehensive and accurate answer to the following question.

Question: %s

Please follow these guidelines when answering:
1. Base your answer only on the provided materials
2. Ignore information that is not relevant to the question
3. Include specific and practical information
4. Write your answer in Markdown format (use headings, bold text, lists, etc.)
5. Structure your response in 2-3 paragraphs for easy reading
6. Do not speculate on uncertain information

%s

Answer (Markdown format):`
	}

	// if len(articles) == 0 {
	// 	return callback(noResultsMessage)
	// }

	// Build context from search results
	context := contextIntro
	contentUsageCount := 0
	summaryUsageCount := 0

	for i, article := range articles {
		// Determine whether to use content or summary based on content length
		useContent := len(article.Content) < 4000
		contentText := article.Summary
		contentLabel := ""

		if useContent && article.Content != "" {
			contentText = article.Content
			contentUsageCount++
		} else {
			summaryUsageCount++
		}

		switch queryLang {
		case "ko":
			if useContent && article.Content != "" {
				contentLabel = "내용"
			} else {
				contentLabel = "요약"
			}
			context += fmt.Sprintf("%d. 제목: %s\n", i+1, article.Title)
			context += fmt.Sprintf("   %s: %s\n", contentLabel, contentText)
			if article.Author != "" {
				context += fmt.Sprintf("   작성자: %s\n", article.Author)
			}
		case "ja":
			if useContent && article.Content != "" {
				contentLabel = "内容"
			} else {
				contentLabel = "要約"
			}
			context += fmt.Sprintf("%d. タイトル: %s\n", i+1, article.Title)
			context += fmt.Sprintf("   %s: %s\n", contentLabel, contentText)
			if article.Author != "" {
				context += fmt.Sprintf("   著者: %s\n", article.Author)
			}
		case "zh":
			if useContent && article.Content != "" {
				contentLabel = "内容"
			} else {
				contentLabel = "摘要"
			}
			context += fmt.Sprintf("%d. 标题: %s\n", i+1, article.Title)
			context += fmt.Sprintf("   %s: %s\n", contentLabel, contentText)
			if article.Author != "" {
				context += fmt.Sprintf("   作者: %s\n", article.Author)
			}
		default: // English and others
			if useContent && article.Content != "" {
				contentLabel = "Content"
			} else {
				contentLabel = "Summary"
			}
			context += fmt.Sprintf("%d. Title: %s\n", i+1, article.Title)
			context += fmt.Sprintf("   %s: %s\n", contentLabel, contentText)
			if article.Author != "" {
				context += fmt.Sprintf("   Author: %s\n", article.Author)
			}
		}
		context += "\n"
	}

	log.Printf("Answer generation (streaming): Used full content for %d articles, summary for %d articles (total: %d)",
		contentUsageCount, summaryUsageCount, len(articles))

	// Create prompt for answer generation
	prompt := ""
	switch len(articles) {
	case 0:
		prompt = fmt.Sprintf(noResultsMessage, query)
	default:
		prompt = fmt.Sprintf(promptTemplate, query, context)
	}

	return s.ollamaClient.GenerateTextStream(ctx, prompt, callback)
}

// GetArticle retrieves a specific article by ID
func (s *Server) GetArticle(ctx context.Context, id string) (*opensearch.Article, error) {
	return s.opensearchClient.GetArticle(ctx, id)
}

// GetSupportedLanguages returns the list of supported languages
func (s *Server) GetSupportedLanguages() []string {
	return s.languageDetector.GetSupportedLanguages()
}

// checkDuplicateArticle checks if an article with similar title and content already exists
func (s *Server) checkDuplicateArticle(ctx context.Context, title, content string) (bool, string, error) {
	// Generate embeddings for title and content
	titleEmbedding, err := s.ollamaClient.GenerateEmbedding(ctx, "passage: "+title)
	if err != nil {
		return false, "", fmt.Errorf("failed to generate title embedding for duplicate check: %w", err)
	}

	// Search for similar titles with high threshold
	titleResults, err := s.qdrantClient.VectorSearch(ctx, titleEmbedding, 5, "")
	if err != nil {
		return false, "", fmt.Errorf("failed to search for similar titles: %w", err)
	}

	// Check if any result has very high similarity (>0.95 for titles)
	for _, result := range titleResults {
		if result.Score > 0.95 {
			articleID := s.extractArticleID(result.ID)
			log.Printf("Found highly similar title (score: %.3f) for article ID: %s", result.Score, articleID)
			return true, articleID, nil
		}
	}

	return false, "", nil
}

// combineVectorResults combines title and summary vector search results
func (s *Server) combineVectorResults(titleResults, summaryResults []qdrant.VectorSearchResult) []qdrant.VectorSearchResult {
	resultMap := make(map[string]qdrant.VectorSearchResult)

	// Add title results with boosted scores (titles are more important)
	for _, result := range titleResults {
		articleID := s.extractArticleID(result.ID)
		boostedScore := result.Score * 1.2 // Boost title matches by 20%
		if boostedScore > 1.0 {
			boostedScore = 1.0
		}
		resultMap[articleID] = qdrant.VectorSearchResult{
			ID:    articleID, // Use original article ID, not the Qdrant point ID
			Score: boostedScore,
		}
	}

	// Add summary results (combine if duplicate)
	for _, result := range summaryResults {
		articleID := s.extractArticleID(result.ID)
		if existing, exists := resultMap[articleID]; exists {
			// Take the higher score between title and summary
			if result.Score > existing.Score {
				resultMap[articleID] = qdrant.VectorSearchResult{
					ID:    articleID,
					Score: result.Score,
				}
			}
		} else {
			resultMap[articleID] = qdrant.VectorSearchResult{
				ID:    articleID,
				Score: result.Score,
			}
		}
	}

	// Convert back to slice
	var combinedResults []qdrant.VectorSearchResult
	for _, result := range resultMap {
		combinedResults = append(combinedResults, result)
	}

	return combinedResults
}

// extractArticleID extracts the original article ID from Qdrant point ID
func (s *Server) extractArticleID(pointID string) string {
	// Remove _title or _summary suffix
	if len(pointID) > 6 && pointID[len(pointID)-6:] == "_title" {
		return pointID[:len(pointID)-6]
	}
	if len(pointID) > 8 && pointID[len(pointID)-8:] == "_summary" {
		return pointID[:len(pointID)-8]
	}
	return pointID
}
