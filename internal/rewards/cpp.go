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
	"strings"
)

// CPPResult is the result of a CPP lookup.
type CPPResult struct {
	CPP    float64
	Source string // "milesopedia" | "openrouter" | "hardcoded"
}

// cppDefaults is the hardcoded fallback. Fixed programs have a definitive value.
// Variable programs use a midpoint estimate from Milesopedia/PoT.
var cppDefaults = map[string]float64{
	"scene+":                  1.0,
	"td rewards":              0.5,
	"bmo rewards":             0.67,
	"westjet rewards":         1.0,
	"amex membership rewards": 2.0,
	"aeroplan":                2.0,
	"rbc avion rewards":       1.7,
	"cibc aventura":           1.2,
	"air miles":               10.5,
}

// FetchCPP fetches the CPP for a Canadian rewards program.
// Tier 1: Jina fetch of Milesopedia CPP table → OpenRouter parse.
// Tier 2: Hardcoded fallback.
func FetchCPP(ctx context.Context, apiKey, model, program string) (CPPResult, error) {
	// Tier 1: Milesopedia via Jina
	milesURL := "https://www.milesopedia.com/en/points-miles-value-canada/"
	content, err := jinaFetch(ctx, milesURL)
	if err == nil && len(content) > 500 {
		cpp, err := parseCPPFromContent(ctx, apiKey, model, program, content)
		if err == nil && cpp > 0 {
			log.Printf("cpp: %s — milesopedia %.2f¢/pt", program, cpp)
			return CPPResult{CPP: cpp, Source: "milesopedia"}, nil
		}
		log.Printf("cpp: %s — milesopedia parse failed: %v", program, err)
	}

	// Tier 2: Hardcoded fallback
	norm := strings.ToLower(strings.TrimSpace(program))
	if v, ok := cppDefaults[norm]; ok {
		return CPPResult{CPP: v, Source: "hardcoded"}, nil
	}

	// Unknown program, use 1.0 as safe default
	return CPPResult{CPP: 1.0, Source: "hardcoded"}, nil
}

func jinaFetch(ctx context.Context, targetURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://r.jina.ai/"+url.PathEscape(targetURL), nil)
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
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("jina %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
	return string(body), err
}

func parseCPPFromContent(ctx context.Context, apiKey, model, program, content string) (float64, error) {
	prompt := fmt.Sprintf(`From the following Milesopedia page content, find the cents-per-point value for the "%s" rewards program in Canada.
Return JSON: {"cpp": 1.75}
If not found, return {"cpp": 0}.

Content:
%s`, program, content[:minInt(len(content), 10000)])

	reqBody := map[string]any{
		"model":    model,
		"messages": []map[string]string{{"role": "user", "content": prompt}},
		"response_format": map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "cpp_result",
				"strict": true,
				"schema": map[string]any{
					"type": "object",
					"properties": map[string]any{"cpp": map[string]any{"type": "number"}},
					"required":             []string{"cpp"},
					"additionalProperties": false,
				},
			},
		},
		"temperature": 0.1,
	}
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://holo")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var orResp struct {
		Choices []struct {
			Message struct{ Content string `json:"content"` } `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &orResp); err != nil || len(orResp.Choices) == 0 {
		return 0, fmt.Errorf("parse response: %w", err)
	}
	var result struct{ CPP float64 `json:"cpp"` }
	if err := json.Unmarshal([]byte(orResp.Choices[0].Message.Content), &result); err != nil {
		return 0, err
	}
	return result.CPP, nil
}
