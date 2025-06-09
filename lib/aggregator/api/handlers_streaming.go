package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/snowmerak/open-librarian/lib/client/opensearch"
	"github.com/snowmerak/open-librarian/lib/client/qdrant"
)

// SearchStreamHandler handles search requests with SSE streaming
func (h *HTTPServer) SearchStreamHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid_json", "Invalid JSON format")
		return
	}

	// Basic validation
	if req.Query == "" {
		writeErrorResponse(w, http.StatusBadRequest, "missing_query", "Query is required")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Send initial message
	sendSSEMessage(w, "status", "Starting search...")
	w.(http.Flusher).Flush()

	// 1. Detect query language
	queryLang := h.server.languageDetector.DetectLanguage(req.Query)

	// 2. Generate query embedding for vector search
	queryEmbedding, err := h.server.ollamaClient.GenerateEmbedding(ctx, "query: "+req.Query)
	if err != nil {
		sendSSEMessage(w, "error", fmt.Sprintf("Failed to generate query embedding: %v", err))
		return
	}

	sendSSEMessage(w, "status", "Performing search...")
	w.(http.Flusher).Flush()

	// 3. Set default size if not provided
	size := req.Size
	if size == 0 {
		size = 5 // Default to top 5 results for AI answer generation
	}

	// 4. Perform parallel searches
	// 4a. Vector search with Qdrant
	allVectorResults, err := h.server.qdrantClient.VectorSearch(ctx, queryEmbedding, uint64(size*4), queryLang)
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
	combinedVectorResults := h.server.combineVectorResults(titleVectorResults, summaryVectorResults)

	// 4b. Keyword search with OpenSearch
	keywordResp, err := h.server.opensearchClient.KeywordSearch(ctx, req.Query, queryLang, size*2, req.From)
	if err != nil {
		log.Printf("Keyword search failed: %v", err)
		keywordResp = &opensearch.SearchResponse{Results: []opensearch.SearchResult{}}
	}

	// 5. Get articles by IDs from vector search results
	var vectorArticleIDs []string
	uniqueIDs := make(map[string]bool)
	for _, result := range combinedVectorResults {
		articleID := h.server.extractArticleID(result.ID)
		if !uniqueIDs[articleID] {
			vectorArticleIDs = append(vectorArticleIDs, articleID)
			uniqueIDs[articleID] = true
		}
	}

	var vectorArticles []opensearch.Article
	if len(vectorArticleIDs) > 0 {
		vectorArticles, err = h.server.opensearchClient.GetArticlesByIDs(ctx, vectorArticleIDs)
		if err != nil {
			log.Printf("Failed to get articles by IDs: %v", err)
			vectorArticles = []opensearch.Article{}
		}
	}

	// 6. Combine and deduplicate results
	combinedResults := h.server.combineSearchResults(combinedVectorResults, vectorArticles, keywordResp.Results, size)

	// Send sources information
	sourcesData, _ := json.Marshal(combinedResults)
	sendSSEMessage(w, "sources", string(sourcesData))
	w.(http.Flusher).Flush()

	// 7. Extract articles for AI answer generation
	articles := make([]opensearch.Article, len(combinedResults))
	for i, result := range combinedResults {
		articles[i] = result.Article
	}

	sendSSEMessage(w, "status", "Generating AI answer...")
	w.(http.Flusher).Flush()

	// 8. Generate AI answer using search results with streaming
	err = h.server.generateAnswerStream(ctx, req.Query, articles, func(chunk string) error {
		sendSSEMessage(w, "answer", chunk)
		w.(http.Flusher).Flush()
		return nil
	})

	if err != nil {
		sendSSEMessage(w, "error", fmt.Sprintf("Failed to generate answer: %v", err))
		return
	}

	sendSSEMessage(w, "done", "")
	w.(http.Flusher).Flush()
}
