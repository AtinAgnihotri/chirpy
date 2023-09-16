package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
)

type ChirpResource struct {
	Body string `json:"body"`
	ID   int    `json:"id"`
}

type DBData struct {
	Chirps map[int]ChirpResource `json:"chirps"`
}

type DB struct {
	path string
	mux  *sync.RWMutex
}

var DataBase DB

func NewDB(path string) (*DB, error) {
	db := &DB{
		path: path,
		mux:  &sync.RWMutex{},
	}
	err := db.ensureDB()
	return db, err
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

func (db *DB) createDB() error {
	return db.writeDB(DBData{
		Chirps: map[int]ChirpResource{},
	})
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
