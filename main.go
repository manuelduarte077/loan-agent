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

	serverErr := make(chan error, 1)
	go func() {
		log.Println("🚀 API corriendo en http://localhost:8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Printf("Error starting server: %v", err)
		return
	case <-quit:
		log.Println("Shutting down server...")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Error during server shutdown: %v", err)
	}

	log.Println("Server exited")
}
