
package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"whatsapp-gateway/gateway"
)

func BenchmarkGatewayLoad(b *testing.B) {
	// 1. Setup the infrastructure inside the test
	payloadQueue := make(chan gateway.WhatsAppPayload, 1000)

	// Start background workers
	for i := 1; i <= 10; i++ {
		go gateway.StartWorker(i, payloadQueue)
	}

	appHandler := gateway.NewHandler(payloadQueue)
	server := httptest.NewServer(appHandler)
	defer server.Close()

	payloadData := []byte(`{"message_id": "msg_987654321","from": "628123456789","text_body": "Halo","message_type": "text"}`)

	b.ResetTimer()

	// 2. Flood it concurrently
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 50) // Max 50 simultaneous connections

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		semaphore <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-semaphore }()

			resp, err := http.Post(server.URL, "application/json", bytes.NewBuffer(payloadData))
			if err == nil {
				resp.Body.Close()
			}
		}()
	}

	wg.Wait()
}
