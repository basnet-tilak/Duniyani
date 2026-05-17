package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTransactionHash tests the hashing function for the Transaction struct.
func TestTransactionHash(t *testing.T) {
	t.Parallel()

	tx := &Transaction{
		From:      []byte("sender"),
		To:        []byte("receiver"),
		Value:     100,
		Timestamp: time.Now().UnixNano(),
	}

	hash1 := tx.Hash()
	hash2 := tx.Hash() // Second call should return the cached hash

	require.NotEmpty(t, hash1, "Hash should not be empty")
	assert.Equal(t, hash1, hash2, "Consecutive hash calls should return the same value")

	// Modify the transaction and ensure the hash changes
	tx.Value = 200
	tx.hash = nil // Invalidate cache
	hash3 := tx.Hash()
	assert.NotEqual(t, hash1, hash3, "Hash should change when transaction data changes")
}

// TestBlockCreationAndHash tests the creation of a block and its hashing.
func TestBlockCreationAndHash(t *testing.T) {
	t.Parallel()

	tx1 := &Transaction{Value: 10, Signature: []byte("sig1")}
	tx2 := &Transaction{Value: 20, Signature: []byte("sig2")}

	testCases := []struct {
		name         string
		header       *BlockHeader
		transactions []*Transaction
	}{
		{
			name: "Block with two transactions",
			header: &BlockHeader{
				Height:    1,
				Timestamp: time.Now().UnixNano(),
			},
			transactions: []*Transaction{tx1, tx2},
		},
		{
			name: "Block with no transactions",
			header: &BlockHeader{
				Height:    2,
				Timestamp: time.Now().UnixNano(),
			},
			transactions: []*Transaction{},
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			block := NewBlock(tc.header, tc.transactions)

			require.NotNil(t, block, "NewBlock should not return nil")
			require.NotEmpty(t, block.Hash(), "Block hash should not be empty")
			assert.NotEmpty(t, block.Header.TxRoot, "TxRoot should be calculated")

			// Ensure Merkle root is correct
			expectedTxRoot := CalculateMerkleRoot(tc.transactions)
			assert.Equal(t, expectedTxRoot, block.Header.TxRoot)
		})
	}
}

// --- Benchmarks ---
// To run benchmarks, use the command:
// go test -bench=. -benchmem,
// To compare benchmarks, use bench stat.

// BenchmarkTransactionHash measures the performance of transaction hashing.
func BenchmarkTransactionHash(b *testing.B) {
	tx := &Transaction{
		From:      []byte("from"),
		To:        []byte("to"),
		Value:     12345,
		Timestamp: time.Now().UnixNano(),
	}
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tx.hash = nil // Invalidate cache to re-calculate
			tx.Hash()
		}
	})
}

// BenchmarkBlockValidation measures the performance of creating a new block, which includes hashing.
func BenchmarkBlockValidation(b *testing.B) {
	txs := make([]*Transaction, 100)
	for i := 0; i < 100; i++ {
		txs[i] = &Transaction{Value: uint64(i)}
	}
	header := &BlockHeader{Height: 1, Timestamp: time.Now().UnixNano()}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			NewBlock(header, txs)
		}
	})
}
