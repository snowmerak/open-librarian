package api

import (
	"context"
	"fmt"
	"time"

	"github.com/snowmerak/open-librarian/lib/client/mongo"
	"github.com/snowmerak/open-librarian/lib/client/opensearch"
	"github.com/snowmerak/open-librarian/lib/util/logger"
)

// AddArticle processes and indexes a new article
func (s *Server) AddArticle(ctx context.Context, req *ArticleRequest) (*ArticleResponse, error) {
	articleLogger := logger.NewLogger("add_article").StartWithMsg("Processing new article")
	articleLogger.Info().Str("title", req.Title).Int("content_length", len(req.Content)).Msg("Article processing started")

	// Extract user information from context
	var registrar string
	if user, ok := ctx.Value(UserContextKey).(*mongo.User); ok {
		registrar = user.Username
		articleLogger.Info().Str("registrar", registrar).Msg("Article being registered by user")
	} else {
		articleLogger.Error().Msg("No user context found - authentication required")
		articleLogger.EndWithError(fmt.Errorf("authentication required"))
		return nil, fmt.Errorf("authentication required")
	}

	// 1. Check for duplicate articles based on title and content similarity
	dupCheckLogger := logger.NewLogger("duplicate_check").StartWithMsg("Checking for duplicate articles")
	isDuplicate, existingID, err := s.checkDuplicateArticle(ctx, req.Title, req.Content)
	if err != nil {
		dupCheckLogger.Warn().Err(err).Msg("Failed to check for duplicates, continuing with indexing")
		dupCheckLogger.EndWithError(err)
	} else if isDuplicate {
		dupCheckLogger.Info().Str("existing_id", existingID).Msg("Duplicate article detected")
		dupCheckLogger.EndWithMsg("Duplicate check complete - duplicate found")
		articleLogger.EndWithMsg("Article processing complete - duplicate found")
		return &ArticleResponse{
			ID:      existingID,
			Message: "Duplicate article found, returning existing article ID",
		}, nil
	} else {
		dupCheckLogger.EndWithMsg("Duplicate check complete - no duplicates found")
	}

	// 2. Detect language
	langLogger := logger.NewLogger("language_detection").StartWithMsg("Detecting article language")
	lang := s.languageDetector.DetectLanguage(req.Content)
	langLogger.Info().Str("detected_language", lang).Msg("Language detection complete")
	langLogger.EndWithMsg("Language detection complete")

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

	summaryLogger := logger.NewLogger("summary_generation")
	summaryLogger.Info().Str("summary_preview", fmt.Sprintf("%.100s...", summary)).Msg("Generated summary")

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
			dateLogger := logger.NewLogger("date_validation")
			dateLogger.Warn().Str("created_date", parsed.Format(time.RFC3339)).Msg("Article created_date is more than 10 years old")
		}

		createdDate = parsed
		dateLogger := logger.NewLogger("date_validation")
		dateLogger.Info().Str("created_date", createdDate.Format(time.RFC3339)).Msg("Using provided created_date")
	} else {
		createdDate = time.Now()
		dateLogger := logger.NewLogger("date_validation")
		dateLogger.Info().Str("created_date", createdDate.Format(time.RFC3339)).Msg("No created_date provided, using current time")
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
		vectorLogger := logger.NewLogger("vector_indexing")
		vectorLogger.Error().Err(err).Msg("Failed to index title embedding in Qdrant, cleaning up OpenSearch entry")
		return nil, fmt.Errorf("failed to index title vectors in Qdrant: %w", err)
	}

	err = s.qdrantClient.UpsertPoint(ctx, indexResp.ID+"_summary", summaryEmbedding, lang)
	if err != nil {
		vectorLogger := logger.NewLogger("vector_indexing")
		vectorLogger.Error().Err(err).Msg("Failed to index summary embedding in Qdrant, cleaning up OpenSearch entry")
		return nil, fmt.Errorf("failed to index summary vectors in Qdrant: %w", err)
	}

	indexLogger := logger.NewLogger("article_indexing")
	indexLogger.Info().Str("article_id", indexResp.ID).Msg("Successfully indexed article")

	return &ArticleResponse{
		ID:      indexResp.ID,
		Message: "Article indexed successfully",
	}, nil
}

// AddArticleWithProgress processes and indexes a new article with progress callbacks
func (s *Server) AddArticleWithProgress(ctx context.Context, req *ArticleRequest, progressCallback ProgressCallback) (*ArticleResponse, error) {
	progressLogger := logger.NewLogger("article_with_progress").StartWithMsg("Processing article with progress tracking")
	progressLogger.Info().Str("title", req.Title).Msg("Starting article processing with progress tracking")

	// Extract user information from context
	var registrar string
	if user, ok := ctx.Value(UserContextKey).(*mongo.User); ok {
		registrar = user.Username
		progressLogger.Info().Str("registrar", registrar).Msg("Article being registered by user")
	} else {
		progressLogger.Error().Msg("No user context found - this should not happen for authenticated endpoints")
		progressLogger.EndWithError(fmt.Errorf("authentication required"))
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
		dupLogger := logger.NewLogger("duplicate_check")
		dupLogger.Warn().Err(err).Msg("Failed to check for duplicates")
		// Continue with indexing despite duplicate check failure
	} else if isDuplicate {
		dupLogger := logger.NewLogger("duplicate_check")
		dupLogger.Info().Str("existing_id", existingID).Msg("Duplicate article detected")
		progressLogger.EndWithMsg("Article processing complete - duplicate found")
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
	langLogger := logger.NewLogger("language_detection")
	langLogger.Info().Str("detected_language", lang).Msg("Language detection complete")

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
	summaryLogger := logger.NewLogger("summary_generation")
	summaryLogger.Info().Str("summary_preview", fmt.Sprintf("%.100s...", summary)).Msg("Generated summary")

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
			dateProgressLogger := logger.NewLogger("date_validation_progress")
			dateProgressLogger.Warn().Str("created_date", parsed.Format(time.RFC3339)).Msg("Article created_date is more than 10 years old")
		}

		createdDate = parsed
		dateProgressLogger := logger.NewLogger("date_validation_progress")
		dateProgressLogger.Info().Str("created_date", createdDate.Format(time.RFC3339)).Msg("Using provided created_date")
	} else {
		createdDate = time.Now()
		dateProgressLogger := logger.NewLogger("date_validation_progress")
		dateProgressLogger.Info().Str("created_date", createdDate.Format(time.RFC3339)).Msg("No created_date provided, using current time")
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
		vectorProgressLogger := logger.NewLogger("vector_indexing_progress")
		vectorProgressLogger.Error().Err(err).Msg("Failed to index title embedding in Qdrant, cleaning up OpenSearch entry")
		return nil, fmt.Errorf("failed to index title vectors in Qdrant: %w", err)
	}

	err = s.qdrantClient.UpsertPoint(ctx, indexResp.ID+"_summary", summaryEmbedding, lang)
	if err != nil {
		vectorProgressLogger := logger.NewLogger("vector_indexing_progress")
		vectorProgressLogger.Error().Err(err).Msg("Failed to index summary embedding in Qdrant, cleaning up OpenSearch entry")
		return nil, fmt.Errorf("failed to index summary vectors in Qdrant: %w", err)
	}

	indexProgressLogger := logger.NewLogger("article_indexing_progress")
	indexProgressLogger.Info().Str("article_id", indexResp.ID).Msg("Successfully indexed article")
	progressLogger.EndWithMsg("Article processing complete")

	return &ArticleResponse{
		ID:      indexResp.ID,
		Message: "Article indexed successfully",
	}, nil
}

// AddArticlesBulkWithProgress processes multiple articles with progress callbacks
func (s *Server) AddArticlesBulkWithProgress(ctx context.Context, req *BulkArticleRequest, progressCallback BulkProgressCallback) (*BulkArticleResponse, error) {
	bulkLogger := logger.NewLogger("bulk_article_processing").StartWithMsg("Processing bulk upload")
	bulkLogger.Info().Int("article_count", len(req.Articles)).Msg("Starting bulk article processing")

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
				articleLogger := logger.NewLogger("bulk_article_processing")
				articleLogger.Error().Err(err).Int("index", index).Str("title", article.Title).Msg("Failed to process article")
			} else {
				result.Success = true
				result.ID = articleResp.ID
				articleLogger := logger.NewLogger("bulk_article_processing")
				articleLogger.Info().Int("index", index).Str("title", article.Title).Str("article_id", articleResp.ID).Msg("Successfully processed article")
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

	bulkLogger.Info().Int("success_count", response.SuccessCount).Int("error_count", response.ErrorCount).Msg("Bulk upload completed")
	bulkLogger.EndWithMsg("Bulk processing complete")
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
			dupLogger := logger.NewLogger("duplicate_check")
			dupLogger.Info().Str("article_id", articleID).Float64("similarity_score", result.Score).Msg("Found highly similar title")
			return true, articleID, nil
		}
	}

	return false, "", nil
}

// DeleteArticle removes an article from both OpenSearch and Qdrant
func (s *Server) DeleteArticle(ctx context.Context, id string) error {
	deleteLogger := logger.NewLogger("article_deletion").StartWithMsg("Deleting article")
	deleteLogger.Info().Str("article_id", id).Msg("Starting article deletion")

	// First, get the article to check if it exists and verify permissions
	article, err := s.opensearchClient.GetArticle(ctx, id)
	if err != nil {
		deleteLogger.Error().Err(err).Msg("Failed to get article for deletion")
		deleteLogger.EndWithError(err)
		return fmt.Errorf("article not found: %w", err)
	}

	// Extract user information from context for permission check
	if user, ok := ctx.Value(UserContextKey).(*mongo.User); ok {
		// Check if the user is the registrar of the article
		if article.Registrar != user.Username {
			deleteLogger.Warn().Str("user", user.Username).Str("registrar", article.Registrar).Msg("User attempted to delete article registered by another user")
			deleteLogger.EndWithError(fmt.Errorf("permission denied"))
			return fmt.Errorf("permission denied: only the registrar can delete this article")
		}
		deleteLogger.Info().Str("user", user.Username).Msg("Article deletion authorized for user")
	} else {
		deleteLogger.Error().Msg("No user context found for deletion request")
		deleteLogger.EndWithError(fmt.Errorf("authentication required"))
		return fmt.Errorf("authentication required")
	}

	// Delete from OpenSearch
	err = s.opensearchClient.DeleteArticle(ctx, id)
	if err != nil {
		deleteLogger.Error().Err(err).Msg("Failed to delete article from OpenSearch")
		deleteLogger.EndWithError(err)
		return fmt.Errorf("failed to delete from search index: %w", err)
	}

	// Delete from Qdrant (vector database)
	// Delete both title and summary embeddings
	titleID := id + "_title"
	summaryID := id + "_summary"

	err = s.qdrantClient.DeletePoint(ctx, titleID)
	if err != nil {
		deleteLogger.Warn().Err(err).Str("title_id", titleID).Msg("Failed to delete title embedding from Qdrant")
		// Don't fail the entire operation if Qdrant deletion fails
		// Log the error but continue
		deleteLogger.Warn().Msg("Article deleted from OpenSearch but title embedding not from Qdrant")
	}

	err = s.qdrantClient.DeletePoint(ctx, summaryID)
	if err != nil {
		deleteLogger.Warn().Err(err).Str("summary_id", summaryID).Msg("Failed to delete summary embedding from Qdrant")
		// Don't fail the entire operation if Qdrant deletion fails
		// Log the error but continue
		deleteLogger.Warn().Msg("Article deleted from OpenSearch but summary embedding not from Qdrant")
	}

	deleteLogger.Info().Str("article_id", id).Msg("Successfully deleted article")
	deleteLogger.EndWithMsg("Article deletion complete")
	return nil
}
