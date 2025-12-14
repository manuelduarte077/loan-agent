package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// Constants
const (
	defaultOpenAIModel    = "gpt-4o-mini"
	defaultOpenAIURL      = "https://api.openai.com/v1/chat/completions"
	defaultHTTPTimeout    = 30 * time.Second
	defaultMaxTokens      = 400
	debtStrategyMaxTokens = 500
)

// Domain types for better type safety
type (
	DebtInfo struct {
		Name         string
		Amount       float64
		InterestRate float64
	}

	AlternativeTerm struct {
		Term           int
		MonthlyPayment float64
		TotalInterest  float64
	}

	StrategyComparison struct {
		SnowballInterest  float64
		AvalancheInterest float64
		InterestSaved     float64
		MonthsSaved       int
	}
)

// Configuration maps
var (
	preferenceDescriptions = map[string]string{
		"minimize_interest": "minimizar el costo total de intereses",
		"minimize_payment":  "minimizar el pago mensual",
		"balanced":          "balance entre pago mensual y costo total",
	}

	strategyInfo = map[string]struct {
		Name        string
		Description string
		Tip         string
	}{
		"snowball": {
			Name:        "Snowball (Bola de Nieve)",
			Description: "Prioriza pagar primero las deudas más pequeñas, generando motivación psicológica al ver progreso rápido.",
			Tip:         "Ideal si necesitas ver resultados rápidos para mantenerte motivado. Cada deuda pagada libera capital que puedes aplicar a la siguiente.",
		},
		"avalanche": {
			Name:        "Avalanche (Avalancha)",
			Description: "Prioriza pagar primero las deudas con mayor tasa de interés, minimizando matemáticamente el costo total de intereses.",
			Tip:         "Ideal si tu objetivo principal es minimizar el costo financiero total. Requiere disciplina pero maximiza el ahorro.",
		},
	}

	systemPrompt = `Eres un asesor financiero certificado especializado en el mercado crediticio de Nicaragua. 
Tu expertise incluye:
- Análisis de préstamos personales, tarjetas de crédito e hipotecas
- Estrategias de gestión de deuda (Snowball y Avalanche)
- Planificación financiera personal
- Optimización de términos crediticios

REGLAS DE COMUNICACIÓN:
1. Siempre presenta montos en USD y NIO usando la tasa de cambio proporcionada
2. Usa números exactos, nunca aproximaciones
3. Sé específico, claro y profesional
4. Proporciona contexto financiero relevante para Nicaragua
5. Mantén un tono motivacional pero realista
6. Evita jerga técnica innecesaria, pero sé preciso con términos financieros`
)

// AIService handles AI-powered financial explanations
type AIService struct {
	apiKey     string
	apiURL     string
	enabled    bool
	httpClient *http.Client
}

type OpenAIRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

// NewAIService creates a new AI service instance
func NewAIService() *AIService {
	apiKey := os.Getenv("OPENAI_API_KEY")
	return &AIService{
		apiKey:  apiKey,
		apiURL:  defaultOpenAIURL,
		enabled: apiKey != "",
		httpClient: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

func convertToNIO(usdAmount float64) float64 {
	return usdAmount * GetUSDToNIORate()
}

func formatCurrencyPair(usdAmount float64) string {
	return fmt.Sprintf("$%.2f USD (C$%.2f NIO)", usdAmount, convertToNIO(usdAmount))
}

func getPreferenceDescription(preference string) string {
	if desc, ok := preferenceDescriptions[preference]; ok {
		return desc
	}
	return preference
}

func getStrategyInfo(strategy string) (name, description, tip string) {
	if info, ok := strategyInfo[strategy]; ok {
		return info.Name, info.Description, info.Tip
	}
	return "Estrategia desconocida", "", ""
}

func formatMonthsAsYears(months int) float64 {
	return float64(months) / 12.0
}

// ============================================================================
// Term Recommendation Explanations
// ============================================================================

// GenerateTermRecommendationExplanation generates an intelligent explanation for a term recommendation
func (s *AIService) GenerateTermRecommendationExplanation(
	amount float64,
	interestRate float64,
	recommendedTerm int,
	monthlyPayment float64,
	totalInterest float64,
	preference string,
	alternativeTerms []AlternativeTerm,
) string {
	if !s.enabled {
		return s.generateTermFallback(recommendedTerm, monthlyPayment, totalInterest, preference)
	}

	prompt := s.buildTermRecommendationPrompt(
		amount, interestRate, recommendedTerm,
		monthlyPayment, totalInterest, preference, alternativeTerms,
	)

	explanation, err := s.callLLM(prompt)
	if err != nil {
		log.Printf("Error calling AI for term recommendation: %v", err)
		return s.generateTermFallback(recommendedTerm, monthlyPayment, totalInterest, preference)
	}

	return explanation
}

func (s *AIService) buildTermRecommendationPrompt(
	amount, interestRate float64,
	recommendedTerm int,
	monthlyPayment, totalInterest float64,
	preference string,
	alternatives []AlternativeTerm,
) string {
	preferenceDesc := getPreferenceDescription(preference)
	alternativesText := s.formatAlternatives(alternatives)
	totalCost := amount + totalInterest

	return fmt.Sprintf(`Como asesor financiero, analiza esta recomendación de préstamo para Nicaragua y genera una explicación profesional y educativa.

DATOS DEL PRÉSTAMO:
- Monto solicitado: %s
- Tasa de interés anual: %.2f%%
- Plazo recomendado: %d meses (%.1f años)
- Cuota mensual: %s
- Intereses totales: %s
- Costo total del préstamo: %s
- Objetivo del cliente: %s%s

ANÁLISIS REQUERIDO:
1. JUSTIFICACIÓN FINANCIERA: Explica por qué este plazo de %d meses es óptimo según el objetivo "%s". 
   Incluye análisis del costo de oportunidad y el impacto en el flujo de caja mensual.

2. COMPARACIÓN DE COSTOS: Menciona específicamente:
   - El costo total de financiamiento (%s en intereses)
   - El costo mensual (%s por mes)
   - El costo total a pagar (%s incluyendo capital e intereses)
   Todos los montos deben aparecer en USD y NIO.

3. ANÁLISIS DE ALTERNATIVAS: %s
   Si hay alternativas, explica brevemente por qué esta opción es superior considerando el objetivo del cliente.

4. RECOMENDACIÓN PRÁCTICA: Proporciona una recomendación específica y accionable basada en estos números exactos.

FORMATO:
- Máximo 4-5 oraciones
- Usa números exactos proporcionados (no aproximaciones)
- Menciona ambos montos (USD y NIO) siempre
- Sé específico y profesional`,
		formatCurrencyPair(amount), interestRate,
		recommendedTerm, formatMonthsAsYears(recommendedTerm),
		formatCurrencyPair(monthlyPayment),
		formatCurrencyPair(totalInterest),
		formatCurrencyPair(totalCost),
		preferenceDesc, alternativesText,
		recommendedTerm, preferenceDesc,
		formatCurrencyPair(totalInterest),
		formatCurrencyPair(monthlyPayment),
		formatCurrencyPair(totalCost),
		s.getAlternativesAnalysisText(len(alternatives)))
}

func (s *AIService) getAlternativesAnalysisText(count int) string {
	if count > 0 {
		return fmt.Sprintf("Se evaluaron %d alternativas de plazo.", count)
	}
	return "Esta es la única opción viable dentro de los parámetros establecidos."
}

func (s *AIService) formatAlternatives(alternatives []AlternativeTerm) string {
	if len(alternatives) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("\n\nALTERNATIVAS EVALUADAS:\n")
	for _, alt := range alternatives {
		builder.WriteString(fmt.Sprintf("- %d meses: Cuota %s | Intereses %s\n",
			alt.Term, formatCurrencyPair(alt.MonthlyPayment), formatCurrencyPair(alt.TotalInterest)))
	}
	return builder.String()
}

// GenerateAlternativeTermExplanation generates explanation for alternative terms
func (s *AIService) GenerateAlternativeTermExplanation(
	amount, interestRate float64,
	term int,
	monthlyPayment, totalInterest float64,
	preference string,
	bestTerm int,
	bestMonthlyPayment, bestTotalInterest float64,
) string {
	if !s.enabled {
		return s.generateTermFallback(term, monthlyPayment, totalInterest, preference)
	}

	comparison := s.buildTermComparison(term, monthlyPayment, totalInterest, bestTerm, bestMonthlyPayment, bestTotalInterest)
	prompt := s.buildAlternativeTermPrompt(amount, interestRate, term, monthlyPayment, totalInterest, preference, bestTerm, bestMonthlyPayment, bestTotalInterest, comparison)

	explanation, err := s.callLLM(prompt)
	if err != nil {
		log.Printf("Error calling AI for alternative term: %v", err)
		return s.generateTermFallback(term, monthlyPayment, totalInterest, preference)
	}

	return explanation
}

func (s *AIService) buildAlternativeTermPrompt(
	amount, interestRate float64,
	term int,
	monthlyPayment, totalInterest float64,
	preference string,
	bestTerm int,
	bestMonthlyPayment, bestTotalInterest float64,
	comparison string,
) string {
	return fmt.Sprintf(`Genera una explicación concisa para esta alternativa de plazo de préstamo en Nicaragua.

DATOS DE ESTA OPCIÓN:
- Monto: %s
- Tasa anual: %.2f%%
- Plazo: %d meses (%.1f años)
- Cuota mensual: %s
- Intereses totales: %s
- Objetivo: %s

COMPARACIÓN CON LA MEJOR OPCIÓN (%d meses):
- Mejor opción: Cuota %s | Intereses %s
%s

INSTRUCCIONES:
1. Describe las características específicas de este plazo de %d meses
2. Compara con la mejor opción usando los números exactos
3. Explica cuándo esta alternativa podría ser preferible
4. Menciona montos en USD y NIO

Formato: 2-3 oraciones, específico y claro.`,
		formatCurrencyPair(amount), interestRate, term, formatMonthsAsYears(term),
		formatCurrencyPair(monthlyPayment), formatCurrencyPair(totalInterest),
		getPreferenceDescription(preference),
		bestTerm, formatCurrencyPair(bestMonthlyPayment), formatCurrencyPair(bestTotalInterest),
		comparison, term)
}

func (s *AIService) buildTermComparison(term int, monthlyPayment, totalInterest float64, bestTerm int, bestMonthlyPayment, bestTotalInterest float64) string {
	var parts []string

	if term != bestTerm {
		diff := term - bestTerm
		if diff < 0 {
			parts = append(parts, fmt.Sprintf("Este plazo es %d meses más corto que la mejor opción.", -diff))
		} else {
			parts = append(parts, fmt.Sprintf("Este plazo es %d meses más largo que la mejor opción.", diff))
		}
	}

	paymentDiff := monthlyPayment - bestMonthlyPayment
	if paymentDiff != 0 {
		if paymentDiff > 0 {
			parts = append(parts, fmt.Sprintf("Requiere una cuota %s mayor.", formatCurrencyPair(paymentDiff)))
		} else {
			parts = append(parts, fmt.Sprintf("Tiene una cuota %s menor.", formatCurrencyPair(-paymentDiff)))
		}
	}

	interestDiff := totalInterest - bestTotalInterest
	if interestDiff != 0 {
		if interestDiff > 0 {
			parts = append(parts, fmt.Sprintf("Incrementa los intereses en %s.", formatCurrencyPair(interestDiff)))
		} else {
			parts = append(parts, fmt.Sprintf("Reduce los intereses en %s.", formatCurrencyPair(-interestDiff)))
		}
	}

	if len(parts) == 0 {
		return "Esta opción es idéntica a la mejor opción."
	}

	return strings.Join(parts, " ")
}

// ============================================================================
// Debt Strategy Explanations
// ============================================================================

// GenerateDebtStrategyExplanation generates intelligent explanation for debt payoff strategy
func (s *AIService) GenerateDebtStrategyExplanation(
	strategy string,
	totalDebt, totalInterestPaid float64,
	monthsToPayoff int,
	debts []DebtInfo,
	comparison *StrategyComparison,
) string {
	strategyName, strategyDesc, strategyTip := getStrategyInfo(strategy)
	if strategyDesc == "" {
		strategyDesc = "Estrategia de pago de deudas optimizada."
	}

	paymentOrder := s.buildPaymentOrder(strategy, debts)
	comparisonText := s.buildDebtComparisonText(strategy, monthsToPayoff, comparison)
	debtsText := s.formatDebtsList(debts)

	if !s.enabled {
		return s.generateDebtFallback(strategy, strategyName, strategyTip, totalDebt, totalInterestPaid, monthsToPayoff, comparison, paymentOrder)
	}

	prompt := s.buildDebtStrategyPrompt(strategyName, strategyDesc, totalDebt, totalInterestPaid, monthsToPayoff, debtsText, paymentOrder, comparisonText)

	explanation, err := s.callLLMWithMaxTokens(prompt, debtStrategyMaxTokens)
	if err != nil {
		log.Printf("Error calling AI for debt strategy: %v", err)
		return s.generateDebtFallback(strategy, strategyName, strategyTip, totalDebt, totalInterestPaid, monthsToPayoff, comparison, paymentOrder)
	}

	return explanation
}

func (s *AIService) buildDebtStrategyPrompt(
	strategyName, strategyDesc string,
	totalDebt, totalInterestPaid float64,
	monthsToPayoff int,
	debtsText, paymentOrder, comparisonText string,
) string {
	totalCost := totalDebt + totalInterestPaid

	return fmt.Sprintf(`Como asesor financiero certificado, analiza este plan de salida de deudas para Nicaragua y genera una explicación profesional y accionable.

ESTRATEGIA RECOMENDADA: %s
Descripción: %s

ANÁLISIS FINANCIERO:
- Deuda total inicial: %s
- Intereses a pagar: %s
- Tiempo estimado: %d meses (%.1f años)
- Costo total del plan: %s

PORTFOLIO DE DEUDAS:
%s

ORDEN DE PAGO ESTRATÉGICO:
%s%s

REQUERIMIENTOS PARA LA EXPLICACIÓN:
1. DEFINICIÓN Y MECÁNICA: Explica qué es %s y cómo funciona paso a paso (1-2 oraciones)

2. ANÁLISIS CUANTITATIVO: Proporciona números específicos:
   - Deuda inicial: %s
   - Costo de financiamiento: %s en intereses
   - Tiempo total: %d meses (%.1f años)
   - Inversión total requerida: %s
   Todos en USD y NIO.

3. JUSTIFICACIÓN ESTRATÉGICA: Explica por qué este orden de pago es óptimo para esta situación específica de deudas.

4. COMPARACIÓN ESTRATÉGICA: %s
   Si hay comparación disponible, menciona el ahorro específico comparado con la otra estrategia.

5. RECOMENDACIÓN PRÁCTICA: Proporciona un consejo específico y accionable para mantener la disciplina y cumplir el plan.

FORMATO:
- Máximo 5-6 oraciones informativas
- Usa números exactos proporcionados
- Sé específico, profesional y motivacional
- Evita repeticiones innecesarias`,
		strategyName, strategyDesc,
		formatCurrencyPair(totalDebt), formatCurrencyPair(totalInterestPaid),
		monthsToPayoff, formatMonthsAsYears(monthsToPayoff),
		formatCurrencyPair(totalCost),
		debtsText, paymentOrder, comparisonText,
		strategyName,
		formatCurrencyPair(totalDebt), formatCurrencyPair(totalInterestPaid),
		monthsToPayoff, formatMonthsAsYears(monthsToPayoff),
		formatCurrencyPair(totalCost),
		s.getComparisonAnalysisText(comparisonText))
}

func (s *AIService) getComparisonAnalysisText(comparisonText string) string {
	if comparisonText != "" {
		return "Compara esta estrategia con la alternativa, destacando el ahorro específico en intereses y tiempo."
	}
	return "No hay comparación disponible con otras estrategias."
}

func (s *AIService) buildPaymentOrder(strategy string, debts []DebtInfo) string {
	if len(debts) == 0 {
		return ""
	}

	sorted := make([]DebtInfo, len(debts))
	copy(sorted, debts)

	if strategy == "snowball" {
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Amount < sorted[j].Amount
		})
	} else {
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].InterestRate > sorted[j].InterestRate
		})
	}

	var builder strings.Builder
	strategyName, _, _ := getStrategyInfo(strategy)
	builder.WriteString(fmt.Sprintf("\nCon %s, el orden de pago es:\n", strategyName))
	for i, debt := range sorted {
		builder.WriteString(fmt.Sprintf("%d. %s: %s (%.2f%% anual)\n",
			i+1, debt.Name, formatCurrencyPair(debt.Amount), debt.InterestRate))
	}
	return builder.String()
}

func (s *AIService) buildDebtComparisonText(strategy string, monthsToPayoff int, comparison *StrategyComparison) string {
	if comparison == nil {
		return ""
	}

	snowballMonths, avalancheMonths := s.calculateStrategyMonths(strategy, monthsToPayoff, comparison.MonthsSaved)
	monthsText := s.formatMonthsComparison(strategy, comparison.MonthsSaved)

	return fmt.Sprintf(`
COMPARACIÓN DE ESTRATEGIAS:
- Snowball: %s en intereses | %d meses (%.1f años)
- Avalanche: %s en intereses | %d meses (%.1f años)
- Con %s ahorras: %s en intereses y terminas %s`,
		formatCurrencyPair(comparison.SnowballInterest), snowballMonths, formatMonthsAsYears(snowballMonths),
		formatCurrencyPair(comparison.AvalancheInterest), avalancheMonths, formatMonthsAsYears(avalancheMonths),
		s.getStrategyName(strategy), formatCurrencyPair(comparison.InterestSaved), monthsText)
}

func (s *AIService) calculateStrategyMonths(strategy string, monthsToPayoff, monthsSaved int) (snowball, avalanche int) {
	// MonthsSaved = snowballMonths - avalancheMonths
	if strategy == "avalanche" {
		avalanche = monthsToPayoff
		snowball = avalanche + monthsSaved
	} else {
		snowball = monthsToPayoff
		avalanche = snowball - monthsSaved
	}
	return
}

func (s *AIService) formatMonthsComparison(strategy string, monthsSaved int) string {
	if strategy == "avalanche" {
		if monthsSaved > 0 {
			return fmt.Sprintf("%d meses antes", monthsSaved)
		} else if monthsSaved < 0 {
			return fmt.Sprintf("%d meses después", -monthsSaved)
		}
		return "en el mismo tiempo"
	}
	// snowball
	if monthsSaved > 0 {
		return fmt.Sprintf("%d meses después", monthsSaved)
	} else if monthsSaved < 0 {
		return fmt.Sprintf("%d meses antes", -monthsSaved)
	}
	return "en el mismo tiempo"
}

func (s *AIService) getStrategyName(strategy string) string {
	name, _, _ := getStrategyInfo(strategy)
	return name
}

func (s *AIService) formatDebtsList(debts []DebtInfo) string {
	if len(debts) == 0 {
		return "No se proporcionaron detalles de las deudas."
	}

	var builder strings.Builder
	for _, debt := range debts {
		builder.WriteString(fmt.Sprintf("- %s: %s al %.2f%% anual\n",
			debt.Name, formatCurrencyPair(debt.Amount), debt.InterestRate))
	}
	return builder.String()
}

// ============================================================================
// Fallback Explanations
// ============================================================================

func (s *AIService) generateTermFallback(term int, monthlyPayment, totalInterest float64, preference string) string {
	totalCost := monthlyPayment * float64(term)

	switch preference {
	case "minimize_interest":
		return fmt.Sprintf("Este plazo de %d meses minimiza el costo total de intereses (%s), aunque requiere una cuota mensual de %s. El costo total del préstamo será %s. Esta opción es ideal si tu prioridad es reducir el costo financiero total en el mercado crediticio nicaragüense.",
			term, formatCurrencyPair(totalInterest), formatCurrencyPair(monthlyPayment), formatCurrencyPair(totalCost))
	case "minimize_payment":
		return fmt.Sprintf("Este plazo de %d meses minimiza tu cuota mensual a %s, proporcionando mayor flexibilidad presupuestaria. Pagarás %s en intereses para un costo total de %s. Ideal para préstamos personales cuando necesitas maximizar tu capacidad de pago mensual.",
			term, formatCurrencyPair(monthlyPayment), formatCurrencyPair(totalInterest), formatCurrencyPair(totalCost))
	default:
		return fmt.Sprintf("Este plazo de %d meses ofrece un balance óptimo entre cuota mensual (%s) y costo total de intereses (%s). El costo total del préstamo será %s. Esta recomendación equilibra tu capacidad de pago mensual con el costo financiero total en el contexto nicaragüense.",
			term, formatCurrencyPair(monthlyPayment), formatCurrencyPair(totalInterest), formatCurrencyPair(totalCost))
	}
}

func (s *AIService) generateDebtFallback(
	strategy, strategyName, strategyTip string,
	totalDebt, totalInterest float64,
	months int,
	comparison *StrategyComparison,
	paymentOrder string,
) string {
	totalCost := totalDebt + totalInterest
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("La estrategia %s te permitirá liquidar todas tus deudas en %d meses (%.1f años). ", strategyName, months, formatMonthsAsYears(months)))
	builder.WriteString(fmt.Sprintf("Tu deuda inicial es %s y pagarás %s en intereses, para un costo total de %s. ", formatCurrencyPair(totalDebt), formatCurrencyPair(totalInterest), formatCurrencyPair(totalCost)))

	if paymentOrder != "" {
		builder.WriteString("\n\nOrden de pago:\n")
		builder.WriteString(paymentOrder)
	}

	if comparison != nil {
		builder.WriteString(s.buildFallbackComparison(strategy, comparison))
	}

	if strategyTip != "" {
		builder.WriteString(fmt.Sprintf("\n\nRecomendación: %s", strategyTip))
	}

	return builder.String()
}

func (s *AIService) buildFallbackComparison(strategy string, comparison *StrategyComparison) string {
	monthsDiff := comparison.MonthsSaved
	var text string

	if strategy == "snowball" {
		if comparison.InterestSaved > 0 {
			if monthsDiff > 0 {
				text = fmt.Sprintf("\nComparado con Avalanche, pagarás %s más en intereses y tomará %d meses más, pero ofrece mayor motivación psicológica.",
					formatCurrencyPair(comparison.InterestSaved), monthsDiff)
			} else if monthsDiff < 0 {
				text = fmt.Sprintf("\nComparado con Avalanche, pagarás %s más en intereses pero terminarás %d meses antes, combinando motivación con eficiencia temporal.",
					formatCurrencyPair(comparison.InterestSaved), -monthsDiff)
			} else {
				text = fmt.Sprintf("\nComparado con Avalanche, pagarás %s más en intereses en el mismo tiempo, pero con mayor motivación psicológica.",
					formatCurrencyPair(comparison.InterestSaved))
			}
		}
	} else {
		if comparison.InterestSaved > 0 {
			if monthsDiff > 0 {
				text = fmt.Sprintf("\nComparado con Snowball, ahorrarás %s en intereses y terminarás %d meses antes, minimizando tu costo financiero total.",
					formatCurrencyPair(comparison.InterestSaved), monthsDiff)
			} else if monthsDiff < 0 {
				text = fmt.Sprintf("\nComparado con Snowball, ahorrarás %s en intereses aunque tomará %d meses más, maximizando el ahorro financiero.",
					formatCurrencyPair(comparison.InterestSaved), -monthsDiff)
			} else {
				text = fmt.Sprintf("\nComparado con Snowball, ahorrarás %s en intereses en el mismo tiempo, optimizando el costo financiero.",
					formatCurrencyPair(comparison.InterestSaved))
			}
		}
	}

	return text
}

// ============================================================================
// LLM Communication
// ============================================================================

func (s *AIService) callLLM(prompt string) (string, error) {
	return s.callLLMWithMaxTokens(prompt, defaultMaxTokens)
}

func (s *AIService) callLLMWithMaxTokens(prompt string, maxTokens int) (string, error) {
	reqBody := OpenAIRequest{
		Model: defaultOpenAIModel,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
		MaxTokens: maxTokens,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", s.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var openAIResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	return openAIResp.Choices[0].Message.Content, nil
}
