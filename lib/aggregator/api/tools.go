package api

import (
	"encoding/json"

	"github.com/snowmerak/open-librarian/lib/client/llm"
)

const (
	ToolVectorSearch  = "vector_search"
	ToolKeywordSearch = "keyword_search"
)

func GetSearchTools() []llm.Tool {
	return []llm.Tool{
		{
			Type: "function",
			Function: llm.FunctionDent{
				Name:        ToolVectorSearch,
				Description: "Perform a semantic vector search using embedding. Useful for finding documents based on meaning, concept, or when exact keywords are not known. Returns relevant document summaries.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"query": {
							"type": "string",
							"description": "The semantic search query sentence. Should be a full sentence or descriptive phrase in the target language."
						}
					},
					"required": ["query"]
				}`),
			},
		},
		{
			Type: "function",
			Function: llm.FunctionDent{
				Name:        ToolKeywordSearch,
				Description: "Perform a keyword-based search. Useful for finding documents containing specific proper nouns, technical terms, or exact phrases. Returns relevant document summaries.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"keywords": {
							"type": "string",
							"description": "Space-separated keywords for search."
						}
					},
					"required": ["keywords"]
				}`),
			},
		},
	}
}
