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

type TermRecommendationHandler struct {
	service *service.TermRecommendationService
}

func NewTermRecommendationHandler(service *service.TermRecommendationService) *TermRecommendationHandler {
	return &TermRecommendationHandler{service: service}
}

func (h *TermRecommendationHandler) RecommendTerm(w http.ResponseWriter, r *http.Request) {
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

	var input domain.TermRecommendationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.service.RecommendTerm(input)
	if err != nil {
		log.Printf("Error recommending term: %v", err)
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

