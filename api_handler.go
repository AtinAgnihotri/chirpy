package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/AtinAgnihotri/chirpy/internal/database"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
)

type Chirp struct {
	Body string `json:"body"`
}

type CleanedChirp struct {
	CleanedBody string `json:"cleaned_body"`
}

type keyCfg struct {
	Key []byte
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

	// Chirps endpoints
	r.Post("/chirps", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		decoder := json.NewDecoder(r.Body)
		chirp := Chirp{}
		err := decoder.Decode(&chirp)

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

	// Users endpoints
	r.Post("/users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		decoder := json.NewDecoder(r.Body)
		user := database.DetailedUserResource{}
		err := decoder.Decode(&user)

		if err != nil {
			log.Printf("Error decoding request body %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}

		hashBytes, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Error decoding request body %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}
		userRsc, err := db.CreateUsers(user.Email, string(hashBytes))
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Unable to create a user")
			return
		}

		RespondWithJSON(w, http.StatusCreated, userRsc)

	}))

	r.Get("/users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		users, err := db.GetUsers()
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "unable to fetch users")
			return
		}
		RespondWithJSON(w, http.StatusOK, users)
	}))

	r.Get("/users/{userid}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		param := chi.URLParam(r, "userid")
		id, err := strconv.Atoi(param)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "unable to fetch users")
			return
		}
		chirps, err := db.GetChirp(id)
		if err != nil {
			RespondWithError(w, http.StatusNotFound, "unable to fetch users")
			return
		}
		RespondWithJSON(w, http.StatusOK, chirps)
	}))

	r.Put("/users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		authHeader := strings.Replace(r.Header.Get("Authorization"), "Bearer ", "", 1)

		if len(authHeader) == 0 {
			RespondWithError(w, http.StatusUnauthorized, "Authorization token not recieved")
			return
		}
		claims := jwt.MapClaims{}
		_, err := jwt.ParseWithClaims(authHeader, claims, func(token *jwt.Token) (interface{}, error) {
			// since we only use the one private key to sign the tokens,
			// we also only use its public counter part to verify
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil {
			RespondWithError(w, http.StatusUnauthorized, "Authorization failed")
			return
		}

		for key, val := range claims {
			fmt.Printf("Key: %v, value: %v\n", key, val)
		}

		decoder := json.NewDecoder(r.Body)
		user := database.DetailedUserResource{}
		err = decoder.Decode(&user)

		if err != nil {
			log.Printf("Error decoding request body %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}
		subject, err := claims.GetSubject()
		if err != nil {
			log.Printf("Error getting subject %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}
		id, err := strconv.Atoi(subject)
		if err != nil {
			log.Printf("Error getting user id %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}
		user.ID = id
		hashedPwd, err := GetHashedPassword(user.Password)
		if err != nil {
			log.Printf("Error hashing pwd %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something went Wrong")
			return
		}
		user.Password = hashedPwd

		db.UpdateUsers(user)
		RespondWithJSON(w, http.StatusOK, database.UserResource{
			ID:    user.ID,
			Email: user.Email,
		})
	}))

	// login endpoint
	r.Post("/login", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		decoder := json.NewDecoder(r.Body)
		user := database.DetailedUserResource{}
		err := decoder.Decode(&user)

		if err != nil {
			log.Printf("Error decoding request body %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}
		userMap, err := db.GetUserMapByEmails()
		if err != nil {
			log.Printf("Error getting users %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}
		usr, ok := userMap[user.Email]
		if !ok {
			log.Printf("Error getting users data for user %v", user.Email)
			RespondWithError(w, http.StatusNotFound, fmt.Sprintf("%v user data not found", user.Email))
			return
		}

		hashCmpErr := bcrypt.CompareHashAndPassword([]byte(usr.Password), []byte(user.Password))
		if hashCmpErr != nil {
			log.Printf("Error matching password for %v", user.Email)
			RespondWithError(w, http.StatusUnauthorized, fmt.Sprintf("%v password incorrect", user.Email))
			return
		}
		expiresTime := 60 * 60 * 24
		if user.ExpiresInSeconds != nil {
			if *user.ExpiresInSeconds < expiresTime {
				expiresTime = *user.ExpiresInSeconds
			}
		}
		claims := jwt.RegisteredClaims{
			Issuer:    "chirpy",
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiresTime) * time.Second)),
			Subject:   fmt.Sprintf("%v", usr.ID),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signedString, err := token.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			signedString = "INVALID_TOKEN"
			log.Println("Couldn't generate a token", err)
		}
		RespondWithJSON(w, http.StatusOK, database.UserResource{
			Email: usr.Email,
			ID:    usr.ID,
			Token: signedString,
		})

	}))

	return r
}
