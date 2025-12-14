package domain

type Debt struct {
	Name           string
	Amount         float64
	InterestRate   float64
	MinimumPayment float64
}

type DebtExitInput struct {
	Debts                   []Debt
	AvailableMonthlyPayment float64
	Strategy                string // "snowball", "avalanche", "compare"
}

type MonthlyPayment struct {
	DebtName         string
	Payment          float64
	RemainingBalance float64
}

type MonthlyPlan struct {
	Month     int
	Payments  []MonthlyPayment
	TotalPaid float64
}

type StrategyResult struct {
	TotalInterestPaid float64
	MonthsToPayoff    int
}

type Comparison struct {
	Snowball  StrategyResult
	Avalanche StrategyResult
	Savings   struct {
		InterestSaved float64
		MonthsSaved   int
	}
}

type DebtExitResult struct {
	Strategy          string
	TotalDebt         float64
	TotalInterestPaid float64
	MonthsToPayoff    int
	MonthlyPlan       []MonthlyPlan
	Comparison        *Comparison `json:",omitempty"`
	Explanation       string      `json:",omitempty"` // Explicación generada por IA
}
