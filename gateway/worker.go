package gateway

import (
	"log"
	"time"
)

// StartWorker listens to the payload channel and processes items concurrently
func StartWorker(id int, queue <-chan WhatsAppPayload) { // Read-only channel constraint
	log.Printf("[Worker %d] Initialized and waiting for messages...", id)

	// The loop naturally terminates when the channel is closed in main.go
	for payload := range queue {
		log.Printf("[Worker %d] Processing message %s from %s", id, payload.MessageID, payload.From)

		// Simulate intensive processing (e.g., Calling an LLM, RAG parsing, writing to PG)
		time.Sleep(500 * time.Millisecond) 

		log.Printf("[Worker %d] Successfully finished processing %s", id, payload.MessageID)
	}
}
