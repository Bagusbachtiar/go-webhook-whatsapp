package gateway

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StartWorker listens to the channel and writes incoming payloads directly to Neon Postgres
func StartWorker(id int, queue <-chan WhatsAppPayload, dbPool *pgxpool.Pool) {
	log.Printf("[Worker %d] Initialized and waiting for database jobs...", id)

	for payload := range queue {
		log.Printf("[Worker %d] Processing message %s from %s", id, payload.MessageID, payload.From)

		// SQL Query to insert the WhatsApp payload safely into your cloud database table
		query := `
			INSERT INTO whatsapp_messages (message_id, sender, text_body, message_type)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (message_id) DO NOTHING;
		`

		// Execute the insert asynchronously using your high-speed cloud connection pool
		_, err := dbPool.Exec(context.Background(), query, payload.MessageID, payload.From, payload.TextBody, payload.MessageType)
		if err != nil {
			log.Printf("[Worker %d] ERROR saving message %s: %v", id, payload.MessageID, err)
			continue
		}

		log.Printf("[Worker %d] Successfully saved message %s to Neon Postgres!", id, payload.MessageID)
	}
}
