package service

import (
	"fmt"
	"os"
)

const (
	MaxLoanAmount        = 1_000_000_000.0 // 1 billón
	MaxInterestRate      = 1000.0          // 1000% anual
	MaxTermMonths        = 600             // 50 años
	MinTermMonths        = 1
	MaxDebtAmount        = 100_000_000.0 // 100 millones
	MaxDebtsPerRequest   = 50            // máximo de deudas por request
	MaxDebtPayoffMonths  = 600           // 50 años máximo para pagar deudas
	DebtBalanceTolerance = 0.01          // tolerancia para considerar deuda pagada

	MaxTermRangeMonths = 120 // máximo rango de términos a evaluar (10 años)
)

// GetUSDToNIORate gets the exchange rate from USD to NIO
// Can be configured via USD_TO_NIO_RATE environment variable
// Default is 36.5 NIO per USD
func GetUSDToNIORate() float64 {
	if envRate := os.Getenv("USD_TO_NIO_RATE"); envRate != "" {
		if parsedRate := parseFloat(envRate); parsedRate > 0 {
			return parsedRate
		}
	}

	return 36.5
}

func parseFloat(s string) float64 {
	var result float64
	_, err := fmt.Sscanf(s, "%f", &result)
	if err != nil {
		return 0
	}
	return result
}
