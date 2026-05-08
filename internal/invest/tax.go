package invest

// bracket represents a single tax bracket: [low, high) income range and marginal rate.
type bracket struct {
	low  float64
	high float64
	rate float64
}

// federalBrackets2025 are the federal marginal rates for 2025.
var federalBrackets2025 = []bracket{
	{0, 57375, 0.15},
	{57375, 114750, 0.205},
	{114750, 158519, 0.26},
	{158519, 220000, 0.29},
	{220000, 1e18, 0.33},
}

// marginalRate returns the rate from a bracket table for the given income.
func marginalRate(income float64, brackets []bracket) float64 {
	for _, b := range brackets {
		if income >= b.low && income < b.high {
			return b.rate
		}
	}
	return 0
}

// provincialBrackets2025 maps province/territory code → bracket slices (provincial-only rates).
// Sources: approximate 2025 provincial rates, suitable for planning purposes.
var provincialBrackets2025 = map[string][]bracket{
	// Alberta: flat 10%
	"AB": {
		{0, 1e18, 0.10},
	},
	// British Columbia
	"BC": {
		{0, 45654, 0.0506},
		{45654, 91310, 0.077},
		{91310, 104835, 0.105},
		{104835, 127299, 0.1229},
		{127299, 172602, 0.147},
		{172602, 240716, 0.168},
		{240716, 1e18, 0.205},
	},
	// Manitoba
	"MB": {
		{0, 47000, 0.108},
		{47000, 100000, 0.1275},
		{100000, 1e18, 0.174},
	},
	// New Brunswick
	"NB": {
		{0, 47715, 0.094},
		{47715, 95431, 0.14},
		{95431, 176756, 0.16},
		{176756, 1e18, 0.195},
	},
	// Newfoundland and Labrador
	"NL": {
		{0, 43198, 0.087},
		{43198, 86395, 0.145},
		{86395, 154244, 0.158},
		{154244, 215943, 0.178},
		{215943, 275870, 0.198},
		{275870, 551739, 0.208},
		{551739, 1e18, 0.213},
	},
	// Nova Scotia
	"NS": {
		{0, 29590, 0.0879},
		{29590, 59180, 0.1495},
		{59180, 93000, 0.1667},
		{93000, 150000, 0.175},
		{150000, 1e18, 0.21},
	},
	// Northwest Territories
	"NT": {
		{0, 50597, 0.059},
		{50597, 101198, 0.086},
		{101198, 164525, 0.122},
		{164525, 1e18, 0.1405},
	},
	// Nunavut
	"NU": {
		{0, 53268, 0.04},
		{53268, 106537, 0.07},
		{106537, 173205, 0.09},
		{173205, 1e18, 0.115},
	},
	// Ontario
	"ON": {
		{0, 51446, 0.0505},
		{51446, 102894, 0.0915},
		{102894, 150000, 0.1116},
		{150000, 220000, 0.1216},
		{220000, 1e18, 0.1316},
	},
	// Prince Edward Island
	"PE": {
		{0, 32656, 0.096},
		{32656, 64313, 0.1337},
		{64313, 105000, 0.167},
		{105000, 140000, 0.18},
		{140000, 1e18, 0.1875},
	},
	// Quebec (highest provincial rates)
	"QC": {
		{0, 51780, 0.14},
		{51780, 103545, 0.19},
		{103545, 126000, 0.24},
		{126000, 1e18, 0.2575},
	},
	// Saskatchewan
	"SK": {
		{0, 49720, 0.105},
		{49720, 142058, 0.125},
		{142058, 1e18, 0.145},
	},
	// Yukon
	"YT": {
		{0, 57375, 0.064},
		{57375, 114750, 0.09},
		{114750, 158519, 0.109},
		{158519, 500000, 0.128},
		{500000, 1e18, 0.15},
	},
}

// MarginalRate returns the combined federal+provincial marginal tax rate
// for the given annual income and province (2025 tax year).
// Brackets are approximate — suitable for planning recommendations.
func MarginalRate(annualIncome float64, province string) float64 {
	if annualIncome <= 0 {
		return 0
	}

	federal := marginalRate(annualIncome, federalBrackets2025)

	provBrackets, ok := provincialBrackets2025[province]
	if !ok {
		// Default to Ontario if province unknown
		provBrackets = provincialBrackets2025["ON"]
	}
	provincial := marginalRate(annualIncome, provBrackets)

	return federal + provincial
}
