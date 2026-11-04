# Go Enterprise WhatsApp Triage & Support Gateway Proxy

A high-throughput, concurrent microservice built natively with Go to ingest, cryptographically verify, and intelligently triage massive volumes of inbound customer support webhooks from the Meta Cloud API. By decoupling payload ingestion from business routing, this gateway ensures ultra-low latency execution under heavy traffic spikes.

---

### 🛠️ Production Architecture

*   **Front Ingestion Layer:** Built using Go's standard `net/http` package to handle high-concurrency requests and echo Meta Developer verification handshakes (`GET`) while processing message ingestions (`POST`).
*   **Decoupled Async Processing:** Utilizes internal buffered channels (`chan`) to instantly ingest traffic loops and safeguard the application from network spikes without external brokers like Redis.
*   **Parallel Execution Infrastructure:** Employs concurrent `goroutines` running worker-pool logic across all available virtual CPU cores to distribute heavy third-party interactions parallelly.
*   **Data Layer and Resilience:** Integrated with standard `pgxpool` cluster connections utilizing `.env` variable parsing strategies to commit message payloads to cloud **Neon Serverless Postgres** structures asynchronously.
*   **Outbound Communication Module:** Integrates a native Go HTTP client loop inside the thread processing loop to transmit real-time text responses back to the user via the Meta Graph API boundaries.
*   **Cryptographic Security Middleware:** Enforces strict validation on incoming webhooks by extracting the payload signature header (`X-Hub-Signature-256`) sent from Meta. The middleware reads the raw request byte stream, computes a local verification signature using an application secret key via the HMAC-SHA256 algorithm, and performs a time-constant comparison checking routine (`subtle.ConstantTimeCompare`) to securely drop unauthenticated traffic before it can consume internal queue memory or access database resources.
*   **Intelligent Triage & Traffic Routing:** Features a thread-safe classification engine inside the background worker pool that parses message intent signatures. It automatically channels requests to specialized downstream streams (e.g., `FINANCE_DEPT`, `CRITICAL_ENGINEERING`, `GENERAL_AI_TIER`) and triggers dynamic contextual workflows.

---

### 🏎️ Performance Benchmark Results

To scientifically evaluate the structural scaling limits of this architecture, a native high-concurrency stress test was executed utilizing the built-in Go testing engine running on an Intel Core i5-12400F environment (12 available virtual CPU threads).

The gateway was flooded with a burst of **10,000 total incoming JSON payload webhooks** over **50 simultaneous network channels**:

*   **Total Processing Time:** `1.268 seconds` (to completely ingest all 10,000 requests)
*   **Average Ingestion Latency:** `43,099 ns/op` (~`0.043 milliseconds` per individual request)
*   **Ingestion Throughput Rate:** `~7,886 requests per second (RPS)`
