package types

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
)

// Transaction represents a standard transaction in the blockchain.
type Transaction struct {
	From      []byte
	To        []byte
	Value     uint64
	Timestamp int64
	Signature []byte
	hash      []byte
}

// Hash calculates and returns the SHA256 hash of the transaction.
func (t *Transaction) Hash() []byte {
	if t.hash != nil {
		return t.hash
	}
	var buf bytes.Buffer
	tempTx := *t
	tempTx.Signature = nil
	tempTx.hash = nil
	if err := gob.NewEncoder(&buf).Encode(tempTx); err != nil {
		panic(err)
	}
	hash := sha256.Sum256(buf.Bytes())
	t.hash = hash[:]
	return t.hash
}

// Encode serializes the transaction using gob.
func (t *Transaction) Encode() ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(t); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// BlockHeader defines the structure of a block's header.
type BlockHeader struct {
	Height           uint64
	Timestamp        int64
	PrevStateRoot    []byte
	TxRoot           []byte
	ConsensusPayload []byte
}

// Block represents a full block in the blockchain.
type Block struct {
	Header       *BlockHeader
	Transactions []*Transaction
	hash         []byte
}

// NewBlock creates a new block, calculating the Merkle root and block hash.
func NewBlock(header *BlockHeader, transactions []*Transaction) *Block {
	header.TxRoot = CalculateMerkleRoot(transactions)
	b := &Block{
		Header:       header,
		Transactions: transactions,
	}
	b.hash = b.calculateHash()
	return b
}

// Hash returns the cached hash of the block.
func (b *Block) Hash() []byte {
	return b.hash
}

// calculateHash computes the SHA256 hash of the block header.
func (b *Block) calculateHash() []byte {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(b.Header); err != nil {
		panic(err)
	}
	hash := sha256.Sum256(buf.Bytes())
	return hash[:]
}

// Encode serializes the block using gob.
func (b *Block) Encode() ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(b); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DecodeBlock deserializes a block from a byte slice.
func DecodeBlock(data []byte) (*Block, error) {
	var block Block
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&block); err != nil {
		return nil, err
	}
	return &block, nil
}

// DecodeTransaction deserializes a transaction from a byte slice.
func DecodeTransaction(data []byte) (*Transaction, error) {
	var tx Transaction
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&tx); err != nil {
		return nil, err
	}
	return &tx, nil
}

// CalculateMerkleRoot computes the Merkle root for a slice of transactions.
func CalculateMerkleRoot(transactions []*Transaction) []byte {
	if len(transactions) == 0 {
		hash := sha256.Sum256([]byte{})
		return hash[:]
	}
	var txHashes [][]byte
	for _, tx := range transactions {
		txHashes = append(txHashes, tx.Hash())
	}
	for len(txHashes) > 1 {
		if len(txHashes)%2 != 0 {
			txHashes = append(txHashes, txHashes[len(txHashes)-1])
		}
		var newLevel [][]byte
		for i := 0; i < len(txHashes); i += 2 {
			combined := append(txHashes[i], txHashes[i+1]...)
			hash := sha256.Sum256(combined)
			newLevel = append(newLevel, hash[:])
		}
		txHashes = newLevel
	}
	return txHashes[0]
}
