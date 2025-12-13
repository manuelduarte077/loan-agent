package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

	prompt := fmt.Sprintf(`Eres un asesor financiero experto. Analiza esta recomendación de préstamo y genera una explicación clara y útil en español.

Contexto:
- Monto del préstamo: $%.2f
- Tasa de interés anual: %.2f%%
- Plazo recomendado: %d meses
- Pago mensual: $%.2f
- Total de intereses: $%.2f
- Preferencia del usuario: %s

Genera una explicación breve (2-3 oraciones) que explique por qué este plazo es recomendado, considerando el balance entre pago mensual y costo total. Sé específico con los números y proporciona contexto útil.`,
		amount, interestRate, recommendedTerm, monthlyPayment, totalInterest, preference)

	explanation, err := s.callLLM(prompt)
	if err != nil {
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

	prompt := fmt.Sprintf(`Eres un asesor financiero experto. Analiza este plan de salida de deudas y genera una explicación motivacional y útil en español.

Estrategia: %s
Total de deuda: $%.2f
Total de intereses a pagar: $%.2f
Meses para pagar todo: %d

Deudas:
%s

%s

Genera una explicación breve (3-4 oraciones) que explique la estrategia, los beneficios, y motive al usuario. Incluye datos específicos y consejos prácticos.`,
		strategy, totalDebt, totalInterestPaid, monthsToPayoff,
		s.formatDebts(debts),
		s.formatComparison(comparison))

	explanation, err := s.callLLM(prompt)
	if err != nil {
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
				Content: "Eres un asesor financiero experto y amigable. Proporciona explicaciones claras, precisas y motivacionales en español.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens: 200,
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
		return "", fmt.Errorf("API error: %s", string(body))
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
	var result string
	for _, debt := range debts {
		result += fmt.Sprintf("- %s: $%.2f al %.2f%% anual\n", debt.Name, debt.Amount, debt.InterestRate)
	}
	return result
}

func (s *AIService) formatComparison(comparison *struct {
	SnowballInterest  float64
	AvalancheInterest float64
	InterestSaved     float64
	MonthsSaved       int
}) string {
	if comparison == nil {
		return ""
	}
	return fmt.Sprintf(`Comparación:
- Método Snowball: $%.2f en intereses, %d meses
- Método Avalanche: $%.2f en intereses, %d meses
- Ahorro con Avalanche: $%.2f y %d meses menos`,
		comparison.SnowballInterest, 0, // meses calculados separadamente
		comparison.AvalancheInterest, 0,
		comparison.InterestSaved, comparison.MonthsSaved)
}

func (s *AIService) generateFallbackExplanation(
	term int,
	monthlyPayment float64,
	totalInterest float64,
	preference string,
) string {
	switch preference {
	case "minimize_interest":
		return fmt.Sprintf("Este plazo de %d meses está optimizado para minimizar el costo total de intereses ($%.2f), aunque requiere un pago mensual de $%.2f.", term, totalInterest, monthlyPayment)
	case "minimize_payment":
		return fmt.Sprintf("Este plazo de %d meses minimiza tu pago mensual a $%.2f, ideal para mantener flexibilidad en tu presupuesto mensual.", term, monthlyPayment)
	default:
		return fmt.Sprintf("Este plazo de %d meses ofrece un balance óptimo entre pago mensual ($%.2f) y costo total de intereses ($%.2f).", term, monthlyPayment, totalInterest)
	}
}

func (s *AIService) generateFallbackDebtExplanation(
	strategy string,
	totalInterest float64,
	months int,
) string {
	strategyName := "Snowball"
	if strategy == "avalanche" {
		strategyName = "Avalanche"
	}
	return fmt.Sprintf("Con la estrategia %s, pagarás $%.2f en intereses y terminarás de pagar todas tus deudas en %d meses. %s",
		strategyName, totalInterest, months,
		s.getStrategyTip(strategy))
}

func (s *AIService) getStrategyTip(strategy string) string {
	if strategy == "snowball" {
		return "Esta estrategia te ayuda a mantener la motivación al ver progreso rápido pagando deudas pequeñas primero."
	}
	return "Esta estrategia minimiza el costo total pagando primero las deudas con mayor interés."
}
