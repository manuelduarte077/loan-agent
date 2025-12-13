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

	rateLimiter := httpLayer.NewRateLimiter(5, time.Minute)

	http.Handle(
		"/loan/calculate",
		httpLayer.RateLimitMiddleware(
			rateLimiter,
			http.HandlerFunc(loanHandler.CalculateLoan),
		),
	)

	log.Println("🚀 API corriendo en http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
