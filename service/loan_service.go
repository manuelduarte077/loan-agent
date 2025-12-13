package service

import (
	"errors"
	"math"

	"loan-agent/domain"
	"loan-agent/repository"
)

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
	if input.InterestRate < 0 {
		return domain.LoanResult{}, errors.New("tasa inválida")
	}
	if input.TermMonths <= 0 {
		return domain.LoanResult{}, errors.New("plazo inválido")
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
		MonthlyPayment: cuota,
		TotalPayment:   total,
		TotalInterest:  intereses,
	}

	// Guardar el resultado
	_ = s.repo.Save(input, result)

	return result, nil
}
