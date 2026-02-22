// registry-agent provides HTTP CRUD for the project registry and publishes NATS events.
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
	regPath := flag.String("registry", "specs/project-registry.yaml", "path to project-registry.yaml")
	natsURL := flag.String("nats", os.Getenv("NATS_URL"), "NATS server URL")
	addr := flag.String("addr", ":8080", "HTTP listen address")
	apiKey := flag.String("api-key", os.Getenv("REGISTRY_API_KEY"), "Optional API key for auth (Bearer or X-API-Key)")
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

	store := registry.NewStore(registry.StoreConfig{RegistryPath: *regPath})
	if err := store.Load(); err != nil {
		log.Fatalf("load registry: %v", err)
	}

	var b bus.Bus
	if *natsURL != "" {
		var err error
		b, err = bus.ConnectAndProvision(ctx, *natsURL)
		if err != nil {
			log.Printf("NATS connect failed: %v (continuing without NATS)", err)
		} else {
			defer b.Close()
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/projects", handleList(store))
	mux.HandleFunc("GET /api/v1/projects/{id}", handleGet(store))
	mux.HandleFunc("POST /api/v1/projects", handleCreate(store, b))
	mux.HandleFunc("PUT /api/v1/projects/{id}", handleUpdate(store, b))
	mux.HandleFunc("DELETE /api/v1/projects/{id}", handleDelete(store, b))

	var h http.Handler = mux
	if *apiKey != "" {
		h = apiKeyAuth(*apiKey)(mux)
	}

	srv := &http.Server{Addr: *addr, Handler: h, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("registry-agent listening on %s", *addr)
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

func handleList(store *registry.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projects := store.List()
		writeJSON(w, http.StatusOK, projects)
	}
}

func handleGet(store *registry.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if err := intake.ValidateProjectID(id); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		p, ok := store.Get(id)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func handleCreate(store *registry.Store, b bus.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var p registry.Project
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := intake.ValidateProjectID(p.ID); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := store.Create(&p); err != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		if err := store.Save(); err != nil {
			log.Printf("save after create: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to persist"})
			return
		}
		publishNATS(b, "sdp.registry.project.created", p.ID)
		writeJSON(w, http.StatusCreated, p)
	}
}

func handleUpdate(store *registry.Store, b bus.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		id := r.PathValue("id")
		if err := intake.ValidateProjectID(id); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		var p registry.Project
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		p.ID = id
		if err := store.Update(&p); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		if err := store.Save(); err != nil {
			log.Printf("save after update: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to persist"})
			return
		}
		publishNATS(b, "sdp.registry.project.updated", p.ID)
		writeJSON(w, http.StatusOK, p)
	}
}

func handleDelete(store *registry.Store, b bus.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if err := intake.ValidateProjectID(id); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := store.Delete(id); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		if err := store.Save(); err != nil {
			log.Printf("save after delete: %v", err)
		}
		publishNATS(b, "sdp.registry.project.deleted", id)
		w.WriteHeader(http.StatusNoContent)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func publishNATS(b bus.Bus, subject, projectID string) {
	if b == nil {
		return
	}
	payload, _ := json.Marshal(map[string]string{"project_id": projectID})
	env := bus.Envelope{
		IssueID:       projectID,
		ArtifactID:    "registry-event",
		ArtifactClass: "registry",
		Phase:         subject,
		Role:          "registry-agent",
		Payload:       payload,
	}
	if err := b.Publish(subject, env); err != nil {
		log.Printf("NATS publish %s: %v", subject, err)
	}
}

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
