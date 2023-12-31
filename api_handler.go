package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/AtinAgnihotri/chirpy/internal/database"
	"github.com/go-chi/chi/v5"
)

type RefreshResponse struct {
	Token string `json:"token"`
}

type Chirp struct {
	Body string `json:"body"`
}

type CleanedChirp struct {
	CleanedBody string `json:"cleaned_body"`
}

type keyCfg struct {
	Key []byte
}

type PolkaRequest struct {
	Event string `json:"event"`
	Data  struct {
		UserID int `json:"user_id"`
	} `json:"data"`
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

		authToken, err := GetAuthBearer(r)
		if err != nil {
			log.Printf("Error getting auth token %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}

		claims, err := GetJWTClaims(authToken, cfg.JWTSecret)
		if err != nil {
			log.Printf("Error getting claims from JWT %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something Went Wrong")
			return
		}

		issuer, err := claims.GetIssuer()
		if err != nil {
			log.Printf("Error getting issuer  %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something Went Wrong")
			return
		}

		if issuer != "chirpy-access" {
			log.Printf("Invalid issuer recieved")
			RespondWithError(w, http.StatusUnauthorized, "Authorization Rejected")
			return
		}

		subject, err := claims.GetSubject()
		if err != nil {
			log.Printf("Error getting subject %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}
		userId, err := strconv.Atoi(subject)
		if err != nil {
			log.Printf("Error getting user id %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}

		decoder := json.NewDecoder(r.Body)
		chirp := Chirp{}
		err = decoder.Decode(&chirp)

		if err != nil {
			log.Printf("Error decoding request body %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}

		if len(chirp.Body) > 140 {
			RespondWithError(w, http.StatusBadRequest, "Chirp is too long")
			return
		}
		chirpRsc, err := db.CreateChirp(CleanupBody(chirp.Body), userId)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Unable to create a chirp")
			return
		}

		RespondWithJSON(w, http.StatusCreated, chirpRsc)

	}))

	r.Delete("/chirps/{chirpid}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		authToken, err := GetAuthBearer(r)
		if err != nil {
			log.Printf("Error getting auth token %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}

		claims, err := GetJWTClaims(authToken, cfg.JWTSecret)
		if err != nil {
			log.Printf("Error getting claims from JWT %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something Went Wrong")
			return
		}

		issuer, err := claims.GetIssuer()
		if err != nil {
			log.Printf("Error getting issuer  %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something Went Wrong")
			return
		}

		if issuer != "chirpy-access" {
			log.Printf("Invalid issuer recieved")
			RespondWithError(w, http.StatusUnauthorized, "Authorization Rejected")
			return
		}

		subject, err := claims.GetSubject()
		if err != nil {
			log.Printf("Error getting subject %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}
		userId, err := strconv.Atoi(subject)
		if err != nil {
			log.Printf("Error getting user id %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}
		param := chi.URLParam(r, "chirpid")
		chirpId, err := strconv.Atoi(param)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "unable to fetch chirps")
			return
		}

		err = db.DeleteChirp(chirpId, userId)
		if err != nil {
			message := err.Error()
			consumerMessage := "Something went wrong"
			consumerCode := http.StatusInternalServerError
			if message == "Chirp Author Invalid Authorization" {
				consumerCode = http.StatusForbidden
				consumerMessage = message
			}
			log.Printf(message)
			RespondWithError(w, consumerCode, consumerMessage)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))

	r.Get("/chirps", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		chirps, err := db.GetChirps()
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "unable to fetch chirps")
			return
		}
		sortBy := "asc"
		sortByParam := r.URL.Query().Get("sort")
		if sortByParam == "desc" {
			sortBy = sortByParam
		}
		fmt.Println("Chirps before sort", len(chirps), sortBy)
		sort.Slice(chirps, func(p, q int) bool {
			if sortBy == "desc" {
				return chirps[p].ID > chirps[q].ID
			}
			return chirps[p].ID < chirps[q].ID
		})
		fmt.Println("Chirps after sort", len(chirps))
		for idx, sl := range chirps {
			fmt.Println(fmt.Sprintf("%v: %v, %v, %v", idx, sl.AuthorID, sl.ID, sl.Body))
		}
		authorIdParam := r.URL.Query().Get("author_id")
		if len(authorIdParam) == 0 {
			RespondWithJSON(w, http.StatusOK, chirps)
			return
		}
		authorId, err := strconv.Atoi(authorIdParam)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}
		var chirpsForAuthor []database.ChirpResource
		for _, chirp := range chirps {
			if chirp.AuthorID == authorId {
				chirpsForAuthor = append(chirpsForAuthor, chirp)
			}
		}
		RespondWithJSON(w, http.StatusOK, chirpsForAuthor)
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
		claims, err := GetJWTClaims(authHeader, cfg.JWTSecret)
		if err != nil {
			log.Printf("Error getting claims from JWT %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something Went Wrong")
			return
		}

		issuer, err := claims.GetIssuer()
		if err != nil {
			log.Printf("Error getting issuer  %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something Went Wrong")
			return
		}

		if issuer != "chirpy-access" {
			log.Printf("Invalid issuer recieved")
			RespondWithError(w, http.StatusUnauthorized, "Authorization Rejected")
			return
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
		accessToken, err := GenerateAccessToken(usr.ID, cfg.JWTSecret)
		if err != nil {
			log.Printf("Error generating access token %v", err)
			RespondWithError(w, http.StatusUnauthorized, "Something went wrong")
			return
		}
		refreshToken, err := GenerateRefreshToken(usr.ID, cfg.JWTSecret)
		if err != nil {
			log.Printf("Error generating refresh token %v", err)
			RespondWithError(w, http.StatusUnauthorized, "Something went wrong")
			return
		}
		RespondWithJSON(w, http.StatusOK, database.UserResource{
			Email:        usr.Email,
			ID:           usr.ID,
			Token:        accessToken,
			RefreshToken: refreshToken,
			IsChirpyRed:  usr.IsChirpyRed,
		})

	}))

	r.Post("/refresh", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		authHeader := strings.Replace(r.Header.Get("Authorization"), "Bearer ", "", 1)

		if len(authHeader) == 0 {
			RespondWithError(w, http.StatusUnauthorized, "Authorization token not recieved")
			return
		}

		claims, err := GetJWTClaims(authHeader, cfg.JWTSecret)
		if err != nil {
			log.Printf("Error getting claims from JWT %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something Went Wrong")
			return
		}

		issuer, err := claims.GetIssuer()
		if err != nil {
			log.Printf("Error getting issuer  %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something Went Wrong")
			return
		}

		if issuer != "chirpy-refresh" {
			log.Printf("Invalid issuer recieved")
			RespondWithError(w, http.StatusUnauthorized, "Authorization Rejected")
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

		revokedTokens, err := db.GetRevokedTokens()
		if err != nil {
			log.Printf("Error getting revoked tokens %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something Went Wrong")
			return
		}

		if Includes[string](revokedTokens, authHeader) {
			log.Printf("Invalid issuer recieved")
			RespondWithError(w, http.StatusUnauthorized, "Authorization Rejected")
			return
		}

		accessToken, err := GenerateAccessToken(id, cfg.JWTSecret)

		if err != nil {
			log.Printf("Error getting user id %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}

		RespondWithJSON(w, http.StatusOK, RefreshResponse{
			Token: accessToken,
		})
	}))

	r.Post("/revoke", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authToken, err := GetAuthBearer(r)
		if err != nil {
			log.Printf("Error getting auth token %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}

		claims, err := GetJWTClaims(authToken, cfg.JWTSecret)
		if err != nil {
			log.Printf("Error getting claims from JWT %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something Went Wrong")
			return
		}

		issuer, err := claims.GetIssuer()
		if err != nil {
			log.Printf("Error getting issuer  %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something Went Wrong")
			return
		}

		if issuer != "chirpy-refresh" {
			log.Printf("Invalid issuer recieved")
			RespondWithError(w, http.StatusUnauthorized, "Authorization Rejected")
			return
		}

		err = db.RevokeToken(authToken)
		if err != nil {
			log.Printf("Error revoking token  %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something Went Wrong")
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	r.Post("/polka/webhooks", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		event := PolkaRequest{}

		apiKey, err := GetAuthApiKey(r)
		if err != nil {
			log.Printf("error getting api key %v", err)
			RespondWithError(w, http.StatusUnauthorized, "Not Authorized")
			return
		}
		if apiKey != cfg.PolkaApiKey {
			log.Printf("error getting api key %v", err)
			RespondWithError(w, http.StatusUnauthorized, "Not Authorized")
			return
		}

		decoder := json.NewDecoder(r.Body)
		err = decoder.Decode(&event)
		if err != nil {
			log.Printf("error decoding polka response %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}
		fmt.Println("Event", event.Event)
		if event.Event != "user.upgraded" {
			fmt.Println("Reaches here 2")
			w.WriteHeader(http.StatusOK)
			return
		}
		fmt.Println("Reaches here 3")
		err = db.MarkUserChirpyRed(event.Data.UserID)
		if err != nil {
			log.Printf("error marking user red %v", err)
			message := err.Error()
			consumerCode := http.StatusInternalServerError
			consumerMessage := "Something went wrong"
			if message == "User Not Found" {
				consumerCode = http.StatusNotFound
				consumerMessage = "User Not Found"
			}
			RespondWithError(w, consumerCode, consumerMessage)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}))

	return r
}
