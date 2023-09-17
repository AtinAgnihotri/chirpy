package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func RespondWithError(w http.ResponseWriter, code int, msg string) error {
	return RespondWithJSON(w, code, map[string]string{"error": msg})
}

func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) error {
	response, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
	return nil
}

func IsBannedWord(token string) bool {
	banned := []string{"kerfuffle", "sharbert", "fornax"}
	for _, b := range banned {
		if strings.ToLower(token) == b {
			return true
		}
	}
	return false
}

func CleanupBody(body string) string {
	clean := strings.TrimSpace(body)
	tokens := strings.Split(body, " ")
	for _, token := range tokens {
		if IsBannedWord(token) {
			clean = strings.Replace(clean, token, "****", 1)
		}
	}
	return clean
}

func GetHashedPassword(pwd string) (string, error) {
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashBytes), nil
}
