package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type apiConfig struct {
	fileServerHits int
}

type Chirp struct {
	Body string `json:"body"`
}

type ChirpValidity struct {
	Valid bool `json:"valid"`
}

func (cfg *apiConfig) middlewareMetricsIncrement(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits += 1
		next.ServeHTTP(w, r)
	})
}

func respondWithError(w http.ResponseWriter, code int, msg string) error {
	return respondWithJSON(w, code, map[string]string{"error": msg})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) error {
	response, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
	return nil
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
	cfg := apiConfig{}
	// mux := http.NewServeMux()
	port := "8080"
	fileDir := http.Dir(".")
	r := chi.NewRouter()

	// Mount /api namespace
	r.Mount("/api", apiHandler(&cfg))

	// Mount /admin namespace
	r.Mount("/admin", adminHandler(&cfg))

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

func apiHandler(cfg *apiConfig) http.Handler {
	r := chi.NewRouter()
	// health endpoint
	r.Get("/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("OK"))
		return
	}))

	// metrics endpoints
	r.Get("/metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Hits: %v", cfg.fileServerHits)))
		return
	}))
	r.Handle("/reset", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		cfg.fileServerHits = 0
		return
	}))

	// validate endpoint
	r.Post("/validate_chirp", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		decoder := json.NewDecoder(r.Body)
		chirp := Chirp{}
		err := decoder.Decode(&chirp)

		errResponse := []byte(`{
			"error": "Something went wrong"
		}`)

		w.Header().Set("Content-Type", "application/json")

		if err != nil {
			log.Printf("Error decoding request body %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(errResponse)
			return
		}

		tooLongResponse := []byte(`{
			"error": "Chirp is too long"
		}`)
		validResponse := []byte(`{
			"valid":true
		}`)

		if len(chirp.Body) > 140 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write(tooLongResponse)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(validResponse)
		}

	}))
	return r
}

func adminHandler(cfg *apiConfig) http.Handler {
	r := chi.NewRouter()

	// metrics endpoints
	r.Get("/metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`<html>

		<body>
			<h1>Welcome, Chirpy Admin</h1>
			<p>Chirpy has been visited %d times!</p>
		</body>
		
		</html>`, cfg.fileServerHits)))
		return
	}))

	return r
}
