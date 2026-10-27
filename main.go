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

	"github.com/joho/godotenv"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// 1. Load the local .env configuration file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: No .env file found, reading from system env vars")
	}

	// 2. Fetch the connection string safely
	dbConnectionString := os.Getenv("DATABASE_URL")
	if dbConnectionString == "" {
		log.Fatalf("DATABASE_URL is missing from environment variables")
	}

	// Initialize high-performance Neon connection pool
	dbPool, err := pgxpool.New(context.Background(), dbConnectionString)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer dbPool.Close()

	// Verify cloud database connection is alive
	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}
	log.Println("Successfully connected to the Neon Serverless Postgres Pool!")

	// 3. Setup internal queue channel
	payloadQueue := make(chan gateway.WhatsAppPayload, 1000)

	// 4. Start 10 database queue workers
	log.Println("Starting cloud database worker pool...")
	for i := 1; i <= 10; i++ {
		go gateway.StartWorker(i, payloadQueue, dbPool)
	}

	// 5. Initialize HTTP Handler
	appHandler := gateway.NewHandler(payloadQueue)
	server := &http.Server{
		Addr:    ":8080",
		Handler: appHandler,
	}

	go func() {
		log.Println("Gateway API running on port :8080")
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Graceful Shutdown Sequence
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down gateway gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	close(payloadQueue)
	log.Println("Gateway stopped cleanly.")
}
