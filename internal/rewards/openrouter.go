package rewards

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

// FetchedRate is a single reward rate entry parsed from the card's page.
type FetchedRate struct {
	Category  string   `json:"category"`
	Rate      float64  `json:"rate"`
	RateType  string   `json:"rate_type"`
	CapAmount *float64 `json:"cap_amount"`
	CapPeriod *string  `json:"cap_period"`
	Notes     string   `json:"notes"`
}

var ratesSchema = map[string]any{
	"type": "json_schema",
	"json_schema": map[string]any{
		"name":   "card_rates",
		"strict": true,
		"schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"rates": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"category":  map[string]any{"type": "string"},
							"rate":      map[string]any{"type": "number"},
							"rate_type": map[string]any{"type": "string", "enum": []string{"cashback", "points"}},
							"cap_amount": map[string]any{"anyOf": []map[string]string{
								{"type": "number"},
								{"type": "null"},
							}},
							"cap_period": map[string]any{"anyOf": []map[string]string{
								{"type": "string"},
								{"type": "null"},
							}},
							"notes": map[string]any{"type": "string"},
						},
						"required":             []string{"category", "rate", "rate_type", "cap_amount", "cap_period", "notes"},
						"additionalProperties": false,
					},
				},
			},
			"required":             []string{"rates"},
			"additionalProperties": false,
		},
	},
}

// FetchRates fetches earn rates for a Canadian credit card.
// Jina Search pulls live page content; OpenRouter/DeepSeek parses the JSON.
func FetchRates(ctx context.Context, apiKey, model, cardName string) ([]FetchedRate, error) {
	pageContent, err := jinaSearch(ctx, cardName+" credit card Canada earn rates site:creditcardgenius.ca")
	if err != nil || len(pageContent) < 500 {
		// Fallback: broader search
		pageContent, _ = jinaSearch(ctx, cardName+" credit card Canada reward earn rates")
	}
	log.Printf("fetch-rates: %s — jina returned %d chars", cardName, len(pageContent))

	var prompt string
	if pageContent != "" {
		prompt = fmt.Sprintf(`Extract the reward earn rates for the "%s" credit card in Canada from the following web content.

%s

Return a JSON object with a "rates" array. Each element must have:
- "category": string (e.g. "dining & restaurants", "groceries", "travel", "everything else")
- "rate": number (cashback %%: 4.0 = 4%%; points multiplier: 5.0 = 5x)
- "rate_type": "cashback" or "points"
- "cap_amount": number or null (CAD cap if any)
- "cap_period": "monthly" or "annual" or null
- "notes": string (empty string if none)

Include an "everything else" catch-all entry.`,
			cardName, pageContent[:minInt(len(pageContent), 12000)])
	} else {
		prompt = fmt.Sprintf(`What are the current reward earn rates for the "%s" credit card in Canada?

Return a JSON object with a "rates" array. Each element must have:
- "category": string (e.g. "dining & restaurants", "groceries", "travel", "everything else")
- "rate": number (cashback %%: 4.0 = 4%%; points multiplier: 5.0 = 5x)
- "rate_type": "cashback" or "points"
- "cap_amount": number or null (CAD cap if any)
- "cap_period": "monthly" or "annual" or null
- "notes": string (empty string if none)

Include an "everything else" catch-all entry.`, cardName)
	}

	reqBody := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"response_format": ratesSchema,
		"temperature":     0.1,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://holo")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openrouter request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openrouter error %d: %s", resp.StatusCode, string(raw))
	}

	var orResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &orResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if len(orResp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from OpenRouter")
	}

	var wrapper struct {
		Rates []FetchedRate `json:"rates"`
	}
	if err := json.Unmarshal([]byte(orResp.Choices[0].Message.Content), &wrapper); err != nil {
		return nil, fmt.Errorf("parse rates: %w — raw: %.200s", err, orResp.Choices[0].Message.Content)
	}
	log.Printf("fetch-rates: %s — OpenRouter returned %d rates", cardName, len(wrapper.Rates))
	return wrapper.Rates, nil
}

// jinaSearch fetches search results via Jina AI's free search API (no key needed).
func jinaSearch(ctx context.Context, query string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://s.jina.ai/"+url.PathEscape(query), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/plain")
	req.Header.Set("X-Retain-Images", "none")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("jina %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 12*1024))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
