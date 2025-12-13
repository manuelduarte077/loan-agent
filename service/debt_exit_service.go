package service

import (
	"errors"
	"math"
	"sort"

	"loan-agent/domain"
)

type DebtExitService struct {
	loanService *LoanService
}

func NewDebtExitService(loanService *LoanService) *DebtExitService {
	return &DebtExitService{loanService: loanService}
}

// CalculateDebtExitPlan calcula el plan de salida de deudas usando snowball o avalanche
func (s *DebtExitService) CalculateDebtExitPlan(
	input domain.DebtExitInput,
) (domain.DebtExitResult, error) {

	// Validaciones
	if len(input.Debts) == 0 {
		return domain.DebtExitResult{}, errors.New("no se proporcionaron deudas")
	}
	if input.AvailableMonthlyPayment <= 0 {
		return domain.DebtExitResult{}, errors.New("pago mensual disponible inválido")
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
		if debt.InterestRate < 0 {
			return domain.DebtExitResult{}, errors.New("tasa de interés inválida")
		}
		if debt.MinimumPayment <= 0 {
			return domain.DebtExitResult{}, errors.New("pago mínimo inválido")
		}
		totalMinimumPayments += debt.MinimumPayment
	}

	if totalMinimumPayments > input.AvailableMonthlyPayment {
		return domain.DebtExitResult{}, errors.New("el pago mensual disponible es insuficiente para cubrir los pagos mínimos")
	}

	var result domain.DebtExitResult
	var comparison *domain.Comparison

	if input.Strategy == "compare" {
		// Calcular ambos métodos y comparar
		snowballResult := s.calculateStrategy(input, "snowball")
		avalancheResult := s.calculateStrategy(input, "avalanche")

		// Usar el mejor método como resultado principal
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

	return result, nil
}

func (s *DebtExitService) calculateStrategy(
	input domain.DebtExitInput,
	strategy string,
) domain.DebtExitResult {

	// Crear copia de las deudas para trabajar
	debts := make([]domain.Debt, len(input.Debts))
	copy(debts, input.Debts)

	// Ordenar según estrategia
	if strategy == "snowball" {
		// Ordenar por monto ascendente
		sort.Slice(debts, func(i, j int) bool {
			return debts[i].Amount < debts[j].Amount
		})
	} else {
		// Ordenar por tasa de interés descendente
		sort.Slice(debts, func(i, j int) bool {
			return debts[i].InterestRate > debts[j].InterestRate
		})
	}

	// Crear mapa de balances restantes
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

		// Primera pasada: calcular intereses y pagar mínimos de todas las deudas activas
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
			// El pago mínimo debe ser al menos el interés más algo de capital
			minRequiredPayment := interest
			if debt.MinimumPayment > minRequiredPayment {
				minRequiredPayment = debt.MinimumPayment
			}

			payment := minRequiredPayment
			if payment > balances[debt.Name]+interest {
				payment = balances[debt.Name] + interest
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
					DebtName:        debt.Name,
					Payment:         roundTo2Decimals(payment),
					RemainingBalance: roundTo2Decimals(balances[debt.Name]),
				})

				available -= payment
				totalPaid += payment
			}
		}

		// Segunda pasada: aplicar excedente a la primera deuda activa según estrategia
		if available > 0 {
			for _, debt := range debts {
				if balances[debt.Name] > 0 && available > 0 {
					extraPayment := available
					if extraPayment > balances[debt.Name] {
						extraPayment = balances[debt.Name]
					}

					// Actualizar el pago existente
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
			if balances[debt.Name] > 0.01 { // Tolerancia para errores de redondeo
				allPaid = false
				break
			}
		}

		if allPaid {
			break
		}

		// Límite de seguridad para evitar loops infinitos
		if month > 600 { // 50 años máximo
			break
		}
	}

	// Calcular total de deuda inicial
	totalDebt := 0.0
	for _, debt := range input.Debts {
		totalDebt += debt.Amount
	}

	return domain.DebtExitResult{
		Strategy:         strategy,
		TotalDebt:        roundTo2Decimals(totalDebt),
		TotalInterestPaid: roundTo2Decimals(totalInterestPaid),
		MonthsToPayoff:   month,
		MonthlyPlan:      monthlyPlan,
	}
}

