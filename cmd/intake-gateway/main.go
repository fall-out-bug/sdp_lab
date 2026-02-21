// intake-gateway provides HTTP API for task intake and publishes to NATS.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"sdp_dev/internal/bus"
	"sdp_dev/internal/intake"
	"sdp_dev/internal/registry"
)

func main() {
	natsURL := flag.String("nats", os.Getenv("NATS_URL"), "NATS server URL")
	addr := flag.String("addr", ":8081", "HTTP listen address")
	flag.Parse()

	var b bus.Bus
	if *natsURL != "" {
		ctx := context.Background()
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

	log.Printf("intake-gateway listening on %s", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func handleIntake(b bus.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
