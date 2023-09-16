package main

import (
	"log"
	"net/http"

	"github.com/AtinAgnihotri/chirpy/internal/database"
	"github.com/go-chi/chi/v5"
)

type ApiConfig struct {
	fileServerHits int
}

func (cfg *ApiConfig) middlewareMetricsIncrement(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits += 1
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	cfg := ApiConfig{}
	db, err := database.NewDB("./db.json")

	if err != nil {
		log.Fatal("Error setting up db", err)
	}
	// mux := http.NewServeMux()
	port := "8080"
	fileDir := http.Dir(".")
	r := chi.NewRouter()

	// Mount /api namespace
	r.Mount("/api", ApiHandler(&cfg, db))

	// Mount /admin namespace
	r.Mount("/admin", AdminHandler(&cfg))

	// fileserver endpoint
	fsHandler := cfg.middlewareMetricsIncrement(http.StripPrefix("/app", http.FileServer(fileDir)))
	r.Handle("/app/*", fsHandler)
	r.Handle("/app", fsHandler)

	corsMux := corsMiddleware(r)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: corsMux,
	}

	log.Printf("Serving files from %s on port: %s\n", fileDir, port)
	log.Fatal(srv.ListenAndServe())

}
