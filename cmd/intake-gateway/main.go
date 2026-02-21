// intake-gateway provides HTTP API for task intake and publishes to NATS.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"sdp_dev/internal/bus"
	"sdp_dev/internal/intake"
	"sdp_dev/internal/registry"
)

const maxBodyBytes = 64 * 1024 // 64KB

func main() {
	natsURL := flag.String("nats", os.Getenv("NATS_URL"), "NATS server URL")
	addr := flag.String("addr", ":8081", "HTTP listen address")
	apiKey := flag.String("api-key", os.Getenv("INTAKE_API_KEY"), "Optional API key for auth (Bearer or X-API-Key)")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("shutdown signal received")
		cancel()
	}()

	var b bus.Bus
	if *natsURL != "" {
		var err error
		b, err = bus.ConnectAndProvision(ctx, *natsURL)
		if err != nil {
			log.Fatalf("NATS: %v", err)
		}
		defer b.Close()
	}

	store := registry.NewStore(registry.StoreConfig{})
	_ = store.Load()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/intake", handleIntake(b))
	mux.HandleFunc("GET /api/v1/projects", handleProjects(store))
	mux.HandleFunc("GET /api/v1/status/{id}", handleStatus)
	mux.HandleFunc("GET /api/v1/stream", handleStream(b))

	var h http.Handler = mux
	if *apiKey != "" {
		h = apiKeyAuth(*apiKey)(mux)
	}

	srv := &http.Server{Addr: *addr, Handler: h, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("intake-gateway listening on %s", *addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func handleIntake(b bus.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var req intake.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if req.Title == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title required"})
			return
		}
		if err := intake.Normalize(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := intake.ValidateProjectID(req.ProjectID); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		taskID := "task-" + time.Now().Format("20060102150405")
		if b != nil {
			subject := "sdp.intake." + req.ProjectID
			payload, _ := json.Marshal(req)
			env := bus.Envelope{
				IssueID:       taskID,
				ArtifactID:    "intake",
				ArtifactClass: "intake",
				Phase:         "created",
				Role:          "gateway",
				Payload:       payload,
				ProjectID:     req.ProjectID,
			}
			if err := b.Publish(subject, env); err != nil {
				log.Printf("publish intake: %v", err)
			}
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"id":         taskID,
			"project_id": req.ProjectID,
			"status":     "queued",
		})
	}
}

func handleProjects(store *registry.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projects := store.List()
		writeJSON(w, http.StatusOK, projects)
	}
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	writeJSON(w, http.StatusOK, map[string]any{
		"id":     id,
		"status": "unknown",
	})
}

func handleStream(b bus.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Placeholder: would upgrade to SSE and subscribe to sdp.lifecycle.>
		_, _ = w.Write([]byte("data: {\"status\":\"connected\"}\n\n"))
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// apiKeyAuth returns middleware that requires Authorization: Bearer <key> or X-API-Key: <key> when apiKey is non-empty.
func apiKeyAuth(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-API-Key")
			if key == "" {
				if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
					key = strings.TrimPrefix(auth, "Bearer ")
				}
			}
			if key != apiKey {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
