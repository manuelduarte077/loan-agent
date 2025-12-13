package main

import (
	"log"
	"net/http"
	"time"

	httpLayer "loan-agent/http"
	"loan-agent/repository"
	"loan-agent/service"
)

func main() {

	loanRepo := repository.NewLoanRepositoryMemory()

	// cache := repository.NewRedisCache("localhost:6379")
	cache := repository.NewMockCache()

	loanService := service.NewLoanService(loanRepo, cache)

	loanHandler := httpLayer.NewLoanHandler(loanService)

	termRecommendationService := service.NewTermRecommendationService(loanService)
	termRecommendationHandler := httpLayer.NewTermRecommendationHandler(termRecommendationService)

	debtExitService := service.NewDebtExitService(loanService)
	debtExitHandler := httpLayer.NewDebtExitHandler(debtExitService)

	rateLimiter := httpLayer.NewRateLimiter(5, time.Minute)

	http.Handle(
		"/loan/calculate",
		httpLayer.RateLimitMiddleware(
			rateLimiter,
			http.HandlerFunc(loanHandler.CalculateLoan),
		),
	)

	http.Handle(
		"/loan/recommend-term",
		httpLayer.RateLimitMiddleware(
			rateLimiter,
			http.HandlerFunc(termRecommendationHandler.RecommendTerm),
		),
	)

	http.Handle(
		"/loan/debt-exit-plan",
		httpLayer.RateLimitMiddleware(
			rateLimiter,
			http.HandlerFunc(debtExitHandler.CalculateDebtExitPlan),
		),
	)

	log.Println("🚀 API corriendo en http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
