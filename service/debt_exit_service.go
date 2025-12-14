package service

import (
	"errors"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"

	"loan-agent/domain"
)

type DebtExitService struct {
	loanService *LoanService
}

func NewDebtExitService(loanService *LoanService) *DebtExitService {
	return &DebtExitService{
		loanService: loanService,
	}
}

// CalculateDebtExitPlan calcula el plan de salida de deudas usando snowball o avalanche
func (s *DebtExitService) CalculateDebtExitPlan(
	input domain.DebtExitInput,
) (domain.DebtExitResult, error) {

	if len(input.Debts) == 0 {
		return domain.DebtExitResult{}, errors.New("no se proporcionaron deudas")
	}
	if len(input.Debts) > MaxDebtsPerRequest {
		return domain.DebtExitResult{}, fmt.Errorf("número de deudas excede el máximo de %d", MaxDebtsPerRequest)
	}
	if input.AvailableMonthlyPayment <= 0 {
		return domain.DebtExitResult{}, errors.New("pago mensual disponible inválido")
	}

	// Validar nombres únicos
	debtNames := make(map[string]bool)
	for _, debt := range input.Debts {
		if debt.Name == "" {
			return domain.DebtExitResult{}, errors.New("nombre de deuda no puede estar vacío")
		}
		if debtNames[debt.Name] {
			return domain.DebtExitResult{}, fmt.Errorf("nombre de deuda duplicado: %s", debt.Name)
		}
		debtNames[debt.Name] = true
	}

	strategies := map[string]bool{
		"snowball":  true,
		"avalanche": true,
		"compare":   true,
	}
	if !strategies[input.Strategy] {
		return domain.DebtExitResult{}, errors.New("estrategia inválida")
	}

	// Validar que todas las deudas sean válidas
	totalMinimumPayments := 0.0
	for _, debt := range input.Debts {
		if debt.Amount <= 0 {
			return domain.DebtExitResult{}, errors.New("monto de deuda inválido")
		}
		if debt.Amount > MaxDebtAmount {
			return domain.DebtExitResult{}, fmt.Errorf("monto de deuda excede el máximo de $%.2f", MaxDebtAmount)
		}
		if debt.InterestRate < 0 {
			return domain.DebtExitResult{}, errors.New("tasa de interés inválida")
		}
		if debt.InterestRate > MaxInterestRate {
			return domain.DebtExitResult{}, fmt.Errorf("tasa de interés excede el máximo de %.2f%%", MaxInterestRate)
		}
		if debt.MinimumPayment <= 0 {
			return domain.DebtExitResult{}, errors.New("pago mínimo inválido")
		}
		// Validar que el pago mínimo sea razonable (al menos cubre el interés mensual)
		monthlyInterest := debt.Amount * (debt.InterestRate / 100) / 12
		if debt.MinimumPayment < monthlyInterest {
			return domain.DebtExitResult{}, fmt.Errorf("pago mínimo de %s ($%.2f) es menor que el interés mensual ($%.2f)", debt.Name, debt.MinimumPayment, monthlyInterest)
		}
		totalMinimumPayments += debt.MinimumPayment
	}

	if totalMinimumPayments > input.AvailableMonthlyPayment {
		return domain.DebtExitResult{}, errors.New("el pago mensual disponible es insuficiente para cubrir los pagos mínimos")
	}

	var result domain.DebtExitResult
	var comparison *domain.Comparison

	if input.Strategy == "compare" {
		snowballResult := s.calculateStrategy(input, "snowball")
		avalancheResult := s.calculateStrategy(input, "avalanche")

		if avalancheResult.TotalInterestPaid < snowballResult.TotalInterestPaid {
			result = avalancheResult
		} else {
			result = snowballResult
		}

		comparison = &domain.Comparison{
			Snowball: domain.StrategyResult{
				TotalInterestPaid: snowballResult.TotalInterestPaid,
				MonthsToPayoff:    snowballResult.MonthsToPayoff,
			},
			Avalanche: domain.StrategyResult{
				TotalInterestPaid: avalancheResult.TotalInterestPaid,
				MonthsToPayoff:    avalancheResult.MonthsToPayoff,
			},
		}
		comparison.Savings.InterestSaved = roundTo2Decimals(
			math.Max(0, snowballResult.TotalInterestPaid-avalancheResult.TotalInterestPaid),
		)
		comparison.Savings.MonthsSaved = snowballResult.MonthsToPayoff - avalancheResult.MonthsToPayoff
		result.Comparison = comparison
	} else {
		result = s.calculateStrategy(input, input.Strategy)
	}

	// Generar explicación
	result.Explanation = s.generateDebtExplanation(
		result.Strategy,
		result.TotalDebt,
		result.TotalInterestPaid,
		result.MonthsToPayoff,
		input.Debts,
		result.Comparison,
	)

	return result, nil
}

func (s *DebtExitService) calculateStrategy(
	input domain.DebtExitInput,
	strategy string,
) domain.DebtExitResult {

	// Crear copia de las deudas para trabajar
	debts := make([]domain.Debt, len(input.Debts))
	copy(debts, input.Debts)

	if strategy == "snowball" {
		sort.Slice(debts, func(i, j int) bool {
			return debts[i].Amount < debts[j].Amount
		})
	} else {
		sort.Slice(debts, func(i, j int) bool {
			return debts[i].InterestRate > debts[j].InterestRate
		})
	}

	balances := make(map[string]float64)
	for _, debt := range debts {
		balances[debt.Name] = debt.Amount
	}

	monthlyPlan := []domain.MonthlyPlan{}
	totalInterestPaid := 0.0
	month := 0

	// Simular pagos mes a mes hasta que todas las deudas estén pagadas
	for {
		month++
		available := input.AvailableMonthlyPayment
		payments := []domain.MonthlyPayment{}
		totalPaid := 0.0

		interestMap := make(map[string]float64)
		for _, debt := range debts {
			if balances[debt.Name] <= 0 {
				continue
			}
			// Calcular interés del mes sobre el balance inicial
			monthlyRate := (debt.InterestRate / 100) / 12
			interest := balances[debt.Name] * monthlyRate
			interestMap[debt.Name] = interest
			totalInterestPaid += interest
		}

		// Pagar mínimos (debe cubrir al menos el interés)
		for _, debt := range debts {
			if balances[debt.Name] <= 0 {
				continue
			}

			interest := interestMap[debt.Name]
			// El pago mínimo debe cubrir al menos el interés mensual
			// Si el pago mínimo es menor que el interés, usar el interés como mínimo
			minRequiredPayment := debt.MinimumPayment
			if minRequiredPayment < interest {
				minRequiredPayment = interest
			}

			// Calcular el pago máximo posible (balance + interés)
			maxPossiblePayment := balances[debt.Name] + interest

			// El pago debe ser al menos el mínimo requerido, pero no más del máximo posible
			payment := minRequiredPayment
			if payment > maxPossiblePayment {
				payment = maxPossiblePayment
			}

			if payment > available {
				payment = available
			}

			if payment > 0 {
				// Aplicar pago: primero cubre interés, luego capital
				principalPaid := payment - interest
				if principalPaid < 0 {
					principalPaid = 0
				}
				balances[debt.Name] -= principalPaid
				if balances[debt.Name] < 0 {
					balances[debt.Name] = 0
				}

				payments = append(payments, domain.MonthlyPayment{
					DebtName:         debt.Name,
					Payment:          roundTo2Decimals(payment),
					RemainingBalance: roundTo2Decimals(balances[debt.Name]),
				})

				available -= payment
				totalPaid += payment
			}
		}

		// Aplicar excedente a la primera deuda activa según estrategia
		if available > 0 {
			for _, debt := range debts {
				if balances[debt.Name] > 0 && available > 0 {
					extraPayment := available
					if extraPayment > balances[debt.Name] {
						extraPayment = balances[debt.Name]
					}

					for i := range payments {
						if payments[i].DebtName == debt.Name {
							payments[i].Payment = roundTo2Decimals(payments[i].Payment + extraPayment)
							balances[debt.Name] -= extraPayment
							if balances[debt.Name] < 0 {
								balances[debt.Name] = 0
							}
							payments[i].RemainingBalance = roundTo2Decimals(balances[debt.Name])
							totalPaid += extraPayment
							available -= extraPayment
							break
						}
					}
					break
				}
			}
		}

		monthlyPlan = append(monthlyPlan, domain.MonthlyPlan{
			Month:     month,
			Payments:  payments,
			TotalPaid: roundTo2Decimals(totalPaid),
		})

		// Verificar si todas las deudas están pagadas
		allPaid := true
		for _, debt := range debts {
			if balances[debt.Name] > DebtBalanceTolerance {
				allPaid = false
				break
			}
		}

		if allPaid {
			break
		}

		// Límite de seguridad para evitar loops infinitos
		if month > MaxDebtPayoffMonths {
			log.Printf("Warning: debt payoff calculation reached maximum months limit (%d)", MaxDebtPayoffMonths)
			break
		}
	}

	totalDebt := 0.0
	for _, debt := range input.Debts {
		totalDebt += debt.Amount
	}

	return domain.DebtExitResult{
		Strategy:          strategy,
		TotalDebt:         roundTo2Decimals(totalDebt),
		TotalInterestPaid: roundTo2Decimals(totalInterestPaid),
		MonthsToPayoff:    month,
		MonthlyPlan:       monthlyPlan,
	}
}

func (s *DebtExitService) generateDebtExplanation(
	strategy string,
	totalDebt, totalInterest float64,
	months int,
	debts []domain.Debt,
	comparison *domain.Comparison,
) string {
	strategyName := "Snowball (Bola de Nieve)"
	strategyTip := "Ideal si necesitas ver resultados rápidos para mantenerte motivado. Cada deuda pagada libera capital que puedes aplicar a la siguiente."
	if strategy == "avalanche" {
		strategyName = "Avalanche (Avalancha)"
		strategyTip = "Ideal si tu objetivo principal es minimizar el costo financiero total. Requiere disciplina pero maximiza el ahorro."
	}

	totalCost := totalDebt + totalInterest
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("La estrategia %s te permitirá liquidar todas tus deudas en %d meses (%.1f años). ", strategyName, months, float64(months)/12.0))
	builder.WriteString(fmt.Sprintf("Tu deuda inicial es %s y pagarás %s en intereses, para un costo total de %s. ",
		formatCurrency(totalDebt), formatCurrency(totalInterest), formatCurrency(totalCost)))

	// Orden de pago
	sortedDebts := make([]domain.Debt, len(debts))
	copy(sortedDebts, debts)
	if strategy == "snowball" {
		sort.Slice(sortedDebts, func(i, j int) bool {
			return sortedDebts[i].Amount < sortedDebts[j].Amount
		})
	} else {
		sort.Slice(sortedDebts, func(i, j int) bool {
			return sortedDebts[i].InterestRate > sortedDebts[j].InterestRate
		})
	}

	builder.WriteString(fmt.Sprintf("\n\nCon %s, el orden de pago es:\n", strategyName))
	for i, debt := range sortedDebts {
		builder.WriteString(fmt.Sprintf("%d. %s: %s (%.2f%% anual)\n",
			i+1, debt.Name, formatCurrency(debt.Amount), debt.InterestRate))
	}

	// Comparación si existe
	if comparison != nil {
		builder.WriteString(s.buildComparisonText(strategy, comparison, months))
	}

	builder.WriteString(fmt.Sprintf("\n\nRecomendación: %s", strategyTip))

	return builder.String()
}

func (s *DebtExitService) buildComparisonText(strategy string, comparison *domain.Comparison, monthsToPayoff int) string {
	monthsDiff := comparison.Savings.MonthsSaved
	interestSaved := comparison.Savings.InterestSaved
	var text string

	if strategy == "snowball" {
		if interestSaved > 0 {
			if monthsDiff > 0 {
				text = fmt.Sprintf("\n\nComparado con Avalanche, pagarás %s más en intereses y tomará %d meses más, pero ofrece mayor motivación psicológica.",
					formatCurrency(interestSaved), monthsDiff)
			} else if monthsDiff < 0 {
				text = fmt.Sprintf("\n\nComparado con Avalanche, terminarás %d meses antes pero pagarás %s más en intereses. Esto ocurre porque pagar deudas pequeñas primero libera capital más rápido, aunque puede resultar en un costo total mayor.",
					-monthsDiff, formatCurrency(interestSaved))
			} else {
				text = fmt.Sprintf("\n\nComparado con Avalanche, pagarás %s más en intereses en el mismo tiempo, pero con mayor motivación psicológica.",
					formatCurrency(interestSaved))
			}
		} else {
			if monthsDiff > 0 {
				text = fmt.Sprintf("\n\nComparado con Avalanche, pagarás los mismos intereses pero tomará %d meses más, aunque ofrece mayor motivación psicológica.",
					monthsDiff)
			} else if monthsDiff < 0 {
				text = fmt.Sprintf("\n\nComparado con Avalanche, pagarás los mismos intereses y terminarás %d meses antes, combinando motivación con eficiencia temporal.",
					-monthsDiff)
			}
		}
	} else {
		if interestSaved > 0 {
			if monthsDiff > 0 {
				text = fmt.Sprintf("\n\nComparado con Snowball, ahorrarás %s en intereses y terminarás %d meses antes, minimizando tu costo financiero total.",
					formatCurrency(interestSaved), monthsDiff)
			} else if monthsDiff < 0 {
				text = fmt.Sprintf("\n\nComparado con Snowball, ahorrarás %s en intereses aunque tomará %d meses más. Esto ocurre porque priorizar deudas con mayor interés minimiza el costo total, aunque puede tomar más tiempo.",
					formatCurrency(interestSaved), -monthsDiff)
			} else {
				text = fmt.Sprintf("\n\nComparado con Snowball, ahorrarás %s en intereses en el mismo tiempo, optimizando el costo financiero.",
					formatCurrency(interestSaved))
			}
		} else {
			if monthsDiff > 0 {
				text = fmt.Sprintf("\n\nComparado con Snowball, pagarás los mismos intereses y terminarás %d meses antes, minimizando el tiempo total.",
					monthsDiff)
			} else if monthsDiff < 0 {
				text = fmt.Sprintf("\n\nComparado con Snowball, pagarás los mismos intereses pero tomará %d meses más, aunque minimiza el costo financiero.",
					-monthsDiff)
			}
		}
	}

	return text
}
