package testdata

import (
	"net/http"

	"github.com/gorilla/mux"
)

func gorillaRoutes() {
	r := mux.NewRouter()
	r.HandleFunc("/products", listProducts).Methods("GET")
	r.HandleFunc("/products", createProduct).Methods("POST")
	r.HandleFunc("/products/{id}", getProduct).Methods("GET")
	r.HandleFunc("/products/{id}", updateProduct).Methods("PUT")
	r.HandleFunc("/products/{id}", deleteProduct).Methods("DELETE")

	s := r.PathPrefix("/admin").Subrouter()
	s.HandleFunc("/users", adminListUsers).Methods("GET")
	s.HandleFunc("/users", adminCreateUser).Methods("POST")
}

func listProducts(w http.ResponseWriter, r *http.Request)   {}
func createProduct(w http.ResponseWriter, r *http.Request)  {}
func getProduct(w http.ResponseWriter, r *http.Request)     {}
func updateProduct(w http.ResponseWriter, r *http.Request)  {}
func deleteProduct(w http.ResponseWriter, r *http.Request)  {}
func adminListUsers(w http.ResponseWriter, r *http.Request) {}
func adminCreateUser(w http.ResponseWriter, r *http.Request) {}
