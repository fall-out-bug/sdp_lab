package testdata

import (
	"net/http"
)

func stdlibRoutes() {
	mux := http.NewServeMux()
	mux.HandleFunc("/home", homePage)
	mux.Handle("/static", http.FileServer(http.Dir(".")))

	http.HandleFunc("/legacy", legacyHandler)
	http.Handle("/health", healthServer{})
}

func homePage(w http.ResponseWriter, r *http.Request) {}
func legacyHandler(w http.ResponseWriter, r *http.Request) {}

type healthServer struct{}

func (h healthServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {}
