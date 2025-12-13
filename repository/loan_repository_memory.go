package repository

import "loan-agent/domain"

// LoanRepositoryMemory is an in-memory implementation of LoanRepository.
type LoanRepositoryMemory struct {
	data []domain.LoanResult
}

// NewLoanRepositoryMemory creates a new in-memory loan repository.
func NewLoanRepositoryMemory() *LoanRepositoryMemory {
	return &LoanRepositoryMemory{
		data: []domain.LoanResult{},
	}
}

// Save stores the loan result in memory.
func (r *LoanRepositoryMemory) Save(
	input domain.LoanInput,
	result domain.LoanResult,
) error {
	r.data = append(r.data, result)
	return nil
}
