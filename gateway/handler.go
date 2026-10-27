package gateway

import (
	"encoding/json"
	"net/http"
)

// Define the payload structure matching WhatsApp Cloud API incoming webhook
type WhatsAppPayload struct {
	MessageID   string `json:"message_id"`
	From        string `json:"from"`
	TextBody    string `json:"text_body"`
	MessageType string `json:"message_type"`
}

type Handler struct {
	queue chan<- WhatsAppPayload // Write-only channel constraint
}

func NewHandler(q chan<- WhatsAppPayload) *Handler {
	return &Handler{queue: q}
}

// ServeHTTP implements the standard http.Handler interface
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload WhatsAppPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "Invalid payload JSON", http.StatusBadRequest)
		return
	}

	// Fail-fast validation (similar to Request validation in Laravel)
	if payload.MessageID == "" || payload.From == "" {
		http.Error(w, "Missing required fields", http.StatusUnprocessableEntity)
		return
	}

	// Non-blocking write to channel. If the queue is full, it drops to default
	select {
	case h.queue <- payload:
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"queued"}`))
	default:
		// Queue full situation. Real production would fallback to an external broker or log aggressively
		http.Error(w, "System overloaded, try again later", http.StatusServiceUnavailable)
	}
}
