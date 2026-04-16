package testdata

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func chiRoutes() chi.Router {
	r := chi.NewRouter()
	r.Use(middlewareLogger)

	r.Get("/users", listUsers)
	r.Post("/users", createUser)
	r.Get("/users/{id}", getUser)
	r.Put("/users/{id}", updateUser)
	r.Delete("/users/{id}", deleteUser)

	r.Route("/admin", func(r chi.Router) {
		r.Get("/settings", getSettings)
		r.Post("/settings", updateSettings)
	})

	return r
}

func listUsers(w http.ResponseWriter, r *http.Request)    {}
func createUser(w http.ResponseWriter, r *http.Request)   {}
func getUser(w http.ResponseWriter, r *http.Request)      {}
func updateUser(w http.ResponseWriter, r *http.Request)   {}
func deleteUser(w http.ResponseWriter, r *http.Request)   {}
func getSettings(w http.ResponseWriter, r *http.Request)  {}
func updateSettings(w http.ResponseWriter, r *http.Request) {}
func middlewareLogger(next http.Handler) http.Handler      { return next }
