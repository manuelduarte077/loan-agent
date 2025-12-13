package repository

import "loan-agent/domain"

type LoanRepository interface {
	Save(input domain.LoanInput, result domain.LoanResult) error
}
