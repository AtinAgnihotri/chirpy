package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const ACCESS_TOKEN_TIME = 60 * 60
const REFRESH_TOKEN_TIME = 60 * 60 * 24 * 60
const ACCESS_TOKEN_TYPE = "access"
const REFRESH_TOKEN_TYPE = "refresh"

func Includes[T comparable](arr []T, val T) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}

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

func generateJWT(userID, expiresTimeInSeconds int, tokenType, jwtSecret string) (string, error) {
	claims := jwt.RegisteredClaims{
		Issuer:    "chirpy-" + tokenType,
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiresTimeInSeconds) * time.Second)),
		Subject:   fmt.Sprintf("%v", userID),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

func GenerateAccessToken(userID int, jwtSecret string) (string, error) {
	return generateJWT(userID, ACCESS_TOKEN_TIME, ACCESS_TOKEN_TYPE, jwtSecret)
}

func GenerateRefreshToken(userID int, jwtSecret string) (string, error) {
	return generateJWT(userID, REFRESH_TOKEN_TIME, REFRESH_TOKEN_TYPE, jwtSecret)
}

func GetJWTClaims(authHeader, jwtSecret string) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(authHeader, claims, func(token *jwt.Token) (interface{}, error) {
		// since we only use the one private key to sign the tokens,
		// we also only use its public counter part to verify
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return claims, err
	}

	return claims, nil
}

func GetAuthToken(r *http.Request) (string, error) {
	authHeader := strings.Replace(r.Header.Get("Authorization"), "Bearer ", "", 1)
	if len(authHeader) == 0 {
		return "", errors.New("No authorization header recieved")
	}
	return authHeader, nil
}
