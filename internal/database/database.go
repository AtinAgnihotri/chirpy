package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

type ChirpResource struct {
	Body string `json:"body"`
	ID   int    `json:"id"`
}

type UserResource struct {
	Email        string `json:"email"`
	ID           int    `json:"id"`
	Token        string `json:"token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type AuthUserResource struct {
	Email string  `json:"email"`
	ID    int     `json:"id"`
	Token *string `json:"token"`
}

type DetailedUserResource struct {
	Email            string `json:"email"`
	ID               int    `json:"id"`
	Password         string `json:"password"`
	ExpiresInSeconds *int   `json:"expires_in_seconds"`
}

type DBData struct {
	Chirps        map[int]ChirpResource        `json:"chirps"`
	Users         map[int]DetailedUserResource `json:"users"`
	RevokedTokens map[string]int64             `json:"revoked_tokens"`
}

type DB struct {
	path string
	mux  *sync.RWMutex
}

var DataBase DB

func NewDB(path string, isDebug bool) (*DB, error) {
	db := &DB{
		path: path,
		mux:  &sync.RWMutex{},
	}
	if isDebug {
		cleanupErr := db.cleanupDBFile()
		if cleanupErr != nil {
			return db, cleanupErr
		}
	}
	err := db.ensureDB()
	return db, err
}

func (db *DB) CreateUsers(email string, hash string) (UserResource, error) {
	var user UserResource
	dbData, err := db.loadDB()
	if err != nil {
		return user, nil
	}
	newId := len(dbData.Users) + 1
	user = UserResource{
		Email: email,
		ID:    newId,
	}
	dbData.Users[newId] = DetailedUserResource{
		Email:    email,
		ID:       newId,
		Password: hash,
	}
	err = db.writeDB(dbData)
	if err != nil {
		return UserResource{}, nil
	}
	return user, nil
}

func (db *DB) CreateChirp(body string) (ChirpResource, error) {
	var chirp ChirpResource
	dbData, err := db.loadDB()
	if err != nil {
		return chirp, nil
	}
	newId := len(dbData.Chirps) + 1
	chirp = ChirpResource{
		Body: body,
		ID:   newId,
	}
	dbData.Chirps[newId] = chirp
	err = db.writeDB(dbData)
	if err != nil {
		return ChirpResource{}, nil
	}
	return chirp, nil
}

func (db *DB) getChirpMap() (map[int]ChirpResource, error) {
	dbData, err := db.loadDB()
	if err != nil {
		return nil, err
	}
	return dbData.Chirps, nil
}

func (db *DB) getUserMap() (map[int]UserResource, error) {
	dbData, err := db.loadDB()
	if err != nil {
		return nil, err
	}
	userMap := map[int]UserResource{}
	for key, val := range dbData.Users {
		userMap[key] = UserResource{
			ID:    val.ID,
			Email: val.Email,
		}
	}
	return userMap, nil
}

func (db *DB) RevokeToken(token string) error {
	dbData, err := db.loadDB()
	if err != nil {
		return err
	}
	dbData.RevokedTokens[token] = time.Now().UTC().Unix()
	return db.writeDB(dbData)
}

func (db *DB) GetRevokedTokens() (tokens []string, err error) {
	var revokedTokens []string
	dbData, err := db.loadDB()
	if err != nil {
		return revokedTokens, err
	}
	for key := range dbData.RevokedTokens {
		revokedTokens = append(revokedTokens, key)
	}
	return revokedTokens, nil
}

func (db *DB) GetUserMapByEmails() (map[string]DetailedUserResource, error) {
	pwdMap := map[string]DetailedUserResource{}
	dbData, err := db.loadDB()
	if err != nil {
		return pwdMap, err
	}
	for _, entry := range dbData.Users {
		pwdMap[entry.Email] = entry
	}
	return pwdMap, nil
}

func (db *DB) GetChirp(id int) (ChirpResource, error) {
	var chirp ChirpResource
	chirpMap, err := db.getChirpMap()
	if err != nil {
		return chirp, nil
	}
	chirp, ok := chirpMap[id]
	if !ok {
		return chirp, errors.New(fmt.Sprintf("No chirp with id %v found", id))
	}
	return chirp, nil
}

func (db *DB) GetChirps() ([]ChirpResource, error) {
	var chirps []ChirpResource
	chirpMap, err := db.getChirpMap()
	if err != nil {
		return chirps, nil
	}
	for _, chirp := range chirpMap {
		chirps = append(chirps, chirp)
	}
	return chirps, nil
}

func (db *DB) GetUser(id int) (UserResource, error) {
	var user UserResource
	userMap, err := db.getUserMap()
	if err != nil {
		return user, nil
	}
	user, ok := userMap[id]
	if !ok {
		return user, errors.New(fmt.Sprintf("No user with id %v found", id))
	}
	return user, nil
}

func (db *DB) GetUsers() ([]UserResource, error) {
	var users []UserResource
	userMap, err := db.getUserMap()
	if err != nil {
		return users, nil
	}
	for _, user := range userMap {
		users = append(users, user)
	}
	return users, nil
}

func (db *DB) UpdateUsers(user DetailedUserResource) error {
	dbData, err := db.loadDB()
	if err != nil {
		return err
	}

	dbData.Users[user.ID] = user

	return db.writeDB(dbData)
}

func (db *DB) createDB() error {
	return db.writeDB(DBData{
		Chirps:        map[int]ChirpResource{},
		Users:         map[int]DetailedUserResource{},
		RevokedTokens: map[string]int64{},
	})
}

func (db *DB) cleanupDBFile() error {
	db.mux.Lock()
	defer db.mux.Unlock()
	err := os.Remove(db.path)
	if err != nil {
		return err
	}
	return nil
}

func (db *DB) ensureDB() error {
	_, err := os.ReadFile(db.path)
	if os.IsNotExist(err) {
		return db.createDB()
	}

	return err
}

func (db *DB) loadDB() (DBData, error) {
	db.mux.RLock()
	defer db.mux.RUnlock()

	dbData := DBData{}
	data, err := os.ReadFile(db.path)
	if err != nil {
		return dbData, err
	}
	err = json.Unmarshal(data, &dbData)
	if err != nil {
		return dbData, err
	}
	return dbData, nil
}

func (db *DB) writeDB(dbData DBData) error {
	db.mux.Lock()
	defer db.mux.Unlock()

	data, err := json.Marshal(dbData)
	if err != nil {
		return err
	}
	err = os.WriteFile(db.path, data, 0600)
	if err != nil {
		return err
	}
	return nil
}
