package database

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	BlocksBucket     = []byte("blocks")
	ChainStateBucket = []byte("chainstate")
)

var ErrNotFound = fmt.Errorf("key not found")

// KeyValueStore defines a minimal persistence interface.
type KeyValueStore interface {
	Put(bucket, key, value []byte) error
	Get(bucket, key []byte) ([]byte, error)
	Delete(bucket, key []byte) error
	Iterate(bucket []byte, fn func(key, value []byte) error) error
}

// Database is a simple file-backed key-value store.
type Database struct {
	path  string
	store map[string][]byte
	mu    sync.RWMutex
}

// InitDatabase initializes or opens a file-backed database.
func InitDatabase(dirPath string) (*Database, func()) {
	path := filepath.Join(dirPath, "duniyani_store.gob")
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		panic(fmt.Errorf("failed to create database directory: %w", err))
	}

	db := &Database{path: path, store: make(map[string][]byte)}
	if err := db.load(); err != nil {
		panic(fmt.Errorf("failed to load database: %w", err))
	}

	closeFunc := func() {
		if err := db.save(); err != nil {
			panic(fmt.Errorf("failed to save database: %w", err))
		}
	}

	return db, closeFunc
}

func (db *Database) fileKey(bucket, key []byte) string {
	return string(bucket) + "|" + string(key)
}

func (db *Database) load() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, err := os.Stat(db.path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	file, err := os.Open(db.path)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	return decoder.Decode(&db.store)
}

func (db *Database) save() error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	file, err := os.Create(db.path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	return encoder.Encode(db.store)
}

// Put stores a key/value pair in the specified bucket.
func (db *Database) Put(bucket, key, value []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	composite := db.fileKey(bucket, key)
	db.store[composite] = append([]byte(nil), value...)
	return db.save()
}

// Get retrieves a value by key from the specified bucket.
func (db *Database) Get(bucket, key []byte) ([]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	composite := db.fileKey(bucket, key)
	value, ok := db.store[composite]
	if !ok {
		return nil, ErrNotFound
	}
	return append([]byte(nil), value...), nil
}

// Delete removes a key from the specified bucket.
func (db *Database) Delete(bucket, key []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	composite := db.fileKey(bucket, key)
	delete(db.store, composite)
	return db.save()
}

// Iterate executes the callback for every entry in a bucket.
func (db *Database) Iterate(bucket []byte, fn func(key, value []byte) error) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	prefix := string(bucket) + "|"
	for composite, value := range db.store {
		if !startsWith(composite, prefix) {
			continue
		}
		key := []byte(composite[len(prefix):])
		if err := fn(key, append([]byte(nil), value...)); err != nil {
			return err
		}
	}
	return nil
}

func startsWith(value, prefix string) bool {
	return len(value) >= len(prefix) && value[:len(prefix)] == prefix
}
