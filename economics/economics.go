package economics

import (
	"bytes"
	"encoding/gob"
	"math"

	"github.com/basnet-tilak/Duniyani/core"
	"github.com/basnet-tilak/Duniyani/crypto"
)

const (
	DropsPerDNY           = 100_000_000
	GenesisCoinbaseAmount = 10_000_000 * DropsPerDNY
	InitialBlockReward    = 50 * DropsPerDNY
	HalvingInterval       = 210_000
	FeePerByte            = 10
	CongestionFeePerTx    = 100
	MinimumNetworkFee     = 100
)

// BlockReward returns the reward for a given block height.
func BlockReward(height uint64) int64 {
	halvings := height / HalvingInterval
	if halvings >= 64 {
		return 0
	}
	reward := float64(InitialBlockReward) / math.Pow(2, float64(halvings))
	return int64(reward)
}

// NewCoinbaseTx creates a new coinbase transaction with a dynamic reward.
func NewCoinbaseTx(toAddress, data string, height uint64) *core.Transaction {
	if data == "" {
		data = "Duniyani miner reward"
	}

	txIn := core.TxInput{
		TxID:      []byte{},
		Vout:      -1,
		Signature: nil,
		PubKey:    []byte(data),
	}

	pubKeyHash := crypto.AddressToPubKeyHash(toAddress)
	reward := BlockReward(height)
	if height == 0 {
		reward = GenesisCoinbaseAmount
	}

	txOut := core.TxOutput{
		Value:      reward,
		PubKeyHash: pubKeyHash,
	}

	tx := &core.Transaction{
		Vin:       []core.TxInput{txIn},
		Vout:      []core.TxOutput{txOut},
		Timestamp: 0,
	}
	tx.ID = tx.Hash()
	return tx
}

// CreateGenesisBlock creates the network genesis block.
func CreateGenesisBlock(genesisAddress string) *core.Block {
	coinbase := NewCoinbaseTx(genesisAddress, "Duniyani Genesis Block", 0)
	coinbase.Vout[0].Value = GenesisCoinbaseAmount
	coinbase.ID = coinbase.Hash()

	return core.NewBlock([]*core.Transaction{coinbase}, []byte{}, 1, 0)
}

// CalculateTransactionFee returns a fee influenced by tx size and mempool congestion.
func CalculateTransactionFee(tx *core.Transaction, pendingTransactions int) (int64, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(tx); err != nil {
		return 0, err
	}

	size := int64(len(buf.Bytes()))
	fee := size*FeePerByte + int64(pendingTransactions)*CongestionFeePerTx
	if fee < MinimumNetworkFee {
		fee = MinimumNetworkFee
	}
	return fee, nil
}
