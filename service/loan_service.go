package service

import (
	"errors"
	"fmt"
	"log"
	"math"

	"loan-agent/domain"
	"loan-agent/repository"
)

// roundTo2Decimals redondea un float64 a 2 decimales
func roundTo2Decimals(value float64) float64 {
	return math.Round(value*100) / 100
}

type LoanService struct {
	repo  repository.LoanRepository
	cache repository.CacheRepository
}

// NewLoanService creates a new LoanService with the given repository.
func NewLoanService(repo repository.LoanRepository,
	cache repository.CacheRepository,
) *LoanService {
	return &LoanService{repo: repo, cache: cache}
}

// CalculateLoan calculates the loan details based on the input parameters.
func (s *LoanService) CalculateLoan(
	input domain.LoanInput,
) (domain.LoanResult, error) {

	// Validar entrada
	if input.Amount <= 0 {
		return domain.LoanResult{}, errors.New("monto inválido")
	}
	if input.Amount > MaxLoanAmount {
		return domain.LoanResult{}, fmt.Errorf("monto excede el máximo permitido de $%.2f", MaxLoanAmount)
	}
	if input.InterestRate < 0 {
		return domain.LoanResult{}, errors.New("tasa inválida")
	}
	if input.InterestRate > MaxInterestRate {
		return domain.LoanResult{}, fmt.Errorf("tasa de interés excede el máximo permitido de %.2f%%", MaxInterestRate)
	}
	if input.TermMonths <= 0 {
		return domain.LoanResult{}, errors.New("plazo inválido")
	}
	if input.TermMonths > MaxTermMonths {
		return domain.LoanResult{}, fmt.Errorf("plazo excede el máximo permitido de %d meses", MaxTermMonths)
	}

	var cuota float64

	if input.InterestRate == 0 {
		cuota = input.Amount / float64(input.TermMonths)
	} else {
		tasaMensual := (input.InterestRate / 100) / 12
		n := float64(input.TermMonths)

		cuota = input.Amount * (tasaMensual /
			(1 - math.Pow(1+tasaMensual, -n)))
	}

	total := cuota * float64(input.TermMonths)
	intereses := total - input.Amount

	result := domain.LoanResult{
		MonthlyPayment: roundTo2Decimals(cuota),
		TotalPayment:   roundTo2Decimals(total),
		TotalInterest:  roundTo2Decimals(intereses),
	}

	// Guardar el resultado (no crítico si falla)
	if err := s.repo.Save(input, result); err != nil {
		log.Printf("Warning: failed to save loan calculation: %v", err)
	}

	return result, nil
}
