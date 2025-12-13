package domain

type TermRecommendationInput struct {
	Amount            float64
	InterestRate      float64
	MinTermMonths     int
	MaxTermMonths     int
	MaxMonthlyPayment float64
	Preference        string // "minimize_interest", "minimize_payment", "balanced"
}

type TermRecommendation struct {
	TermMonths     int
	MonthlyPayment float64
	TotalInterest  float64
	Score          float64
	Reason         string
}

type TermRecommendationResult struct {
	RecommendedTerm int
	Recommendations []TermRecommendation
}
