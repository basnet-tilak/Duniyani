package economics

import (
	"testing"

	"github.com/basnet-tilak/Duniyani/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoinbaseTransaction(t *testing.T) {
	t.Parallel()

	w := wallet.NewWallet()
	addr := w.GetAddress()
	data := "test coinbase"

	tx := NewCoinbaseTX(addr, data, 1)

	require.NotNil(t, tx, "Coinbase transaction should not be nil")
	assert.True(t, tx.IsCoinbase(), "Transaction should be a coinbase transaction")
	require.Len(t, tx.Vin, 1, "Coinbase should have one input")
	assert.Len(t, tx.Vin[0].TxID, 0, "Coinbase input TxID should be empty")
	assert.Equal(t, -1, tx.Vin[0].Vout, "Coinbase input Vout should be -1")
	require.Len(t, tx.Vout, 1, "Coinbase should have one output")
	assert.Equal(t, int64(BlockReward(0)), tx.Vout[0].Value, "Coinbase output should have the block reward")
}

func TestGenesisBlock(t *testing.T) {
	t.Parallel()

	w := wallet.NewWallet()
	addr := w.GetAddress()

	genesis := CreateGenesisBlock(addr)

	require.NotNil(t, genesis, "Genesis block should not be nil")
	require.Len(t, genesis.Transactions, 1, "Genesis block should have one transaction")

	genesisTx := genesis.Transactions[0]
	assert.True(t, genesisTx.IsCoinbase(), "Genesis transaction should be a coinbase")

	// Check that the value is the special Genesis amount, not the standard block reward
	assert.Equal(t, int64(GenesisCoinbaseAmount), genesisTx.Vout[0].Value, "Genesis output should have the genesis amount")

	// Check that the prev block hash is empty
	assert.Len(t, genesis.Header.PrevBlockHash, 0, "Genesis block's prev hash should be empty")
}
