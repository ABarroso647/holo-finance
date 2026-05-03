package rewards

import (
	"context"
	"fmt"
	"log"
	"os"

	db "github.com/abarroso647/holo/internal/db/generated"
	"github.com/google/uuid"
)


// FetchAndStoreRates fetches reward rates for a credit card via Jina + OpenRouter/DeepSeek,
// runs tier 1/2 category matching, and stores to DB.
// cardName is the human-readable name used as the search query (e.g. "American Express Cobalt Canada").
func FetchAndStoreRates(ctx context.Context, queries *db.Queries, accountID, cardName string) error {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		return fmt.Errorf("OPENROUTER_API_KEY not set")
	}

	model, _ := queries.GetSetting(ctx, "openrouter_model")
	if model == "" {
		model = os.Getenv("OPENROUTER_MODEL")
	}
	if model == "" {
		model = "deepseek/deepseek-v4-flash"
	}

	rates, err := FetchRates(ctx, key, model, cardName)
	if err != nil {
		return fmt.Errorf("fetch rates: %w", err)
	}
	if len(rates) == 0 {
		return fmt.Errorf("no rates returned for %q", cardName)
	}

	categories, err := queries.ListCategories(ctx)
	if err != nil {
		return fmt.Errorf("list categories: %w", err)
	}

	if err := queries.DeleteCardRewardRatesForAccount(ctx, accountID); err != nil {
		return fmt.Errorf("clear existing rates: %w", err)
	}

	for _, r := range rates {
		catID := MatchCategory(r.Category, categories)

		rawCat := r.Category
		params := db.UpsertCardRewardRateParams{
			ID:          uuid.New().String(),
			AccountID:   accountID,
			CategoryID:  catID,
			RawCategory: &rawCat,
			RewardRate:  r.Rate,
			CapAmount:   r.CapAmount,
			CapPeriod:   r.CapPeriod,
		}
		if r.Notes != "" {
			params.Notes = &r.Notes
		}

		if _, err := queries.UpsertCardRewardRate(ctx, params); err != nil {
			log.Printf("rewards: store rate for %q account %s: %v", r.Category, accountID, err)
		}
	}

	return nil
}

// RematchRates re-runs tier 1/2 category matching on all stored rates
// that currently have no category_id. Useful after new categories arrive.
func RematchRates(ctx context.Context, queries *db.Queries) (int, error) {
	unmatched, err := queries.ListAllCardRewardRatesWithNullCategory(ctx)
	if err != nil {
		return 0, err
	}
	if len(unmatched) == 0 {
		return 0, nil
	}

	categories, err := queries.ListCategories(ctx)
	if err != nil {
		return 0, err
	}

	matched := 0
	for _, r := range unmatched {
		if r.RawCategory == nil {
			continue
		}
		catID := MatchCategory(*r.RawCategory, categories)
		if catID == nil {
			continue
		}
		if err := queries.UpdateCardRewardRateCategoryID(ctx, db.UpdateCardRewardRateCategoryIDParams{
			CategoryID: catID,
			ID:         r.ID,
		}); err != nil {
			log.Printf("rewards: rematch update %s: %v", r.ID, err)
			continue
		}
		matched++
	}
	return matched, nil
}
