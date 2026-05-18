package state

import (
	"fmt"
	"sync"
	"testing"

	"github.com/basnet-tilak/Duniyani/core"
	"github.com/basnet-tilak/Duniyani/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMempoolAddAndGet tests adding and retrieving transactions from the mempool.
func TestMempoolAddAndGet(t *testing.T) {
	t.Parallel()

	db := database.NewDatabase()
	mempool := NewMempool(db)

	// Create input transactions that will be UTXOs for our test transactions
	inputTx1 := &core.Transaction{
		Vin:  []core.TxInput{{TxID: []byte{}, Vout: -1}}, // Coinbase
		Vout: []core.TxOutput{{Value: 100000}},
	}
	inputTx2 := &core.Transaction{
		Vin:  []core.TxInput{{TxID: []byte{}, Vout: -1}}, // Coinbase
		Vout: []core.TxOutput{{Value: 100000}},
	}
	// Make them different by adding different timestamps
	inputTx1.Timestamp = 1000
	inputTx2.Timestamp = 2000
	inputTx1.ID = inputTx1.Hash()
	inputTx2.ID = inputTx2.Hash()

	// Create regular transactions that spend from our input transactions
	// Make them different too
	tx1 := &core.Transaction{
		Vin:       []core.TxInput{{TxID: inputTx1.ID, Vout: 0}},
		Vout:      []core.TxOutput{{Value: 95000}}, // Leave enough for fees (~3220)
		Timestamp: 1000,
	}
	tx2 := &core.Transaction{
		Vin:       []core.TxInput{{TxID: inputTx2.ID, Vout: 0}},
		Vout:      []core.TxOutput{{Value: 94000}}, // Slightly different value
		Timestamp: 2000,
	}
	
	tx1.ID = tx1.Hash()
	tx2.ID = tx2.Hash()

	// Add UTXO entries from input transactions to the database
	txout1, _ := inputTx1.Vout[0].Serialize()
	txout2, _ := inputTx2.Vout[0].Serialize()
	inputID1Hex := fmt.Sprintf("%x:0", inputTx1.ID)
	inputID2Hex := fmt.Sprintf("%x:0", inputTx2.ID)
	db.Put(database.ChainStateBucket, []byte(inputID1Hex), txout1)
	db.Put(database.ChainStateBucket, []byte(inputID2Hex), txout2)

	testCases := []struct {
		name        string
		tx          *core.Transaction
		expectError bool
	}{
		{"Add a new transaction", tx1, false},
		{"Add duplicate transaction", tx1, true},
		{"Add another new transaction", tx2, false},
	}

	for _, tc := range testCases {
		err := mempool.Add(tc.tx)
		if tc.expectError {
			assert.Error(t, err, "Expected an error for: "+tc.name)
		} else {
			assert.NoError(t, err, "Did not expect an error for: "+tc.name)
			txID := string(tc.tx.ID)
			_, exists := mempool.Transactions[txID]
			assert.True(t, exists, "Transaction should exist in the mempool for: "+tc.name)
		}
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

	db := database.NewDatabase()
	mempool := NewMempool(db)
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
			tx.ID = tx.Hash()

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
