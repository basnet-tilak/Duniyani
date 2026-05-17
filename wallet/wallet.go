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

// Sign generates an ML-DSA-87 signature for a given message.
func (w *Wallet) Sign(msg []byte) ([]byte, error) {
	return mldsa.Sign(w.PrivateKey, msg), nil
}
