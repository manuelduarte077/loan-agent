package main

import (
	"fmt"

	"loan-agent/domain"
	"loan-agent/repository"
	"loan-agent/service"
)

func main() {

	repo := repository.NewLoanRepositoryMemory()
	loanService := service.NewLoanService(repo)

	input := domain.LoanInput{
		Amount:       10000,
		InterestRate: 12,
		TermMonths:   24,
	}

	result, err := loanService.CalculateLoan(input)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Cuota mensual: %.2f\n", result.MonthlyPayment)
	fmt.Printf("Total a pagar: %.2f\n", result.TotalPayment)
	fmt.Printf("Intereses: %.2f\n", result.TotalInterest)
}
