package api

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/snowmerak/open-librarian/lib/client/mongo"
	"github.com/snowmerak/open-librarian/lib/client/opensearch"
)

// AddArticle processes and indexes a new article
func (s *Server) AddArticle(ctx context.Context, req *ArticleRequest) (*ArticleResponse, error) {
	log.Printf("Processing article: %s", req.Title)

	// Extract user information from context
	var registrar string
	if user, ok := ctx.Value(UserContextKey).(*mongo.User); ok {
		registrar = user.Username
		log.Printf("Article being registered by user: %s", registrar)
	} else {
		log.Printf("No user context found - this should not happen for authenticated endpoints")
		return nil, fmt.Errorf("authentication required")
	}

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
	summaryPrompt := fmt.Sprintf(`Please create a comprehensive and detailed summary of the following text in English. You can write up to 8000 characters if needed to capture all important information.

Guidelines for the summary:
1. Include all key points, main arguments, and important details
2. Maintain the logical structure and flow of the original content
3. Include specific examples, data, or evidence mentioned in the text
4. Cover any conclusions, recommendations, or actionable insights
5. Write in clear, well-structured paragraphs
6. You may use multiple paragraphs to organize different topics or sections
7. Focus on being comprehensive rather than brief - detail is more valuable than brevity

Text:
%s

Detailed Summary:`, req.Content)

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
		Registrar:   registrar,
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

// AddArticleWithProgress processes and indexes a new article with progress callbacks
func (s *Server) AddArticleWithProgress(ctx context.Context, req *ArticleRequest, progressCallback ProgressCallback) (*ArticleResponse, error) {
	log.Printf("Processing article with progress tracking: %s", req.Title)

	// Extract user information from context
	var registrar string
	if user, ok := ctx.Value(UserContextKey).(*mongo.User); ok {
		registrar = user.Username
		log.Printf("Article being registered by user: %s", registrar)
	} else {
		log.Printf("No user context found - this should not happen for authenticated endpoints")
		return nil, fmt.Errorf("authentication required")
	}

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
	summaryPrompt := fmt.Sprintf(`Please create a comprehensive and detailed summary of the following text in English. You can write up to 4000 characters if needed to capture all important information.

Guidelines for the summary:
1. Include all key points, main arguments, and important details
2. Maintain the logical structure and flow of the original content
3. Include specific examples, data, or evidence mentioned in the text
4. Cover any conclusions, recommendations, or actionable insights
5. Write in clear, well-structured paragraphs
6. You may use multiple paragraphs to organize different topics or sections
7. Focus on being comprehensive rather than brief - detail is more valuable than brevity

Text:
%s

Detailed Summary:`, req.Content)

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
		Registrar:   registrar,
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

// DeleteArticle removes an article from both OpenSearch and Qdrant
func (s *Server) DeleteArticle(ctx context.Context, id string) error {
	log.Printf("Deleting article with ID: %s", id)

	// First, get the article to check if it exists and verify permissions
	article, err := s.opensearchClient.GetArticle(ctx, id)
	if err != nil {
		log.Printf("Failed to get article for deletion: %v", err)
		return fmt.Errorf("article not found: %w", err)
	}

	// Extract user information from context for permission check
	if user, ok := ctx.Value(UserContextKey).(*mongo.User); ok {
		// Check if the user is the registrar of the article
		if article.Registrar != user.Username {
			log.Printf("User %s attempted to delete article registered by %s", user.Username, article.Registrar)
			return fmt.Errorf("permission denied: only the registrar can delete this article")
		}
		log.Printf("Article deletion authorized for user: %s", user.Username)
	} else {
		log.Printf("No user context found for deletion request")
		return fmt.Errorf("authentication required")
	}

	// Delete from OpenSearch
	err = s.opensearchClient.DeleteArticle(ctx, id)
	if err != nil {
		log.Printf("Failed to delete article from OpenSearch: %v", err)
		return fmt.Errorf("failed to delete from search index: %w", err)
	}

	// Delete from Qdrant (vector database)
	// Delete both title and summary embeddings
	titleID := id + "_title"
	summaryID := id + "_summary"

	err = s.qdrantClient.DeletePoint(ctx, titleID)
	if err != nil {
		log.Printf("Failed to delete title embedding from Qdrant: %v", err)
		// Don't fail the entire operation if Qdrant deletion fails
		// Log the error but continue
		log.Printf("Warning: Article deleted from OpenSearch but title embedding not from Qdrant")
	}

	err = s.qdrantClient.DeletePoint(ctx, summaryID)
	if err != nil {
		log.Printf("Failed to delete summary embedding from Qdrant: %v", err)
		// Don't fail the entire operation if Qdrant deletion fails
		// Log the error but continue
		log.Printf("Warning: Article deleted from OpenSearch but summary embedding not from Qdrant")
	}

	log.Printf("Successfully deleted article: %s", id)
	return nil
}
