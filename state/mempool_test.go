package state

import (
	"encoding/hex"
	"sync"
	"testing"

	"github.com/basnet-tilak/Duniyani/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMempoolAddAndGet tests adding and retrieving transactions from the mempool.
func TestMempoolAddAndGet(t *testing.T) {
	t.Parallel()

	tx1 := &core.Transaction{Vin: []core.TxInput{{}}, Vout: []core.TxOutput{{}}}
	tx2 := &core.Transaction{Vin: []core.TxInput{{}, {}}, Vout: []core.TxOutput{{}}}

	testCases := []struct {
		name        string
		tx          *core.Transaction
		expectError bool
	}{
		{"Add a new transaction", tx1, false},
		{"Add duplicate transaction", tx1, true},
		{"Add another new transaction", tx2, false},
		{"Add transaction with no inputs or outputs", &core.Transaction{Vout: []core.TxOutput{{}}}, true},
	}

	mempool := NewMempool()

	for _, tc := range testCases {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Note: Sub-tests here are sequential because they depend on shared mempool state.
			err := mempool.Add(tc.tx)
			if tc.expectError {
				assert.Error(t, err, "Expected an error but got none")
			} else {
				assert.NoError(t, err, "Did not expect an error but got one")
				txHash := hex.EncodeToString(tc.tx.Hash())
				retrievedTx, exists := mempool.transactions[txHash]
				assert.True(t, exists, "Transaction should exist in the mempool")
				assert.Equal(t, tc.tx, retrievedTx, "Retrieved transaction does not match the original")
			}
		})
	}

	// Test GetAll
	allTxs := mempool.GetAll()
	assert.Len(t, allTxs, 2, "Mempool should contain two transactions")

	// Test Clear
	mempool.Clear()
	assert.Empty(t, mempool.GetAll(), "Mempool should be empty after clearing")
}

// TestMempoolConcurrency aggressively tests the mempool for race conditions.
// To run with the race detector, use: go test -race -run TestMempoolConcurrency
func TestMempoolConcurrency(t *testing.T) {
	t.Parallel()

	mempool := NewMempool()
	numGoroutines := 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()

			// Create a unique transaction for each goroutine.
			tx := &core.Transaction{
				Vin:  []core.TxInput{{Vout: i}},
				Vout: []core.TxOutput{{Value: int64(i)}},
			}

			// Mix of operations: Add, GetAll, Clear
			if err := mempool.Add(tx); err == nil {
				_ = mempool.GetAll() // Read operation
			} else {
				// If add fails (e.g., duplicate), just do a read.
				_ = mempool.GetAll()
			}

			// Occasionally clear the mempool to stress test locking.
			if i%10 == 0 {
				mempool.Clear()
			}
		}(i)
	}

	wg.Wait()

	// Final check on the state of the mempool.
	// The final size is non-deterministic due to the clearing, but the test
	//  would have failed if the race detector caught anything.
	require.NotPanics(t, func() {
		_ = mempool.GetAll()
	}, "Final GetAll should not panic")
}
