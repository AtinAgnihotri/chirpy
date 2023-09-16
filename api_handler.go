package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Chirp struct {
	Body string `json:"body"`
}

type CleanedChirp struct {
	CleanedBody string `json:"cleaned_body"`
}

func ApiHandler(cfg *ApiConfig) http.Handler {
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

	// chirp endpoint
	r.Post("/chirp", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		decoder := json.NewDecoder(r.Body)
		chirp := Chirp{}
		err := decoder.Decode(&chirp)

		w.Header().Set("Content-Type", "application/json")

		if err != nil {
			log.Printf("Error decoding request body %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}

		if len(chirp.Body) > 140 {
			RespondWithError(w, http.StatusBadRequest, "Chirp is too long")
			return
		}

		cleanChirp := CleanedChirp{
			CleanedBody: CleanupBody(chirp.Body),
		}
		fmt.Println(cleanChirp)

		// respondWithJSON(w, http.StatusOK, CleanedChirp{
		// 	CleanedBody: cleanupBody(chirp.Body),
		// })

	}))
	return r
}
