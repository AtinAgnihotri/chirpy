package database

import (
	"encoding/json"
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

func (db *DB) GetChirps() ([]ChirpResource, error) {
	var chirps []ChirpResource
	dbData, err := db.loadDB()
	if err != nil {
		return chirps, nil
	}
	for _, chirp := range dbData.Chirps {
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
