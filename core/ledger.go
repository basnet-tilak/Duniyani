package core

import (
	"bytes"
	"encoding/gob"

	"github.com/basnet-tilak/Duniyani/mldsa"
	"golang.org/x/crypto/sha3"
)

// TxInput references a previous UTXO and provides the ML-DSA signature.
type TxInput struct {
	TxID      []byte
	Vout      int
	PubKey    []byte // ML-DSA Public Key
	Signature []byte // ML-DSA Signature (~4.6KB for ML-DSA-87)
}

// TxOutput locks value (Drops) to a Duniyani Quantum address hash.
type TxOutput struct {
	Value      uint64
	PubKeyHash []byte
}

// Transaction represents a state transition in the UTXO model.
type Transaction struct {
	ID   []byte
	Vin  []TxInput
	Vout []TxOutput
}

// Hash computes the SHA3-384 hash of the transaction.
func (tx *Transaction) Hash() []byte {
	var buf bytes.Buffer
	// Avoid hashing signatures to allow for transaction ID stability
	tempTx := *tx
	for i := range tempTx.Vin {
		tempTx.Vin[i].Signature = nil
	}
	_ = gob.NewEncoder(&buf).Encode(tempTx)
	hash := sha3.Sum384(buf.Bytes())
	return hash[:]
}

// Verify validates all ML-DSA signatures in the transaction's inputs.
func (tx *Transaction) Verify() bool {
	if tx.IsCoinbase() {
		return true
	}

	txCopy := *tx
	for i := range txCopy.Vin {
		txCopy.Vin[i].Signature = nil
	}
	hashMsg := txCopy.Hash() // Message to verify is the hash of the tx

	for _, vin := range tx.Vin {
		pubKey, err := mldsa.NewPublicKey87(vin.PubKey)
		if err != nil {
			return false
		}
		if !mldsa.Verify(pubKey, hashMsg, vin.Signature) {
			return false
		}
	}
	return true
}

// BlockHeader uses SHA3-384 for quantum-resistant block linking.
type BlockHeader struct {
	PrevBlockHash  []byte
	MerkleRoot     []byte
	Timestamp      int64
	Nonce          uint64
	ComputeReceipt []byte // Hash of the AI workload processed
	EnclaveSig     []byte // ML-DSA signature of the ComputeReceipt by an authorized AI hardware enclave
}

type Block struct {
	Header       *BlockHeader
	Transactions []*Transaction
}

// Hash calculates the SHA3-384 hash of the block header.
func (b *Block) Hash() []byte {
	var buf bytes.Buffer
	_ = gob.NewEncoder(&buf).Encode(b.Header)
	hash := sha3.Sum384(buf.Bytes())
	return hash[:]
}

// IsCoinbase checks if the transaction is a newly minted reward.
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].TxID) == 0 && tx.Vin[0].Vout == -1
}
