package service

import (
	"errors"
	"loan-agent/domain"
	"testing"
)

type MockLoanRepository struct {
	SaveCalled bool
	ForceError bool
}

func (m *MockLoanRepository) Save(
	input domain.LoanInput,
	result domain.LoanResult,
) error {
	m.SaveCalled = true
	if m.ForceError {
		return errors.New("save error")
	}
	return nil
}

func TestCalculateLoan_WithInterest(t *testing.T) {

	mockRepo := &MockLoanRepository{}
	service := NewLoanService(mockRepo)

	input := domain.LoanInput{
		Amount:       10000,
		InterestRate: 12,
		TermMonths:   24,
	}

	result, err := service.CalculateLoan(input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.MonthlyPayment <= 0 {
		t.Errorf("expected cuota > 0")
	}

	if !mockRepo.SaveCalled {
		t.Errorf("expected repository Save to be called")
	}
}

func TestCalculateLoan_ZeroInterest(t *testing.T) {

	mockRepo := &MockLoanRepository{}
	service := NewLoanService(mockRepo)

	input := domain.LoanInput{
		Amount:       1200,
		InterestRate: 0,
		TermMonths:   12,
	}

	result, err := service.CalculateLoan(input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := 100.0
	if result.MonthlyPayment != expected {
		t.Errorf("expected %.2f, got %.2f", expected, result.MonthlyPayment)
	}
}

func TestCalculateLoan_InvalidAmount(t *testing.T) {

	mockRepo := &MockLoanRepository{}
	service := NewLoanService(mockRepo)

	input := domain.LoanInput{
		Amount:       0,
		InterestRate: 10,
		TermMonths:   12,
	}

	_, err := service.CalculateLoan(input)

	if err == nil {
		t.Errorf("expected error for invalid amount")
	}

	if mockRepo.SaveCalled {
		t.Errorf("repository Save should NOT be called")
	}
}

func TestCalculateLoan_InvalidTerm(t *testing.T) {

	mockRepo := &MockLoanRepository{}
	service := NewLoanService(mockRepo)

	input := domain.LoanInput{
		Amount:       1000,
		InterestRate: 10,
		TermMonths:   0,
	}

	_, err := service.CalculateLoan(input)

	if err == nil {
		t.Errorf("expected error for invalid term")
	}
}
