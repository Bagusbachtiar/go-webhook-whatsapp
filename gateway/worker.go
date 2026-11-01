package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

type WhatsAppTextRequest struct {
	MessagingProduct string `json:"messaging_product"`
	To               string `json:"to"`
	Type             string `json:"type"`
	Text             struct {
		Body string `json:"body"`
	} `json:"text"`
}

func StartWorker(id int, queue <-chan WhatsAppPayload, dbPool *pgxpool.Pool) {
	log.Printf("[Worker %d] Initialized and waiting for database jobs...", id)

	for payload := range queue {
		log.Printf("[Worker %d] Processing message %s from %s", id, payload.MessageID, payload.From)

		query := `
			INSERT INTO whatsapp_messages (message_id, sender, text_body, message_type)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (message_id) DO NOTHING;
		`
		_, err := dbPool.Exec(context.Background(), query, payload.MessageID, payload.From, payload.TextBody, payload.MessageType)
		if err != nil {
			log.Printf("[Worker %d] ERROR saving message to Postgres: %v", id, err)
			continue
		}
		log.Printf("[Worker %d] Successfully saved message %s to Neon Postgres!", id, payload.MessageID)

		go sendWhatsAppReply(payload.From, fmt.Sprintf("Hi! This is an automated response from my Go Microservice worker pool. I received your message: \"%s\"", payload.TextBody))
	}
}

func sendWhatsAppReply(recipientMobile string, textMessageBody string) {
	accessToken := os.Getenv("WHATSAPP_ACCESS_TOKEN")
	phoneNumberID := os.Getenv("WHATSAPP_PHONE_NUMBER_ID")

	if accessToken == "" || phoneNumberID == "" {
		log.Println("[Outbound skipped]: Missing WHATSAPP_ACCESS_TOKEN or WHATSAPP_PHONE_NUMBER_ID in environment setup")
		return
	}

	requestPayload := WhatsAppTextRequest{
		MessagingProduct: "whatsapp",
		To:               recipientMobile,
		Type:             "text",
	}
	requestPayload.Text.Body = textMessageBody

	jsonData, err := json.Marshal(requestPayload)
	if err != nil {
		log.Printf("[Outbound Error]: Failed to marshal JSON payload: %v", err)
		return
	}

	metaAPIUrl := fmt.Sprintf("https://graph.facebook.com/v18.0/%s/messages", phoneNumberID)

	req, err := http.NewRequest("POST", metaAPIUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[Outbound Error]: Failed to form HTTP request payload: %v", err)
		return
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Outbound Error]: Network layer failed targeting Meta API endpoints: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		log.Printf("[Outbound Success]: Text reply transmitted successfully to %s", recipientMobile)
	} else {
		log.Printf("[Outbound Failure]: Meta API responded with status indicator code: %d", resp.StatusCode)
	}
}
