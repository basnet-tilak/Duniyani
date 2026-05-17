package wallet

import (
	"testing"

	"github.com/basnet-tilak/Duniyani/core"
	"github.com/basnet-tilak/Duniyani/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWallet(t *testing.T) {
	t.Parallel()

	w := NewWallet()
	address := w.GetAddress()

	assert.NotNil(t, w.PrivateKey, "Private key should not be nil")
	assert.NotNil(t, w.PublicKey, "Public key should not be nil")
	assert.Equal(t, "D", string(address[0]), "Address should start with 'D'")
}

func TestSignAndVerifyTransaction(t *testing.T) {
	t.Parallel()

	// 1. Create wallets for sender and receiver
	senderWallet := NewWallet()
	receiverWallet := NewWallet()

	// 2. Create a "previous" transaction that funds the sender
	// This simulates a UTXO that the sender owns.
	prevTx := &core.Transaction{
		ID: []byte("prev_tx_id"),
		Vout: []core.TxOutput{
			{
				Value:      1000,
				PubKeyHash: crypto.HashPubKey(crypto.SerializeCompressed(senderWallet.PublicKey)),
			},
		},
	}

	// 3. Create the new transaction to be signed
	newTx := &core.Transaction{
		Vin: []core.TxInput{
			{
				TxID:      prevTx.ID,
				Vout:      0, // Referencing the first output of the previous tx
				Signature: nil,
				PubKey:    nil, // Will be set during signing
			},
		},
		Vout: []core.TxOutput{
			{
				Value:      500, // Send 500 to receiver
				PubKeyHash: crypto.HashPubKey(crypto.SerializeCompressed(receiverWallet.PublicKey)),
			},
			{
				Value:      500, // Change back to sender
				PubKeyHash: crypto.HashPubKey(crypto.SerializeCompressed(senderWallet.PublicKey)),
			},
		},
	}
	newTx.ID = newTx.Hash()

	// 4. Sign the transaction
	prevTxs := map[string]core.Transaction{string(prevTx.ID): *prevTx}
	err := senderWallet.SignTransaction(newTx, prevTxs)
	require.NoError(t, err, "Signing should not produce an error")
	assert.NotNil(t, newTx.Vin[0].Signature, "Signature should be present after signing")

	// 5. Verify the transaction
	valid, err := VerifyTransaction(newTx, prevTxs)
	require.NoError(t, err, "Verification should not produce an error")
	assert.True(t, valid, "Transaction signature should be valid")

	// 6. Test with a tampered transaction
	newTx.Vout[0].Value = 9999 // Change the amount
	invalid, err := VerifyTransaction(newTx, prevTxs)
	require.NoError(t, err, "Verification of invalid tx should not error")
	assert.False(t, invalid, "Signature of a tampered transaction should be invalid")
}
