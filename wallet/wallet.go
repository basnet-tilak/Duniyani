package wallet

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"

	"github.com/basnet-tilak/Duniyani/core"
	"github.com/basnet-tilak/Duniyani/crypto"
)

// Wallet stores a Duniyani keypair.
type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
}

// NewWallet creates a new wallet using secp256k1.
func NewWallet() *Wallet {
	priv, pub := crypto.NewKeyPair()
	return &Wallet{PrivateKey: priv, PublicKey: pub}
}

// GetAddress returns the Base58-check encoded address for the wallet.
func (w *Wallet) GetAddress() string {
	return crypto.PubKeyToAddress(w.PublicKey)
}

// SignTransaction signs each input of a transaction.
func (w *Wallet) SignTransaction(tx *core.Transaction, prevTxs map[string]core.Transaction) error {
	if tx.IsCoinbase() {
		return nil
	}

	txCopy := tx.TrimmedCopy()

	for inID, vin := range tx.Vin {
		prevTx, ok := prevTxs[string(vin.TxID)]
		if !ok {
			return fmt.Errorf("previous transaction not found for input %d", inID)
		}

		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash
		hashToSign := txCopy.Hash()

		signature, err := ecdsa.SignASN1(rand.Reader, w.PrivateKey, hashToSign)
		if err != nil {
			return fmt.Errorf("failed to sign transaction: %w", err)
		}

		tx.Vin[inID].Signature = signature
		tx.Vin[inID].PubKey = crypto.SerializePublicKey(w.PublicKey)
		txCopy.Vin[inID].PubKey = nil
	}

	tx.ID = tx.Hash()
	return nil
}

// VerifyTransaction verifies the signatures on each transaction input.
func VerifyTransaction(tx *core.Transaction, prevTxs map[string]core.Transaction) (bool, error) {
	if tx.IsCoinbase() {
		return true, nil
	}

	txCopy := tx.TrimmedCopy()

	for inID, vin := range tx.Vin {
		prevTx, ok := prevTxs[string(vin.TxID)]
		if !ok {
			return false, fmt.Errorf("previous transaction not found for input %d", inID)
		}

		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash
		hashToVerify := txCopy.Hash()

		publicKey, err := crypto.ParsePublicKey(vin.PubKey)
		if err != nil {
			return false, fmt.Errorf("invalid public key: %w", err)
		}

		if !ecdsa.VerifyASN1(publicKey, hashToVerify, vin.Signature) {
			return false, nil
		}

		txCopy.Vin[inID].PubKey = nil
	}

	return true, nil
}
