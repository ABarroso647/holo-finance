package categorize

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"holo/internal/db/generated"
)

const batchSize = 80

type openRouterRequest struct {
	Model          string      `json:"model"`
	Messages       []message   `json:"messages"`
	ResponseFormat respFormat  `json:"response_format"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type respFormat struct {
	Type string `json:"type"`
}

type llmResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type categorizeResult struct {
	Results []struct {
		ID       string `json:"id"`
		Merchant string `json:"merchant"`
		Category string `json:"category"`
	} `json:"results"`
}

// Only merchant names are sent — no amounts, dates, or account info.
func RunLLMCategorization(ctx context.Context, queries *db.Queries) (int, error) {
	txns, err := queries.ListUncategorizedTransactions(ctx)
	if err != nil {
		return 0, err
	}
	if len(txns) == 0 {
		return 0, nil
	}

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return 0, fmt.Errorf("OPENROUTER_API_KEY not set")
	}
	model, _ := queries.GetSetting(ctx, "openrouter_model")
	if model == "" {
		model = os.Getenv("OPENROUTER_MODEL")
	}
	if model == "" {
		model = "deepseek/deepseek-v4-flash"
	}

	allCategories, err := queries.ListCategories(ctx)
	if err != nil {
		return 0, err
	}
	// Pass only top-level parent categories to the LLM — cleaner list, better accuracy.
	var parentCategories []db.Category
	for _, c := range allCategories {
		if c.ParentID == nil {
			parentCategories = append(parentCategories, c)
		}
	}
	categoryNames := make([]string, len(parentCategories))
	catIDMap := make(map[string]string, len(parentCategories))
	for i, c := range parentCategories {
		categoryNames[i] = c.Name
		catIDMap[strings.ToLower(c.Name)] = c.ID
	}
	categoryList := strings.Join(categoryNames, ", ")

	total := 0
	for i := 0; i < len(txns); i += batchSize {
		end := i + batchSize
		if end > len(txns) {
			end = len(txns)
		}
		batch := txns[i:end]

		n, err := processBatch(ctx, queries, batch, categoryList, catIDMap, model, apiKey)
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func processBatch(ctx context.Context, queries *db.Queries, batch []db.Transaction, categoryList string, catIDMap map[string]string, model, apiKey string) (int, error) {
	type merchantEntry struct {
		ID       string `json:"id"`
		Merchant string `json:"merchant"`
	}
	entries := make([]merchantEntry, 0, len(batch))
	for _, txn := range batch {
		name := txn.Name
		if txn.MerchantName != nil && *txn.MerchantName != "" {
			name = *txn.MerchantName
		}
		entries = append(entries, merchantEntry{ID: txn.ID, Merchant: name})
	}

	entriesJSON, _ := json.Marshal(entries)

	systemPrompt := fmt.Sprintf(`You are a transaction categorizer. Categorize each merchant into exactly one of these categories: %s.
Return JSON in this exact format: {"results": [{"id": "...", "merchant": "...", "category": "..."}]}
Use the exact category name from the list. If unsure, use "Other".`, categoryList)

	req := openRouterRequest{
		Model: model,
		Messages: []message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: string(entriesJSON)},
		},
		ResponseFormat: respFormat{Type: "json_object"},
	}

	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("HTTP-Referer", "https://holo")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("openrouter request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("openrouter %d: %s", resp.StatusCode, string(body))
	}

	var llmResp llmResponse
	if err := json.NewDecoder(resp.Body).Decode(&llmResp); err != nil {
		return 0, fmt.Errorf("decode llm response: %w", err)
	}
	if len(llmResp.Choices) == 0 {
		return 0, fmt.Errorf("no choices in llm response")
	}

	var result categorizeResult
	if err := json.Unmarshal([]byte(llmResp.Choices[0].Message.Content), &result); err != nil {
		return 0, fmt.Errorf("parse llm json: %w", err)
	}

	count := 0
	for _, r := range result.Results {
		catID, ok := catIDMap[strings.ToLower(r.Category)]
		if !ok {
			catID = "cat_other"
		}
		if err := queries.UpdateTransactionCategory(ctx, db.UpdateTransactionCategoryParams{
			CategoryID:     &catID,
			CategorySource: "llm",
			ID:             r.ID,
		}); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
