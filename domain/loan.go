package domain

type LoanInput struct {
	Amount       float64
	InterestRate float64
	TermMonths   int
}

type LoanResult struct {
	MonthlyPayment float64
	TotalPayment   float64
	TotalInterest  float64
}
