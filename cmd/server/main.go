package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/YashRaJShaRMaCODE/sourceasia-backend-assignment/internal/handlers"
	"github.com/YashRaJShaRMaCODE/sourceasia-backend-assignment/internal/middleware"
	"github.com/YashRaJShaRMaCODE/sourceasia-backend-assignment/internal/services"
	"github.com/YashRaJShaRMaCODE/sourceasia-backend-assignment/internal/store"
)

func main() {
	// Initialize Store
	memStore := store.NewInMemoryStore()

	// Initialize Services
	rateLimitService := services.NewRateLimitService(memStore)
	productService := services.NewProductService(memStore)

	// Initialize Handlers
	h := handlers.NewHandler(rateLimitService, productService)

	// Setup Router
	mux := http.NewServeMux()
	withLogging := middleware.Logging

	mux.HandleFunc("/health", h.HealthCheck)
	mux.HandleFunc("/request", withLogging(h.HandleRequest))
	mux.HandleFunc("/stats", withLogging(h.HandleStats))
	mux.HandleFunc("/products", withLogging(h.HandleProducts))
	mux.HandleFunc("/products/", withLogging(h.HandleProductByID))

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Graceful Shutdown
	go func() {
		log.Printf("Server starting on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
