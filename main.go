package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/AtinAgnihotri/chirpy/internal/database"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

type ApiConfig struct {
	fileServerHits int
	JWTSecret      string
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

func isDebugMode() bool {
	dbg := flag.Bool("debug", false, "Enable debug mode in server")
	flag.Parse()
	return *dbg
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	jwtSecret := os.Getenv("JWT_SECRET")

	cfg := ApiConfig{}
	cfg.JWTSecret = jwtSecret
	db, err := database.NewDB("./db.json", isDebugMode())

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
