package database

import (
	"errors"
	"sync"
)

var (
	ErrNotFound      = errors.New("key not found")
	ChainStateBucket = []byte("chainstate")
	BlocksBucket     = []byte("blocks")
)

// DB defines the interface for our Key-Value store (Mocking BadgerDB).
type DB interface {
	Get(bucket, key []byte) ([]byte, error)
	Put(bucket, key, value []byte) error
	Delete(bucket, key []byte) error
}

// Database is an in-memory mock for the quantum blueprint.
type Database struct {
	mu    sync.RWMutex
	store map[string][]byte
}

// InitDatabase creates a new mock database and returns a no-op close function.
func InitDatabase(path string) (*Database, func()) {
	db := NewDatabase()
	return db, func() {}
}

func NewDatabase() *Database {
	return &Database{store: make(map[string][]byte)}
}

func (db *Database) Get(bucket, key []byte) ([]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	compositeKey := string(bucket) + "|" + string(key)
	if val, ok := db.store[compositeKey]; ok {
		return val, nil
	}
	return nil, ErrNotFound
}

func (db *Database) Put(bucket, key, value []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	compositeKey := string(bucket) + "|" + string(key)
	db.store[compositeKey] = value
	return nil
}

func (db *Database) Delete(bucket, key []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	compositeKey := string(bucket) + "|" + string(key)
	if _, ok := db.store[compositeKey]; !ok {
		return ErrNotFound
	}
	delete(db.store, compositeKey)
	return nil
}

func (db *Database) Iterate(bucket []byte, fn func(key, value []byte) error) error {
	db.mu.RLock()
	defer db.mu.RUnlock()
	prefix := string(bucket) + "|"
	for k, v := range db.store {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			key := []byte(k[len(prefix):])
			if err := fn(key, v); err != nil {
				return err
			}
		}
	}
	return nil
}
