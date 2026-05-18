package wallet

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/basnet-tilak/Duniyani/core"
	"github.com/basnet-tilak/Duniyani/crypto"
	"golang.org/x/crypto/sha3"
)

// Wallet securely stores ECDSA keypairs for the Duniyani network.
type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
}

// NewWallet creates a new wallet.
func NewWallet() *Wallet {
	priv, pub := crypto.NewKeyPair()
	return &Wallet{
		PrivateKey: priv,
		PublicKey:  pub,
	}
}

// GetAddress returns the encoded Duniyani address.
func (w *Wallet) GetAddress() string {
	pubKeyBytes := crypto.SerializePublicKey(w.PublicKey)
	hash := sha3.Sum256(pubKeyBytes)
	return "DQ" + hex.EncodeToString(hash[:])
}

// Sign generates an ECDSA signature for a given message.
func (w *Wallet) Sign(msg []byte) ([]byte, error) {
	return ecdsa.SignASN1(rand.Reader, w.PrivateKey, msg)
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
			return fmt.Errorf("previous transaction isn't found for input %d", inID)
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

	tx.ID = txCopy.Hash()
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
			return false, fmt.Errorf("previous transaction isn't found for input %d", inID)
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
