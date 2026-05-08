package rewards

import (
	"strings"
	"unicode"

	db "holo/internal/db/generated"
)

// synonyms maps normalized raw keywords (from card issuer descriptions) to
// fragment strings that might appear in app category names. Tier 2 matching.
var synonyms = map[string][]string{
	"dining":       {"restaurant", "food and drink", "dining"},
	"restaurants":  {"restaurant", "food and drink"},
	"restaurant":   {"restaurant", "food and drink"},
	"food":         {"restaurant", "food and drink", "grocer"},
	"eating":       {"restaurant", "food and drink"},
	"meals":        {"restaurant", "food and drink"},
	"cafe":         {"restaurant", "food and drink", "coffee"},
	"coffee":       {"coffee", "food and drink", "restaurant"},
	"bars":         {"restaurant", "food and drink", "alcohol"},
	"drinks":       {"restaurant", "food and drink", "alcohol"},
	"beverage":     {"food and drink"},
	"groceries":    {"grocer"},
	"grocery":      {"grocer"},
	"supermarket":  {"grocer"},
	"travel":       {"travel", "airline", "hotel"},
	"airline":      {"airline", "travel"},
	"airlines":     {"airline", "travel"},
	"flights":      {"airline", "travel"},
	"flight":       {"airline", "travel"},
	"hotel":        {"hotel", "travel", "lodging"},
	"hotels":       {"hotel", "travel", "lodging"},
	"lodging":      {"hotel", "travel", "lodging"},
	"accommodation": {"hotel", "travel"},
	"car rental":   {"car rental", "rental", "travel"},
	"rental car":   {"car rental", "rental", "travel"},
	"train":        {"travel", "transit", "rail"},
	"transit":      {"transit", "public transit", "transport"},
	"transportation": {"transit", "travel", "transport"},
	"gas":          {"gas", "fuel", "petrol"},
	"fuel":         {"gas", "fuel", "petrol"},
	"petrol":       {"gas", "fuel"},
	"parking":      {"parking"},
	"streaming":    {"streaming", "digital", "subscription"},
	"subscriptions": {"streaming", "digital", "subscription"},
	"subscription": {"streaming", "digital", "subscription"},
	"digital":      {"digital", "streaming", "subscription"},
	"entertainment": {"entertainment"},
	"shopping":     {"shopping", "retail", "merchandise"},
	"retail":       {"shopping", "retail"},
	"merchandise":  {"shopping", "retail"},
	"pharmacy":     {"pharmacy", "drug"},
	"drugstore":    {"pharmacy", "drug"},
	"drug":         {"pharmacy", "drug"},
	"bills":        {"utilities", "bill", "payment"},
	"utilities":    {"utilities", "bill"},
	"phone":        {"phone", "utilities", "bill", "wireless"},
	"wireless":     {"phone", "wireless", "utilities"},
	"internet":     {"internet", "utilities", "bill"},
	"insurance":    {"insurance"},
	"recurring":    {"bill", "subscription"},
	"preauthorized": {"bill", "subscription"},
}

func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prev := ' '
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prev = r
		} else if prev != ' ' {
			b.WriteRune(' ')
			prev = ' '
		}
	}
	return strings.TrimSpace(b.String())
}

// MatchCategory finds the best matching category from the app's category list
// for a raw string (typically from a card issuer's rate table).
//
// Tier 1: normalized substring match (raw contains category name or vice versa).
// Tier 2: keyword synonym lookup.
// Returns nil if no match found (caller stores as catch-all or unmatched).
func MatchCategory(raw string, categories []db.Category) *string {
	if raw == "" {
		return nil
	}

	// Only match against parent categories. Card rates are configured at the
	// parent level (cat_food, cat_travel, etc.) — sub-categories (Plaid) are
	// too granular and would break the reward rate lookup subquery.
	var parents []db.Category
	for _, c := range categories {
		if c.ParentID == nil {
			parents = append(parents, c)
		}
	}

	normRaw := normalize(raw)
	rawWords := strings.Fields(normRaw)

	var bestT1 *string
	bestT1Len := 0
	for _, cat := range parents {
		normCat := normalize(cat.Name)
		if normCat == "" {
			continue
		}
		if strings.Contains(normRaw, normCat) || strings.Contains(normCat, normRaw) {
			if len(normCat) > bestT1Len {
				id := cat.ID
				bestT1 = &id
				bestT1Len = len(normCat)
			}
		}
	}
	if bestT1 != nil {
		return bestT1
	}

	candidateFrags := map[string]struct{}{}
	for _, word := range rawWords {
		if frags, ok := synonyms[word]; ok {
			for _, f := range frags {
				candidateFrags[f] = struct{}{}
			}
		}
	}
	for i := 0; i < len(rawWords); i++ {
		for j := i + 2; j <= len(rawWords) && j <= i+3; j++ {
			phrase := strings.Join(rawWords[i:j], " ")
			if frags, ok := synonyms[phrase]; ok {
				for _, f := range frags {
					candidateFrags[f] = struct{}{}
				}
			}
		}
	}

	var bestT2 *string
	bestT2Len := 0
	for _, cat := range parents {
		normCat := normalize(cat.Name)
		for frag := range candidateFrags {
			if strings.Contains(normCat, frag) {
				if len(normCat) > bestT2Len {
					id := cat.ID
					bestT2 = &id
					bestT2Len = len(normCat)
				}
			}
		}
	}
	return bestT2
}
