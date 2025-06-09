package api

import (
	"context"
	"fmt"
	"log"

	"github.com/snowmerak/open-librarian/lib/client/opensearch"
)

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
		useContent := len(article.Content) < 12000
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

	log.Printf("Answer generation: Using %d articles (content: %d, summary: %d)",
		len(articles), contentUsageCount, summaryUsageCount)

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

	log.Printf("Answer generation (streaming): Using %d articles (content: %d, summary: %d)",
		len(articles), contentUsageCount, summaryUsageCount)

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
