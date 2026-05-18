package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionHashing(t *testing.T) {
	t.Parallel()
	tx := &Transaction{
		Vin:  []TxInput{{Vout: 1}},
		Vout: []TxOutput{{Value: 10}},
	}
	hash := tx.Hash()

	// SHA256 generates a 32 byte hash length
	assert.Len(t, hash, 32)
}

func TestBlockCreationAndHashing(t *testing.T) {
	t.Parallel()
	tx := &Transaction{
		Vin:  []TxInput{{Vout: 1}},
		Vout: []TxOutput{{Value: 10}},
	}

	block := NewBlock([]*Transaction{tx}, []byte("prevHash"), 1, 10)
	require.NotNil(t, block)
	assert.Equal(t, uint32(1), block.Header.Version)
	assert.Len(t, block.Header.MerkleRoot, 32)

	hash := block.Header.Hash()
	assert.Len(t, hash, 32)
}

func TestMerkleTree(t *testing.T) {
	t.Parallel()
	data := [][]byte{
		[]byte("tx1"),
		[]byte("tx2"),
		[]byte("tx3"),
	}
	tree := NewMerkleTree(data)

	require.NotNil(t, tree)
	require.NotNil(t, tree.RootNode)
	assert.Len(t, tree.RootNode.Data, 32)
}
