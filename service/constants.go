package service

const (
	MaxLoanAmount        = 1_000_000_000.0 // 1 billón
	MaxInterestRate      = 1000.0          // 1000% anual
	MaxTermMonths        = 600             // 50 años
	MinTermMonths        = 1
	MaxDebtAmount        = 100_000_000.0 // 100 millones
	MaxDebtsPerRequest   = 50            // máximo de deudas por request
	MaxDebtPayoffMonths  = 600           // 50 años máximo para pagar deudas
	DebtBalanceTolerance = 0.01          // tolerancia para considerar deuda pagada

	// Límites de términos para recomendación
	MaxTermRangeMonths = 120 // máximo rango de términos a evaluar (10 años)
)
