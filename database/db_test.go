package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabasePutGetDelete(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	db, closeDB := InitDatabase(dir)
	defer closeDB()

	bucket := []byte("testbucket")
	key := []byte("key1")
	value := []byte("value1")

	// Test Put
	err := db.Put(bucket, key, value)
	require.NoError(t, err, "Database Put should succeed without errors")

	// Test Get
	retrieved, err := db.Get(bucket, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved, "Retrieved value should match the original data")

	// Test Delete
	err = db.Delete(bucket, key)
	require.NoError(t, err)

	_, err = db.Get(bucket, key)
	assert.ErrorIs(t, err, ErrNotFound, "Fetching a deleted key must yield ErrNotFound")
}

func TestDatabasePersistenceAndIteration(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	db, closeDB := InitDatabase(dir)

	bucket := []byte("chainstate")
	db.Put(bucket, []byte("utxo1"), []byte("data1"))
	db.Put(bucket, []byte("utxo2"), []byte("data2"))

	// Close the DB to force a flush to disk
	closeDB()

	// Re-open DB to simulate a node reboot
	db2, closeDB2 := InitDatabase(dir)
	defer closeDB2()

	count := 0
	err := db2.Iterate(bucket, func(key, value []byte) error {
		count++
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 2, count, "Iterator should correctly discover 2 keys upon restoring state")

	val, _ := db2.Get(bucket, []byte("utxo1"))
	assert.Equal(t, []byte("data1"), val)
}
