package state

import (
	"fmt"
	"sync"

	"github.com/basnet-tilak/Duniyani/core"
	"github.com/basnet-tilak/Duniyani/database"
	"github.com/basnet-tilak/Duniyani/economics"
)

// Mempool represents a thread-safe pool of unconfirmed transactions.
type Mempool struct {
	mu           sync.RWMutex
	Transactions map[string]*core.Transaction
	db           *database.Database
}

// NewMempool creates a new mempool.
func NewMempool(db *database.Database) *Mempool {
	return &Mempool{
		Transactions: make(map[string]*core.Transaction),
		db:           db,
	}
}

// Add adds a transaction to the mempool after validation.
func (m *Mempool) Add(tx *core.Transaction) error {
	if tx.IsCoinbase() {
		return fmt.Errorf("coinbase transactions cannot be added to the mempool")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	txID := string(tx.ID)
	if _, ok := m.Transactions[txID]; ok {
		return fmt.Errorf("transaction %s already in mempool", txID)
	}

	// Track UTXOs already spent by transactions currently in the mempool
	spentInMempool := make(map[string]bool)
	for _, memTx := range m.Transactions {
		for _, vin := range memTx.Vin {
			spentInMempool[fmt.Sprintf("%x:%d", vin.TxID, vin.Vout)] = true
		}
	}

	for _, vin := range tx.Vin {
		utxoKey := fmt.Sprintf("%x:%d", vin.TxID, vin.Vout)
		if spentInMempool[utxoKey] {
			return fmt.Errorf("potentially double-spend: UTXO %s is already spent in mempool", utxoKey)
		}
	}

	// Database-dependent validation
	if m.db != nil {
		var inputSum int64
		for _, vin := range tx.Vin {
			utxoKey := fmt.Sprintf("%x:%d", vin.TxID, vin.Vout)
			key := []byte(utxoKey)
			val, err := m.db.Get(database.ChainStateBucket, key)
			if err != nil {
				return fmt.Errorf("input %s not found in UTXO set (already spent or invalid)", utxoKey)
			}

			utxo, err := core.DeserializeTxOutput(val)
			if err != nil {
				return fmt.Errorf("failed to deserialize UTXO %s", utxoKey)
			}
			inputSum += utxo.Value
		}

		var outputSum int64
		for _, vout := range tx.Vout {
			outputSum += vout.Value
		}

		providedFee := inputSum - outputSum
		if providedFee < 0 {
			return fmt.Errorf("transaction %s has negative fee: outputs exceed inputs", txID)
		}

		requiredFee, err := economics.CalculateTransactionFee(tx, len(m.Transactions))
		if err != nil {
			return fmt.Errorf("failed to calculate the required fee: %w", err)
		}

		if providedFee < requiredFee {
			return fmt.Errorf("transaction %s fee is too low: provided %d, required %d", txID, providedFee, requiredFee)
		}
	}

	m.Transactions[txID] = tx
	return nil
}

// GetAll returns a slice of all transactions in the mempool.
func (m *Mempool) GetAll() []*core.Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	txs := make([]*core.Transaction, 0, len(m.Transactions))
	for _, tx := range m.Transactions {
		txs = append(txs, tx)
	}
	return txs
}

// Remove removes a transaction from the mempool.
func (m *Mempool) Remove(txID []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.Transactions, string(txID))
}

// Clear empties the mempool.
func (m *Mempool) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Transactions = make(map[string]*core.Transaction)
}
