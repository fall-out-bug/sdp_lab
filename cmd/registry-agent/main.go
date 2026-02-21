// registry-agent provides HTTP CRUD for the project registry and publishes NATS events.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"sdp_dev/internal/bus"
	"sdp_dev/internal/registry"
)

func main() {
	regPath := flag.String("registry", "specs/project-registry.yaml", "path to project-registry.yaml")
	natsURL := flag.String("nats", os.Getenv("NATS_URL"), "NATS server URL")
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	store := registry.NewStore(registry.StoreConfig{RegistryPath: *regPath})
	if err := store.Load(); err != nil {
		log.Fatalf("load registry: %v", err)
	}

	var b bus.Bus
	if *natsURL != "" {
		var err error
		b, err = bus.ConnectAndProvision(context.Background(), *natsURL)
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

	log.Printf("registry-agent listening on %s", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("serve: %v", err)
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
		var p registry.Project
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
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
		id := r.PathValue("id")
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
