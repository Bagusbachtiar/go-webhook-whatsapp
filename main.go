package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"whatsapp-gateway/gateway"
)

func main() {
	// 1. Create a buffered channel for incoming WhatsApp payloads
	// The struct name inside gateway/handler.go is WhatsAppPayload
	payloadQueue := make(chan gateway.WhatsAppPayload, 1000)

	// 2. Start a worker pool of 10 concurrent goroutines
	log.Println("Starting background worker pool...") // Fixed: Println (Capital P)
	for i := 1; i <= 10; i++ {
		go gateway.StartWorker(i, payloadQueue)
	}

	// 3. Initialize the HTTP handler
	appHandler := gateway.NewHandler(payloadQueue)

	server := &http.Server{
		Addr:    ":8080",
		Handler: appHandler,
	}

	// 4. Handle Graceful Shutdown
	go func() {
		log.Println("Gateway API running on port :8080") // Fixed: Println (Capital P)
		// Fixed: ListenAndServe (not ListenAndServer)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Wait for termination signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down gateway gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Close the channel to let workers finish remaining jobs cleanly
	close(payloadQueue)
	log.Println("Gateway stopped cleanly.")
}
