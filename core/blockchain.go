package core

import (
	"bytes"
	"fmt"

	"github.com/basnet-tilak/Duniyani/database"
)

// Blockchain represents the immutable chain and associated state.
type Blockchain struct {
	lastBlockHash []byte
	db            *database.Database
	utxoSet       *UTXOSet
}

// CreateBlockchain creates a new blockchain and stores the genesis block.
func CreateBlockchain(db *database.Database, genesis *Block) (*Blockchain, error) {
	blockBytes, err := genesis.Serialize()
	if err != nil {
		return nil, err
	}

	genesisHash := genesis.Header.Hash()
	if err := db.Put(database.BlocksBucket, genesisHash[:], blockBytes); err != nil {
		return nil, err
	}

	if err := db.Put(database.BlocksBucket, []byte("l"), genesisHash[:]); err != nil {
		return nil, err
	}

	utxoSet := NewUTXOSet(db)
	if err := utxoSet.Update(genesis); err != nil {
		return nil, err
	}

	return &Blockchain{lastBlockHash: genesisHash[:], db: db, utxoSet: utxoSet}, nil
}

// LoadBlockchain loads the chain metadata and UTXO manager.
func LoadBlockchain(db *database.Database) (*Blockchain, error) {
	lastHash, err := db.Get(database.BlocksBucket, []byte("l"))
	if err != nil {
		return nil, err
	}

	utxoSet := NewUTXOSet(db)
	return &Blockchain{lastBlockHash: lastHash, db: db, utxoSet: utxoSet}, nil
}

// AddBlock appends a block and updates the active UTXO set.
func (bc *Blockchain) AddBlock(block *Block) error {
	serialized, err := block.Serialize()
	if err != nil {
		return err
	}

	blockHash := block.Header.Hash()
	if err := bc.db.Put(database.BlocksBucket, blockHash[:], serialized); err != nil {
		return err
	}

	if err := bc.db.Put(database.BlocksBucket, []byte("l"), blockHash[:]); err != nil {
		return err
	}

	if bc.utxoSet == nil {
		bc.utxoSet = NewUTXOSet(bc.db)
	}

	if err := bc.utxoSet.Update(block); err != nil {
		return err
	}

	bc.lastBlockHash = blockHash[:]
	return nil
}

// GetBlock retrieves a block by hash.
func (bc *Blockchain) GetBlock(hash []byte) (*Block, error) {
	data, err := bc.db.Get(database.BlocksBucket, hash)
	if err != nil {
		return nil, err
	}
	return DeserializeBlock(data)
}

// GetLastBlockHash returns the current chain tip hash.
func (bc *Blockchain) GetLastBlockHash() []byte {
	return bc.lastBlockHash
}

// Height estimates the current chain height by scanning blocks.
func (bc *Blockchain) Height() uint64 {
	var height uint64
	_ = bc.db.Iterate(database.BlocksBucket, func(key, _ []byte) error {
		if bytes.Equal(key, []byte("l")) {
			return nil
		}
		height++
		return nil
	})
	return height
}

// GetTransaction finds a transaction by ID in the block history.
func (bc *Blockchain) GetTransaction(txID []byte) (*Transaction, error) {
	var result *Transaction
	err := bc.db.Iterate(database.BlocksBucket, func(key, value []byte) error {
		if bytes.Equal(key, []byte("l")) {
			return nil
		}

		block, err := DeserializeBlock(value)
		if err != nil {
			return err
		}

		for _, tx := range block.Transactions {
			if bytes.Equal(tx.ID, txID) {
				result = tx
				return nil
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("transaction not found")
	}
	return result, nil
}
