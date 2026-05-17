package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/basnet-tilak/Duniyani/database"
)

// TxOutput represents a transaction output in the UTXO model.
type TxOutput struct {
	Value      int64  // Value in Drops (1 DNY = 100,000,000 Drops)
	PubKeyHash []byte // Hash of the recipient's public key
}

// TxInput represents a transaction input in the UTXO model.
type TxInput struct {
	TxID      []byte // ID of the referenced transaction
	Vout      int    // Output index in that transaction
	Signature []byte // Signature authorizing the spend
	PubKey    []byte // Public key of the spender
}

// Transaction defines a Duniyani transaction.
type Transaction struct {
	ID        []byte
	Vin       []TxInput
	Vout      []TxOutput
	Timestamp int64
}

func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].TxID) == 0 && tx.Vin[0].Vout == -1
}

func (tx *Transaction) Serialize() ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(tx); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (tx *Transaction) Hash() []byte {
	var hash [32]byte
	txCopy := tx.TrimmedCopy()
	txCopy.Timestamp = tx.Timestamp
	txCopy.ID = nil

	serialized, err := txCopy.Serialize()
	if err != nil {
		panic(err)
	}

	hash = sha256.Sum256(serialized)
	return hash[:]
}

func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TxInput
	for _, vin := range tx.Vin {
		inputs = append(inputs, TxInput{TxID: vin.TxID, Vout: vin.Vout, Signature: nil, PubKey: nil})
	}

	var outputs []TxOutput
	for _, vout := range tx.Vout {
		outputs = append(outputs, TxOutput{Value: vout.Value, PubKeyHash: vout.PubKeyHash})
	}

	return Transaction{ID: tx.ID, Vin: inputs, Vout: outputs, Timestamp: tx.Timestamp}
}

// BlockHeader defines the header for a Duniyani block.
type BlockHeader struct {
	Version          uint32
	PrevBlockHash    []byte
	MerkleRoot       []byte
	Timestamp        int64
	DifficultyTarget uint32
	Nonce            uint64
	ComputeReceipt   []byte
}

// Hash computes the SHA-256 hash of the block header.
func (h *BlockHeader) Hash() [32]byte {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(h); err != nil {
		panic(err)
	}
	return sha256.Sum256(buf.Bytes())
}

// Block is a Duniyani blockchain block.
type Block struct {
	Header       *BlockHeader
	Transactions []*Transaction
}

func NewBlock(transactions []*Transaction, prevBlockHash []byte, version uint32, difficulty uint32) *Block {
	header := &BlockHeader{
		Version:          version,
		PrevBlockHash:    prevBlockHash,
		Timestamp:        time.Now().Unix(),
		DifficultyTarget: difficulty,
	}

	block := &Block{Header: header, Transactions: transactions}
	block.Header.MerkleRoot = block.ComputeMerkleRoot()
	return block
}

func (b *Block) ComputeMerkleRoot() []byte {
	var txHashes [][]byte
	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.Hash())
	}

	if len(txHashes) == 0 {
		empty := sha256.Sum256([]byte{})
		return empty[:]
	}

	mTree := NewMerkleTree(txHashes)
	return mTree.RootNode.Data
}

func (b *Block) Serialize() ([]byte, error) {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	if err := encoder.Encode(b); err != nil {
		return nil, err
	}
	return result.Bytes(), nil
}

func DeserializeBlock(data []byte) (*Block, error) {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&block); err != nil {
		return nil, err
	}
	return &block, nil
}

// UTXO represents an unspent transaction output pointer.
type UTXO struct {
	TxID   []byte
	Index  int
	Output TxOutput
}

// UTXOSet manages the active UTXO state.
type UTXOSet struct {
	db *database.Database
	mu sync.RWMutex
}

// NewUTXOSet creates a UTXO set manager backed by the database.
func NewUTXOSet(db *database.Database) *UTXOSet {
	return &UTXOSet{db: db}
}

func (u *UTXOSet) utxoKey(txID []byte, index int) []byte {
	return []byte(fmt.Sprintf("%x:%d", txID, index))
}

func (u *UTXOSet) PutUTXO(txID []byte, index int, output TxOutput) error {
	key := u.utxoKey(txID, index)
	serialized, err := output.Serialize()
	if err != nil {
		return err
	}
	return u.db.Put(database.ChainStateBucket, key, serialized)
}

func (u *UTXOSet) DeleteUTXO(txID []byte, index int) error {
	key := u.utxoKey(txID, index)
	return u.db.Delete(database.ChainStateBucket, key)
}

func (u *UTXOSet) GetBalance(pubKeyHash []byte) (int64, error) {
	u.mu.RLock()
	defer u.mu.RUnlock()

	var balance int64
	if err := u.db.Iterate(database.ChainStateBucket, func(_, value []byte) error {
		output, err := DeserializeTxOutput(value)
		if err != nil {
			return err
		}
		if bytes.Equal(output.PubKeyHash, pubKeyHash) {
			balance += output.Value
		}
		return nil
	}); err != nil {
		return 0, err
	}

	return balance, nil
}

func (u *UTXOSet) FindSpendableOutputs(pubKeyHash []byte, amount int64) (int64, map[string][]int, error) {
	u.mu.RLock()
	defer u.mu.RUnlock()

	accumulated := int64(0)
	unspentOutputs := make(map[string][]int)

	if err := u.db.Iterate(database.ChainStateBucket, func(key, value []byte) error {
		output, err := DeserializeTxOutput(value)
		if err != nil {
			return err
		}
		if bytes.Equal(output.PubKeyHash, pubKeyHash) {
			accumulated += output.Value
			unspentOutputs[string(key)] = append(unspentOutputs[string(key)], parseOutputIndex(key))
		}
		return nil
	}); err != nil {
		return 0, nil, err
	}

	return accumulated, unspentOutputs, nil
}

func parseOutputIndex(key []byte) int {
	parts := bytes.Split(key, []byte(":"))
	if len(parts) != 2 {
		return 0
	}
	index, err := strconv.Atoi(string(parts[1]))
	if err != nil {
		return 0
	}
	return index
}

func (u *UTXOSet) Update(block *Block) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	for _, tx := range block.Transactions {
		for outIdx, out := range tx.Vout {
			if err := u.PutUTXO(tx.ID, outIdx, out); err != nil {
				return err
			}
		}

		if tx.IsCoinbase() {
			continue
		}

		for _, vin := range tx.Vin {
			if err := u.DeleteUTXO(vin.TxID, vin.Vout); err != nil {
				return err
			}
		}
	}

	return nil
}

func (u *UTXOSet) Reindex() error {
	u.mu.Lock()
	defer u.mu.Unlock()

	// In production this would rebuild the chain state from block history.
	// For this blueprint, we preserve existing UTXOs and initialize the dataset.
	return nil
}

func (u *UTXOSet) GetUTXO(txID []byte, index int) (*TxOutput, error) {
	key := u.utxoKey(txID, index)
	value, err := u.db.Get(database.ChainStateBucket, key)
	if err != nil {
		return nil, err
	}
	return DeserializeTxOutput(value)
}

func (out *TxOutput) Serialize() ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DeserializeTxOutput(data []byte) (*TxOutput, error) {
	var output TxOutput
	decoder := gob.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&output); err != nil {
		return nil, err
	}
	return &output, nil
}
