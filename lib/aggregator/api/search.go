package api

import (
	"context"
	"fmt"
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"

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

	// Calculate expanded search size (2x the requested size for each search)
	expandedSize := size * 2

	// 4. Perform parallel searches
	// 4a. Vector search with Qdrant (search both title and summary embeddings)
	// Request expandedSize * 2 to account for title and summary results that will be combined
	allVectorResults, err := s.qdrantClient.VectorSearch(ctx, queryEmbedding, uint64(expandedSize*2), queryLang)
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

	// Combine and deduplicate vector results, limiting to expandedSize
	// Combine and deduplicate vector results, limiting to expandedSize
	combinedVectorResults := s.combineVectorResults(titleVectorResults, summaryVectorResults, expandedSize)

	// 4b. Keyword search with OpenSearch
	// Request expandedSize to get more candidates for better score combination
	keywordResp, err := s.opensearchClient.KeywordSearch(ctx, req.Query, queryLang, expandedSize, req.From)
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

	// 6.5. Validate search relevance using LLM
	filteredResults, err := s.validateSearchRelevance(ctx, req.Query, combinedResults)
	if err != nil {
		log.Printf("Failed to validate search relevance: %v", err)
		// Continue with original results if validation fails
		filteredResults = combinedResults
	}

	// 7. Extract articles for AI answer generation
	articles := make([]opensearch.Article, len(filteredResults))
	for i, result := range filteredResults {
		articles[i] = result.Article
	}

	// 8. Generate AI answer using search results
	answer, err := s.generateAnswer(ctx, req.Query, articles)
	if err != nil {
		return nil, fmt.Errorf("failed to generate answer: %w", err)
	}

	return &SearchResponse{
		Answer:  answer,
		Sources: filteredResults,
		Took:    keywordResp.Took, // Use keyword search timing for now
	}, nil
}

// combineSearchResults combines vector and keyword search results with scoring
func (s *Server) combineSearchResults(vectorResults []qdrant.VectorSearchResult, vectorArticles []opensearch.Article, keywordResults []opensearch.SearchResult, limit int) []SearchResultWithScore {
	const minScoreThreshold = 0.35   // Minimum score threshold for quality filtering
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
	x0 := 20.0

	sigmoid := 1.0 / (1.0 + math.Exp(-k*(score-x0)))

	return sigmoid
}

// combineVectorResults combines title and summary vector search results
func (s *Server) combineVectorResults(titleResults, summaryResults []qdrant.VectorSearchResult, limit int) []qdrant.VectorSearchResult {
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

	// Convert to slice and sort by score descending
	var combinedResults []qdrant.VectorSearchResult
	for _, result := range resultMap {
		combinedResults = append(combinedResults, result)
	}

	// Sort by score descending
	for i := 0; i < len(combinedResults)-1; i++ {
		for j := i + 1; j < len(combinedResults); j++ {
			if combinedResults[i].Score < combinedResults[j].Score {
				combinedResults[i], combinedResults[j] = combinedResults[j], combinedResults[i]
			}
		}
	}

	// Limit results to requested size
	if len(combinedResults) > limit {
		combinedResults = combinedResults[:limit]
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

// validateSearchRelevance uses LLM to check if search results are relevant to the user's query
func (s *Server) validateSearchRelevance(ctx context.Context, query string, results []SearchResultWithScore) ([]SearchResultWithScore, error) {
	if len(results) == 0 {
		return results, nil
	}

	// Detect query language for appropriate prompt
	queryLang := s.languageDetector.DetectLanguage(query)

	var relevancePrompt string
	switch queryLang {
	case "ko":
		relevancePrompt = `다음 질문에 대해 제공된 문서들이 얼마나 관련성이 있는지 평가해주세요.

질문: %s

문서들:
%s

각 문서에 대해 다음 형식으로 0-10 점수를 매겨주세요 (10점이 가장 관련성이 높음):
문서1: [점수]
문서2: [점수]
...

평가 기준:
- 질문의 핵심 키워드와 일치하는 정도
- 문서가 질문에 답변할 수 있는 정보를 포함하는 정도
- 문맥상 관련성
- 5점 미만은 관련성이 낮은 것으로 간주됩니다

점수만 제공하고 추가 설명은 하지 마세요.`
	case "ja":
		relevancePrompt = `以下の質問に対して、提供された文書がどの程度関連性があるかを評価してください。

質問: %s

文書:
%s

各文書について以下の形式で0-10のスコアを付けてください（10点が最も関連性が高い）:
文書1: [スコア]
文書2: [スコア]
...

評価基準:
- 質問の核心キーワードとの一致度
- 文書が質問に答えられる情報を含む度合い
- 文脈上の関連性
- 5点未満は関連性が低いと見なされます

スコアのみを提供し、追加説明はしないでください。`
	case "zh":
		relevancePrompt = `请评估以下文档对给定问题的相关性。

问题: %s

文档:
%s

请为每个文档按以下格式评分0-10分（10分表示最相关）:
文档1: [分数]
文档2: [分数]
...

评分标准:
- 与问题核心关键词的匹配程度
- 文档包含能回答问题的信息程度
- 上下文相关性
- 5分以下被认为相关性较低

只提供分数，不要额外说明。`
	default: // English
		relevancePrompt = `Please evaluate how relevant the provided documents are to the given question.

Question: %s

Documents:
%s

Rate each document with a score from 0-10 (10 being most relevant) in the following format:
Document1: [score]
Document2: [score]
...

Evaluation criteria:
- Match with core keywords in the question
- Degree to which the document contains information that can answer the question
- Contextual relevance
- Scores below 5 are considered low relevance

Provide only scores without additional explanations.`
	}

	// Build documents string for LLM evaluation
	var documentsText string
	for i, result := range results {
		// Use summary for relevance check to reduce token usage
		content := result.Article.Summary
		if content == "" {
			// Truncate content if summary is not available
			content = result.Article.Content
			if len(content) > 1000 {
				content = content[:1000] + "..."
			}
		}

		documentsText += fmt.Sprintf("문서%d: 제목: %s\n내용: %s\n\n", i+1, result.Article.Title, content)
	}

	prompt := fmt.Sprintf(relevancePrompt, query, documentsText)

	// Get LLM evaluation
	evaluation, err := s.ollamaClient.GenerateText(ctx, prompt)
	if err != nil {
		log.Printf("Failed to get relevance evaluation from LLM: %v", err)
		// Return original results if LLM evaluation fails
		return results, nil
	}

	log.Printf("LLM Relevance Evaluation:\n%s", evaluation)

	// Parse relevance scores from LLM response
	relevanceScores := s.parseRelevanceScores(evaluation, len(results))

	// Filter results based on relevance scores
	var filteredResults []SearchResultWithScore
	const relevanceThreshold = 5.0 // Minimum relevance score

	for i, result := range results {
		if i < len(relevanceScores) {
			relevanceScore := relevanceScores[i]
			if relevanceScore >= relevanceThreshold {
				log.Printf("Document relevant: ID=%s, SearchScore=%.4f, RelevanceScore=%.1f",
					result.Article.ID, result.Score, relevanceScore)

				// Optionally adjust the final score based on relevance
				// Combine search score (70%) with relevance score normalized to 0-1 (30%)
				adjustedScore := (result.Score * 0.7) + ((relevanceScore / 10.0) * 0.3)
				result.Score = adjustedScore

				filteredResults = append(filteredResults, result)
			} else {
				log.Printf("Document filtered (low relevance): ID=%s, SearchScore=%.4f, RelevanceScore=%.1f",
					result.Article.ID, result.Score, relevanceScore)
			}
		} else {
			// If we couldn't parse the score, keep the result
			log.Printf("Document kept (unparsed relevance): ID=%s, SearchScore=%.4f",
				result.Article.ID, result.Score)
			filteredResults = append(filteredResults, result)
		}
	}

	log.Printf("Relevance filtering: %d -> %d documents", len(results), len(filteredResults))

	return filteredResults, nil
}

// parseRelevanceScores parses LLM response to extract relevance scores
func (s *Server) parseRelevanceScores(evaluation string, expectedCount int) []float64 {
	scores := make([]float64, 0, expectedCount)

	// Simple parsing - look for patterns like "문서1: 8", "Document1: 7", etc.
	lines := strings.Split(evaluation, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for patterns like "문서1: 8", "Document1: 7", "文書1: 6"
		patterns := []string{
			`문서\d+:\s*(\d+(?:\.\d+)?)`,
			`Document\d+:\s*(\d+(?:\.\d+)?)`,
			`文書\d+:\s*(\d+(?:\.\d+)?)`,
			`文档\d+:\s*(\d+(?:\.\d+)?)`,
		}

		for _, pattern := range patterns {
			re := regexp.MustCompile(pattern)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				if score, err := strconv.ParseFloat(matches[1], 64); err == nil {
					scores = append(scores, score)
					break
				}
			}
		}
	}

	return scores
}
