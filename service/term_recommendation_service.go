package service

import (
	"errors"
	"fmt"
	"log"
	"sort"

	"loan-agent/domain"
)

type TermRecommendationService struct {
	loanService *LoanService
	aiService   *AIService
}

func NewTermRecommendationService(loanService *LoanService) *TermRecommendationService {
	return &TermRecommendationService{
		loanService: loanService,
		aiService:   NewAIService(),
	}
}

// RecommendTerm analiza diferentes plazos y recomienda el óptimo
func (s *TermRecommendationService) RecommendTerm(
	input domain.TermRecommendationInput,
) (domain.TermRecommendationResult, error) {

	// Validaciones
	if input.Amount <= 0 {
		return domain.TermRecommendationResult{}, errors.New("monto inválido")
	}
	if input.InterestRate < 0 {
		return domain.TermRecommendationResult{}, errors.New("tasa inválida")
	}
	if input.MinTermMonths <= 0 || input.MaxTermMonths <= 0 {
		return domain.TermRecommendationResult{}, errors.New("plazos inválidos")
	}
	if input.MinTermMonths > input.MaxTermMonths {
		return domain.TermRecommendationResult{}, errors.New("plazo mínimo mayor que máximo")
	}
	if input.MaxTermMonths > MaxTermMonths {
		return domain.TermRecommendationResult{}, fmt.Errorf("plazo máximo excede el límite de %d meses", MaxTermMonths)
	}
	// Validar que el rango no sea demasiado grande para evitar cálculos costosos
	if input.MaxTermMonths-input.MinTermMonths > MaxTermRangeMonths {
		return domain.TermRecommendationResult{}, fmt.Errorf("rango de plazos excede el máximo de %d meses", MaxTermRangeMonths)
	}
	if input.MaxMonthlyPayment <= 0 {
		return domain.TermRecommendationResult{}, errors.New("pago mensual máximo inválido")
	}

	preferences := map[string]bool{
		"minimize_interest": true,
		"minimize_payment":  true,
		"balanced":          true,
	}
	if !preferences[input.Preference] {
		return domain.TermRecommendationResult{}, errors.New("preferencia inválida")
	}

	recommendations := []domain.TermRecommendation{}

	// Calcular escenarios para cada plazo
	for term := input.MinTermMonths; term <= input.MaxTermMonths; term++ {
		loanInput := domain.LoanInput{
			Amount:       input.Amount,
			InterestRate: input.InterestRate,
			TermMonths:   term,
		}

		result, err := s.loanService.CalculateLoan(loanInput)
		if err != nil {
			log.Printf("Warning: failed to calculate loan for term %d: %v", term, err)
			continue
		}

		// Filtrar por pago mensual máximo
		if result.MonthlyPayment > input.MaxMonthlyPayment {
			continue
		}

		// Calcular score según preferencia
		score := s.calculateScore(result, input, term)
		reason := s.generateReason(result, input, term)

		recommendations = append(recommendations, domain.TermRecommendation{
			TermMonths:     term,
			MonthlyPayment: result.MonthlyPayment,
			TotalInterest:  result.TotalInterest,
			Score:          score,
			Reason:         reason,
		})
	}

	// Ordenar por score descendente
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	if len(recommendations) == 0 {
		return domain.TermRecommendationResult{}, errors.New("no se encontraron plazos válidos con el pago mensual máximo especificado")
	}

	recommendedTerm := recommendations[0].TermMonths

	// Generar explicación inteligente con IA para la recomendación principal
	topRecommendation := recommendations[0]
	alternativeTerms := []struct {
		Term           int
		MonthlyPayment float64
		TotalInterest  float64
	}{}

	// Agregar algunas alternativas para contexto
	maxAlternatives := 3
	for i := 1; i < len(recommendations) && i <= maxAlternatives; i++ {
		alternativeTerms = append(alternativeTerms, struct {
			Term           int
			MonthlyPayment float64
			TotalInterest  float64
		}{
			Term:           recommendations[i].TermMonths,
			MonthlyPayment: recommendations[i].MonthlyPayment,
			TotalInterest:  recommendations[i].TotalInterest,
		})
	}

	aiExplanation := s.aiService.GenerateTermRecommendationExplanation(
		input.Amount,
		input.InterestRate,
		topRecommendation.TermMonths,
		topRecommendation.MonthlyPayment,
		topRecommendation.TotalInterest,
		input.Preference,
		alternativeTerms,
	)

	// Actualizar la razón de la recomendación principal con la explicación de IA
	recommendations[0].Reason = aiExplanation

	return domain.TermRecommendationResult{
		RecommendedTerm: recommendedTerm,
		Recommendations: recommendations,
	}, nil
}

func (s *TermRecommendationService) calculateScore(
	result domain.LoanResult,
	input domain.TermRecommendationInput,
	term int,
) float64 {
	var score float64

	// Normalizar valores para scoring (0-10)
	maxPossibleInterest := input.Amount * (input.InterestRate / 100) * float64(input.MaxTermMonths) / 12
	minPossibleInterest := input.Amount * (input.InterestRate / 100) * float64(input.MinTermMonths) / 12

	interestRange := maxPossibleInterest - minPossibleInterest
	paymentRange := input.MaxMonthlyPayment - (input.Amount / float64(input.MaxTermMonths))

	interestScore := 0.0
	paymentScore := 0.0
	termScore := 0.0

	if interestRange > 0 {
		interestScore = 10.0 * (1.0 - (result.TotalInterest-minPossibleInterest)/interestRange)
	}
	if paymentRange > 0 {
		paymentScore = 10.0 * (1.0 - (result.MonthlyPayment-input.Amount/float64(input.MaxTermMonths))/paymentRange)
	}
	termScore = 10.0 * (1.0 - float64(term-input.MinTermMonths)/float64(input.MaxTermMonths-input.MinTermMonths))

	switch input.Preference {
	case "minimize_interest":
		score = 0.6*interestScore + 0.2*paymentScore + 0.2*termScore
	case "minimize_payment":
		score = 0.2*interestScore + 0.6*paymentScore + 0.2*termScore
	case "balanced":
		score = 0.4*interestScore + 0.4*paymentScore + 0.2*termScore
	}

	return roundTo2Decimals(score)
}

func (s *TermRecommendationService) generateReason(
	result domain.LoanResult,
	input domain.TermRecommendationInput,
	term int,
) string {
	switch input.Preference {
	case "minimize_interest":
		return "Plazo optimizado para minimizar el costo total de intereses"
	case "minimize_payment":
		return "Plazo optimizado para minimizar el pago mensual"
	case "balanced":
		return "Balance óptimo entre pago mensual y costo total"
	}
	return "Recomendación basada en los parámetros proporcionados"
}
