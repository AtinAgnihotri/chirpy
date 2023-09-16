package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/AtinAgnihotri/chirpy/internal/database"
	"github.com/go-chi/chi/v5"
)

type Chirp struct {
	Body string `json:"body"`
}

type CleanedChirp struct {
	CleanedBody string `json:"cleaned_body"`
}

func ApiHandler(cfg *ApiConfig, db *database.DB) http.Handler {
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
	r.Post("/chirps", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		decoder := json.NewDecoder(r.Body)
		chirp := Chirp{}
		err := decoder.Decode(&chirp)

		// w.Header().Set("Content-Type", "application/json")

		if err != nil {
			log.Printf("Error decoding request body %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}

		if len(chirp.Body) > 140 {
			RespondWithError(w, http.StatusBadRequest, "Chirp is too long")
			return
		}
		chirpRsc, err := db.CreateChirp(CleanupBody(chirp.Body))
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Unable to create a chirp")
			return
		}

		RespondWithJSON(w, http.StatusCreated, chirpRsc)

	}))

	r.Get("/chirps", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		chirps, err := db.GetChirps()
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "unable to fetch chirps")
			return
		}
		RespondWithJSON(w, http.StatusOK, chirps)
	}))

	r.Get("/chirps/{chirpid}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		param := chi.URLParam(r, "chirpid")
		id, err := strconv.Atoi(param)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "unable to fetch chirps")
			return
		}
		chirps, err := db.GetChirp(id)
		if err != nil {
			RespondWithError(w, http.StatusNotFound, "unable to fetch chirps")
			return
		}
		RespondWithJSON(w, http.StatusOK, chirps)
	}))
	return r
}
