package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

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

func NewAIService() *AIService {
	apiKey := os.Getenv("OPENAI_API_KEY")
	enabled := apiKey != ""

	return &AIService{
		apiKey:  apiKey,
		apiURL:  "https://api.openai.com/v1/chat/completions",
		enabled: enabled,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GenerateTermRecommendationExplanation genera una explicación inteligente para una recomendación de plazo
func (s *AIService) GenerateTermRecommendationExplanation(
	amount float64,
	interestRate float64,
	recommendedTerm int,
	monthlyPayment float64,
	totalInterest float64,
	preference string,
	alternativeTerms []struct {
		Term           int
		MonthlyPayment float64
		TotalInterest  float64
	},
) string {
	if !s.enabled {
		return s.generateFallbackExplanation(recommendedTerm, monthlyPayment, totalInterest, preference)
	}

	// Convertir a córdobas para mostrar ambas monedas
	usdToNIO := GetUSDToNIORate()
	amountNIO := amount * usdToNIO
	monthlyPaymentNIO := monthlyPayment * usdToNIO
	totalInterestNIO := totalInterest * usdToNIO

	preferenceText := map[string]string{
		"minimize_interest": "minimizar el costo total de intereses",
		"minimize_payment":  "minimizar el pago mensual",
		"balanced":          "balance entre pago mensual y costo total",
	}
	preferenceDesc := preferenceText[preference]
	if preferenceDesc == "" {
		preferenceDesc = preference
	}

	prompt := fmt.Sprintf(`Analiza esta recomendación de préstamo para Nicaragua y genera una explicación clara y educativa.

CONTEXTO DEL PRÉSTAMO:
- Monto del préstamo: $%.2f USD (C$%.2f NIO)
- Tasa de interés anual: %.2f%%
- Plazo recomendado: %d meses (%.1f años)
- Pago mensual: $%.2f USD (C$%.2f NIO)
- Total de intereses a pagar: $%.2f USD (C$%.2f NIO)
- Preferencia del usuario: %s

INSTRUCCIONES:
1. Explica de manera clara y sencilla por qué este plazo de %d meses es la mejor opción según la preferencia del usuario (%s).
2. Menciona específicamente los montos en ambas monedas (USD y NIO).
3. Explica el balance entre el pago mensual y el costo total de intereses.
4. Proporciona contexto sobre cómo esto se relaciona con el mercado crediticio nicaragüense (préstamos personales, tarjetas de crédito, hipotecas).
5. Sé motivacional pero realista.

Genera una explicación de 3-4 oraciones que sea fácil de entender para cualquier persona.`,
		amount, amountNIO, interestRate, recommendedTerm, float64(recommendedTerm)/12.0,
		monthlyPayment, monthlyPaymentNIO, totalInterest, totalInterestNIO,
		preferenceDesc, recommendedTerm, preferenceDesc)

	explanation, err := s.callLLM(prompt)
	if err != nil {
		log.Printf("Error calling AI service for term recommendation: %v", err)
		return s.generateFallbackExplanation(recommendedTerm, monthlyPayment, totalInterest, preference)
	}

	return explanation
}

// GenerateDebtStrategyExplanation genera una explicación inteligente para una estrategia de deudas
func (s *AIService) GenerateDebtStrategyExplanation(
	strategy string,
	totalDebt float64,
	totalInterestPaid float64,
	monthsToPayoff int,
	debts []struct {
		Name         string
		Amount       float64
		InterestRate float64
	},
	comparison *struct {
		SnowballInterest  float64
		AvalancheInterest float64
		InterestSaved     float64
		MonthsSaved       int
	},
) string {
	if !s.enabled {
		return s.generateFallbackDebtExplanation(strategy, totalInterestPaid, monthsToPayoff)
	}

	// Convertir a córdobas
	usdToNIO := GetUSDToNIORate()
	totalDebtNIO := totalDebt * usdToNIO
	totalInterestNIO := totalInterestPaid * usdToNIO

	strategyName := "Snowball (Bola de Nieve)"
	strategyDesc := "Esta estrategia prioriza pagar primero las deudas más pequeñas, generando motivación psicológica al ver progreso rápido."
	if strategy == "avalanche" {
		strategyName = "Avalanche (Avalancha)"
		strategyDesc = "Esta estrategia prioriza pagar primero las deudas con mayor tasa de interés, minimizando el costo total de intereses."
	}

	comparisonText := ""
	if comparison != nil {
		comparisonSnowballNIO := comparison.SnowballInterest * usdToNIO
		comparisonAvalancheNIO := comparison.AvalancheInterest * usdToNIO
		interestSavedNIO := comparison.InterestSaved * usdToNIO
		comparisonText = fmt.Sprintf(`
COMPARACIÓN DE ESTRATEGIAS:
- Método Snowball: $%.2f USD (C$%.2f NIO) en intereses, %d meses
- Método Avalanche: $%.2f USD (C$%.2f NIO) en intereses, %d meses
- Ahorro con la estrategia recomendada: $%.2f USD (C$%.2f NIO) y %d meses menos`,
			comparison.SnowballInterest, comparisonSnowballNIO, monthsToPayoff,
			comparison.AvalancheInterest, comparisonAvalancheNIO, monthsToPayoff-comparison.MonthsSaved,
			comparison.InterestSaved, interestSavedNIO, comparison.MonthsSaved)
	}

	prompt := fmt.Sprintf(`Analiza este plan de salida de deudas para Nicaragua y genera una explicación clara, motivacional y educativa.

ESTRATEGIA RECOMENDADA: %s
%s

RESUMEN FINANCIERO:
- Total de deuda: $%.2f USD (C$%.2f NIO)
- Total de intereses a pagar: $%.2f USD (C$%.2f NIO)
- Tiempo estimado para pagar todo: %d meses (%.1f años)

DEUDAS INCLUIDAS:
%s
%s

INSTRUCCIONES:
1. Explica de manera clara qué es la estrategia %s y cómo funciona.
2. Menciona todos los montos en ambas monedas (USD y NIO).
3. Explica por qué esta estrategia es beneficiosa para el usuario, considerando el contexto del sistema crediticio nicaragüense (préstamos personales, tarjetas de crédito, hipotecas).
4. Si hay comparación disponible, explica las diferencias entre las estrategias.
5. Proporciona consejos prácticos y motivacionales para ayudar al usuario a mantenerse comprometido con el plan.
6. Sé específico con los números y tiempos.

Genera una explicación de 4-5 oraciones que sea fácil de entender y que motive al usuario a seguir el plan.`,
		strategyName, strategyDesc,
		totalDebt, totalDebtNIO, totalInterestPaid, totalInterestNIO,
		monthsToPayoff, float64(monthsToPayoff)/12.0,
		s.formatDebts(debts),
		comparisonText,
		strategyName)

	explanation, err := s.callLLM(prompt)
	if err != nil {
		log.Printf("Error calling AI service for debt strategy: %v", err)
		return s.generateFallbackDebtExplanation(strategy, totalInterestPaid, monthsToPayoff)
	}

	return explanation
}

func (s *AIService) callLLM(prompt string) (string, error) {
	reqBody := OpenAIRequest{
		Model: "gpt-4o-mini",
		Messages: []Message{
			{
				Role:    "system",
				Content: "Eres un asesor financiero experto especializado en el mercado crediticio de Nicaragua. Proporcionas explicaciones claras, precisas y motivacionales en español. Conoces profundamente el sistema crediticio nicaragüense, incluyendo préstamos personales, tarjetas de crédito e hipotecas. Siempre presentas los montos tanto en dólares estadounidenses (USD) como en córdobas nicaragüenses (NIO), usando una tasa de cambio aproximada cuando sea necesario. Tus explicaciones son educativas, fáciles de entender y ayudan a los usuarios a tomar decisiones financieras informadas.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens: 300,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", s.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var openAIResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return "", err
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	return openAIResp.Choices[0].Message.Content, nil
}

func (s *AIService) formatDebts(debts []struct {
	Name         string
	Amount       float64
	InterestRate float64
}) string {
	if len(debts) == 0 {
		return ""
	}
	usdToNIO := GetUSDToNIORate()
	var result strings.Builder
	for _, debt := range debts {
		debtNIO := debt.Amount * usdToNIO
		result.WriteString(fmt.Sprintf("- %s: $%.2f USD (C$%.2f NIO) al %.2f%% anual\n",
			debt.Name, debt.Amount, debtNIO, debt.InterestRate))
	}
	return result.String()
}

func (s *AIService) generateFallbackExplanation(
	term int,
	monthlyPayment float64,
	totalInterest float64,
	preference string,
) string {
	usdToNIO := GetUSDToNIORate()
	monthlyPaymentNIO := monthlyPayment * usdToNIO
	totalInterestNIO := totalInterest * usdToNIO

	switch preference {
	case "minimize_interest":
		return fmt.Sprintf("Este plazo de %d meses está optimizado para minimizar el costo total de intereses ($%.2f USD / C$%.2f NIO), aunque requiere un pago mensual de $%.2f USD (C$%.2f NIO). Esta opción es ideal si buscas reducir el costo total del préstamo en el mercado crediticio nicaragüense.",
			term, totalInterest, totalInterestNIO, monthlyPayment, monthlyPaymentNIO)
	case "minimize_payment":
		return fmt.Sprintf("Este plazo de %d meses minimiza tu pago mensual a $%.2f USD (C$%.2f NIO), ideal para mantener flexibilidad en tu presupuesto mensual. Perfecto para préstamos personales o cuando necesitas más espacio financiero cada mes.",
			term, monthlyPayment, monthlyPaymentNIO)
	default:
		return fmt.Sprintf("Este plazo de %d meses ofrece un balance óptimo entre pago mensual ($%.2f USD / C$%.2f NIO) y costo total de intereses ($%.2f USD / C$%.2f NIO). Esta recomendación considera tanto tu capacidad de pago mensual como el costo total del préstamo en el contexto nicaragüense.",
			term, monthlyPayment, monthlyPaymentNIO, totalInterest, totalInterestNIO)
	}
}

func (s *AIService) generateFallbackDebtExplanation(
	strategy string,
	totalInterest float64,
	months int,
) string {
	usdToNIO := GetUSDToNIORate()
	totalInterestNIO := totalInterest * usdToNIO

	strategyName := "Snowball (Bola de Nieve)"
	if strategy == "avalanche" {
		strategyName = "Avalanche (Avalancha)"
	}
	return fmt.Sprintf("Con la estrategia %s, pagarás $%.2f USD (C$%.2f NIO) en intereses y terminarás de pagar todas tus deudas en %d meses (%.1f años). %s Esta estrategia es efectiva para manejar diferentes tipos de crédito en Nicaragua, incluyendo préstamos personales, tarjetas de crédito e hipotecas.",
		strategyName, totalInterest, totalInterestNIO, months, float64(months)/12.0,
		s.getStrategyTip(strategy))
}

func (s *AIService) getStrategyTip(strategy string) string {
	if strategy == "snowball" {
		return "Esta estrategia te ayuda a mantener la motivación al ver progreso rápido pagando deudas pequeñas primero, lo cual es especialmente útil cuando tienes múltiples tarjetas de crédito o préstamos personales en Nicaragua."
	}
	return "Esta estrategia minimiza el costo total pagando primero las deudas con mayor interés, ideal para reducir significativamente los intereses acumulados en préstamos y tarjetas de crédito nicaragüenses."
}
