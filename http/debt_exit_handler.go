package http

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"loan-agent/domain"
	"loan-agent/service"
)

type DebtExitHandler struct {
	service *service.DebtExitService
}

func NewDebtExitHandler(service *service.DebtExitService) *DebtExitHandler {
	return &DebtExitHandler{service: service}
}

func (h *DebtExitHandler) CalculateDebtExitPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	var input domain.DebtExitInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.service.CalculateDebtExitPlan(input)
	if err != nil {
		log.Printf("Error calculating debt exit plan: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Codificar JSON en buffer primero para evitar escribir header si falla
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(result); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := buf.WriteTo(w); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}
