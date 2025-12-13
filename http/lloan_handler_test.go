package http

import (
	"bytes"
	"loan-agent/repository"
	"loan-agent/service"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCalculateLoanHandler_OK(t *testing.T) {

	repo := repository.NewLoanRepositoryMemory()
	service := service.NewLoanService(repo)
	handler := NewLoanHandler(service)

	body := []byte(`{
		"monto": 10000,
		"tasa_anual": 12,
		"plazo_meses": 24
	}`)

	req := httptest.NewRequest(
		http.MethodPost,
		"/loan/calculate",
		bytes.NewBuffer(body),
	)

	w := httptest.NewRecorder()

	handler.CalculateLoan(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestCalculateLoanHandler_MethodNotAllowed(t *testing.T) {

	repo := repository.NewLoanRepositoryMemory()
	service := service.NewLoanService(repo)
	handler := NewLoanHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/loan/calculate", nil)
	w := httptest.NewRecorder()

	handler.CalculateLoan(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestCalculateLoanHandler_BadRequest(t *testing.T) {

	repo := repository.NewLoanRepositoryMemory()
	service := service.NewLoanService(repo)
	handler := NewLoanHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/loan/calculate",
		bytes.NewBuffer([]byte(`{invalid-json}`)),
	)

	w := httptest.NewRecorder()
	handler.CalculateLoan(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
