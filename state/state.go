package state

import (
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/basnet-tilak/Duniyani/core"
)

// StateDB defines a minimal state persistence interface.
type StateDB interface {
	Get(key []byte) ([]byte, error)
	Put(key, value []byte) error
	Delete(key []byte) error
}

// Mempool holds pending transactions for the node.
type Mempool struct {
	mu           sync.RWMutex
	transactions map[string]*core.Transaction
}

// NewMempool creates a new state-aware mempool.
func NewMempool() *Mempool {
	return &Mempool{transactions: make(map[string]*core.Transaction)}
}

// Add inserts a transaction after basic validation.
func (m *Mempool) Add(tx *core.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if tx == nil || len(tx.Vin) == 0 || len(tx.Vout) == 0 {
		return fmt.Errorf("invalid transaction")
	}

	txID := hex.EncodeToString(tx.Hash())
	if _, exists := m.transactions[txID]; exists {
		return fmt.Errorf("transaction %s already in mempool", txID)
	}

	m.transactions[txID] = tx
	return nil
}

// GetAll returns all mempool transactions.
func (m *Mempool) GetAll() []*core.Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	txs := make([]*core.Transaction, 0, len(m.transactions))
	for _, tx := range m.transactions {
		txs = append(txs, tx)
	}
	return txs
}

// Clear removes all transactions from the mempool.
func (m *Mempool) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transactions = make(map[string]*core.Transaction)
}
