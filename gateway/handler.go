package gateway

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

// Meta's real deeply-nested JSON structure
type MetaWebhookPayload struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Changes []struct {
			Field string `json:"field"`
			Value struct {
				MessagingProduct string `json:"messaging_product"`
				Metadata         struct {
					DisplayPhoneNumber string `json:"display_phone_number"`
					PhoneNumberID      string `json:"phone_number_id"`
				} `json:"metadata"`
				Contacts []struct {
					Profile struct {
						Name string `json:"name"`
					} `json:"profile"`
					WaID string `json:"wa_id"`
				} `json:"contacts"`
				Messages []struct {
					From      string `json:"from"`
					ID        string `json:"id"`
					Timestamp string `json:"timestamp"`
					Type      string `json:"type"`
					Text      struct {
						Body string `json:"body"`
					} `json:"text"`
				} `json:"messages"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

// This is the clean, flattened struct we drop down our internal channel queue
type WhatsAppPayload struct {
	MessageID   string
	From        string
	TextBody    string
	MessageType string
}

type Handler struct {
	queue chan<- WhatsAppPayload
}

func NewHandler(q chan<- WhatsAppPayload) *Handler {
	return &Handler{queue: q}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. HANDLE META VERIFICATION HANDSHAKE (GET REQUEST)
	if r.Method == http.MethodGet {
		query := r.URL.Query()
		mode := query.Get("hub.mode")
		token := query.Get("hub.verify_token")
		challenge := query.Get("hub.challenge")

		expectedToken := os.Getenv("WHATSAPP_VERIFY_TOKEN")
		if expectedToken == "" {
			expectedToken = "my_secret_token_123"
		}

		if mode == "subscribe" && token == expectedToken {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(challenge))
			return
		}

		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// 2. HANDLE REAL LIVE MESSAGES (POST REQUEST)
	if r.Method == http.MethodPost {
		var metaPayload MetaWebhookPayload
		err := json.NewDecoder(r.Body).Decode(&metaPayload)
		if err != nil {
			http.Error(w, "Invalid payload JSON", http.StatusBadRequest)
			return
		}

		// Meta sometimes sends status updates (sent, delivered, read) instead of actual messages.
		// We ensure we only process if there is a real text message present.
		if len(metaPayload.Entry) == 0 || 
		   len(metaPayload.Entry[0].Changes) == 0 || 
		   len(metaPayload.Entry[0].Changes[0].Value.Messages) == 0 {
			// Acknowledge receipt to Meta for status updates but skip queueing
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ignored_status_update"}`))
			return
		}

		// Extract Meta's deeply nested structures into our flat internal struct
		rawMessage := metaPayload.Entry[0].Changes[0].Value.Messages[0]
		
		flattenedPayload := WhatsAppPayload{
			MessageID:   rawMessage.ID,
			From:        rawMessage.From,
			TextBody:    rawMessage.Text.Body,
			MessageType: rawMessage.Type,
		}

		// Fail-safe check: only drop into the channel queue if it's a text message
		if flattenedPayload.MessageType != "text" || flattenedPayload.MessageID == "" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"non_text_ignored"}`))
			return
		}

		// Push to in-memory channel queue asynchronously
		select {
		case h.queue <- flattenedPayload:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"queued"}`))
		default:
			log.Println("[Gateway ERROR]: Internal channel buffer queue full!")
			http.Error(w, "System overloaded", http.StatusServiceUnavailable)
		}
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
