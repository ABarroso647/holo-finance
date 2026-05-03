package rewards

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
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

// FetchRates fetches earn rates for a Canadian credit card.
// Jina Search pulls live page content; OpenRouter/DeepSeek parses the JSON.
func FetchRates(ctx context.Context, apiKey, model, cardName string) ([]FetchedRate, error) {
	pageContent, _ := jinaSearch(ctx, cardName+" credit card Canada reward earn rates site:creditcardgenius.ca OR site:milesopedia.com")

	var prompt string
	if pageContent != "" {
		prompt = fmt.Sprintf(`Extract the reward earn rates for the "%s" credit card in Canada from the following web content.

%s

Return ONLY a JSON array (no markdown, no explanation) where each element has:
- "category": string (e.g. "dining & restaurants", "groceries", "travel", "everything else")
- "rate": number (cashback %%: 4.0 = 4%%; points multiplier: 5.0 = 5x)
- "rate_type": "cashback" or "points"
- "cap_amount": number or null (annual CAD cap if any)
- "cap_period": "monthly" or "annual" or null
- "notes": string (empty if none)

Include an "everything else" catch-all entry. Return ONLY the JSON array.`,
			cardName, pageContent[:minInt(len(pageContent), 8000)])
	} else {
		prompt = fmt.Sprintf(`What are the current reward earn rates for the "%s" credit card in Canada?

Return ONLY a JSON array (no markdown, no explanation) where each element has:
- "category": string (e.g. "dining & restaurants", "groceries", "travel", "everything else")
- "rate": number (cashback %%: 4.0 = 4%%; points multiplier: 5.0 = 5x)
- "rate_type": "cashback" or "points"
- "cap_amount": number or null (annual CAD cap if any)
- "cap_period": "monthly" or "annual" or null
- "notes": string (empty if none)

Include an "everything else" catch-all entry. Return ONLY the JSON array.`, cardName)
	}

	reqBody := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"response_format": map[string]string{"type": "json_object"},
		"temperature":     0.1,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/abarroso647/holo")

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

	return parseRatesJSON(orResp.Choices[0].Message.Content)
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

var jsonArrayRe = regexp.MustCompile(`(?s)\[.*\]`)

func parseRatesJSON(text string) ([]FetchedRate, error) {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	// Some models wrap the array in an object like {"rates": [...]}
	if strings.HasPrefix(text, "{") {
		var wrapper map[string]json.RawMessage
		if json.Unmarshal([]byte(text), &wrapper) == nil {
			for _, v := range wrapper {
				if strings.HasPrefix(strings.TrimSpace(string(v)), "[") {
					text = strings.TrimSpace(string(v))
					break
				}
			}
		}
	}

	if !strings.HasPrefix(text, "[") {
		if m := jsonArrayRe.FindString(text); m != "" {
			text = m
		}
	}

	var rates []FetchedRate
	if err := json.Unmarshal([]byte(text), &rates); err != nil {
		return nil, fmt.Errorf("parse rates JSON: %w — raw: %.200s", err, text)
	}
	return rates, nil
}
