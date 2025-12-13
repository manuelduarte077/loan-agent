package http

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"loan-agent/domain"
	"loan-agent/service"
)

type LoanHandler struct {
	service *service.LoanService
}

func NewLoanHandler(service *service.LoanService) *LoanHandler {
	return &LoanHandler{service: service}
}

func (h *LoanHandler) CalculateLoan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validar Content-Type
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	var input domain.LoanInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.service.CalculateLoan(input)
	if err != nil {
		log.Printf("Error calculating loan: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}
