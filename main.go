package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
	defer rateLimiter.Stop()

	mux := http.NewServeMux()
	mux.Handle(
		"/loan/calculate",
		httpLayer.RateLimitMiddleware(
			rateLimiter,
			http.HandlerFunc(loanHandler.CalculateLoan),
		),
	)

	mux.Handle(
		"/loan/recommend-term",
		httpLayer.RateLimitMiddleware(
			rateLimiter,
			http.HandlerFunc(termRecommendationHandler.RecommendTerm),
		),
	)

	mux.Handle(
		"/loan/debt-exit-plan",
		httpLayer.RateLimitMiddleware(
			rateLimiter,
			http.HandlerFunc(debtExitHandler.CalculateDebtExitPlan),
		),
	)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Manejar shutdown graceful
	go func() {
		log.Println("🚀 API corriendo en http://localhost:8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
