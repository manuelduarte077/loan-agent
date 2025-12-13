package http

import (
	"encoding/json"
	"net/http"

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

	var input domain.DebtExitInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.service.CalculateDebtExitPlan(input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

