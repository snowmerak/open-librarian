package api

import (
	"context"
	"fmt"
	"log"
	"math"

	"github.com/snowmerak/open-librarian/lib/client/opensearch"
	"github.com/snowmerak/open-librarian/lib/client/qdrant"
)

// Search performs hybrid search combining vector and keyword search
func (s *Server) Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	// Remove unnecessary log
	// log.Printf("Searching for: %s", req.Query)

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

	// Separate title and summary results and log vector search scores
	var titleVectorResults, summaryVectorResults []qdrant.VectorSearchResult
	for _, result := range allVectorResults {
		if len(result.ID) > 6 && result.ID[len(result.ID)-6:] == "_title" {
			titleVectorResults = append(titleVectorResults, result)
			log.Printf("Vector search (title): ID=%s, Score=%.4f", s.extractArticleID(result.ID), result.Score)
		} else if len(result.ID) > 8 && result.ID[len(result.ID)-8:] == "_summary" {
			summaryVectorResults = append(summaryVectorResults, result)
			log.Printf("Vector search (summary): ID=%s, Score=%.4f", s.extractArticleID(result.ID), result.Score)
		}
	}

	// Combine and deduplicate vector results
	combinedVectorResults := s.combineVectorResults(titleVectorResults, summaryVectorResults)

	// 4b. Keyword search with OpenSearch
	keywordResp, err := s.opensearchClient.KeywordSearch(ctx, req.Query, queryLang, size*2, req.From)
	// keywordResp, err := s.opensearchClient.KeywordSearch(ctx, req.Query, queryLang, size*2, req.From)
	if err != nil {
		log.Printf("Keyword search failed: %v", err)
		keywordResp = &opensearch.SearchResponse{Results: []opensearch.SearchResult{}}
	}

	// Log keyword search scores
	for _, result := range keywordResp.Results {
		log.Printf("Keyword search: ID=%s, Score=%.4f", result.Article.ID, result.Score)
	}

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

	// Remove unnecessary log
	// log.Printf("Extracted unique article IDs: %v", vectorArticleIDs)

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
	const minScoreThreshold = 0.45   // Minimum score threshold for quality filtering
	const singleSourcePenalty = 0.75 // Penalty for non-hybrid results

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

			// Log score combination details
			log.Printf("Score combination: ID=%s, Vector=%.4f, Keyword=%.4f->%.4f, Combined=%.4f",
				result.Article.ID, normalizedVectorScore, result.Score, normalizedKeywordScore, combinedScore)

			resultMap[result.Article.ID] = SearchResultWithScore{
				Article: result.Article,
				Score:   combinedScore,
				Source:  "hybrid",
			}
		} else {
			normalizedScore := s.normalizeKeywordScore(result.Score)
			// Apply penalty for keyword-only results
			penalizedScore := normalizedScore * singleSourcePenalty
			log.Printf("Keyword only: ID=%s, Original=%.4f, Normalized=%.4f, Penalized=%.4f",
				result.Article.ID, result.Score, normalizedScore, penalizedScore)

			resultMap[result.Article.ID] = SearchResultWithScore{
				Article: result.Article,
				Score:   penalizedScore,
				Source:  "keyword",
			}
		}
	}

	// Apply penalty to vector-only results
	for articleID, result := range resultMap {
		if result.Source == "vector" {
			penalizedScore := result.Score * singleSourcePenalty
			log.Printf("Vector only: ID=%s, Original=%.4f, Penalized=%.4f",
				articleID, result.Score, penalizedScore)

			result.Score = penalizedScore
			resultMap[articleID] = result
		}
	}

	// Convert to slice and filter by minimum score threshold
	var combinedResults []SearchResultWithScore
	for _, result := range resultMap {
		if result.Score >= minScoreThreshold {
			combinedResults = append(combinedResults, result)
			log.Printf("Final result: ID=%s, Score=%.4f, Source=%s", result.Article.ID, result.Score, result.Source)
		} else {
			log.Printf("Filtered out (low score): ID=%s, Score=%.4f, Source=%s", result.Article.ID, result.Score, result.Source)
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

// normalizeKeywordScore normalizes OpenSearch keyword scores to 0-1 range using sigmoid function
func (s *Server) normalizeKeywordScore(score float64) float64 {
	if score <= 0 {
		return 0.0
	}

	// Use sigmoid function: 1 / (1 + exp(-k * (x - x0)))
	// k = steepness parameter (higher = steeper curve)
	// x0 = midpoint (score that maps to 0.5)

	// For BM25 scores, typical good matches are around 5-15
	// Set midpoint at 8 (maps to 0.5) and steepness k=0.5
	k := 0.65
	x0 := 18.0

	sigmoid := 1.0 / (1.0 + math.Exp(-k*(score-x0)))

	return sigmoid
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
