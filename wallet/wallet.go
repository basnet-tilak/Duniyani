package wallet

import (
	"encoding/hex"
	"log"

	"github.com/basnet-tilak/Duniyani/mldsa"
	"golang.org/x/crypto/sha3"
)

// Wallet securely stores post-quantum keypairs for the Duniyani network.
type Wallet struct {
	PrivateKey *mldsa.PrivateKey
	PublicKey  *mldsa.PublicKey
}

// NewWallet creates a new quantum-secure wallet.
func NewWallet() *Wallet {
	priv, err := mldsa.GenerateKey87(nil)
	if err != nil {
		log.Panicf("Failed to generate ML-DSA keypair: %v", err)
	}

	return &Wallet{
		PrivateKey: priv,
		PublicKey:  priv.PublicKey(),
	}
}

// GetAddress returns the encoded Duniyani Quantum (DQ) address.
func (w *Wallet) GetAddress() string {
	hash := sha3.Sum256(w.PublicKey.Bytes())
	return "DQ" + hex.EncodeToString(hash[:])
}

<<<<<<< HEAD
// Sign generates an ML-DSA-87 signature for a given message.
func (w *Wallet) Sign(msg []byte) ([]byte, error) {
	return mldsa.Sign(w.PrivateKey, msg), nil
=======
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
>>>>>>> d9197b0be1326238bc1fa836f417cbdcb4125ebe
}
