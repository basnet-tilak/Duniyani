package consensus

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/basnet-tilak/Duniyani/core"
)

// ConsensusEngine defines the interface for a Duniyani consensus engine.
type ConsensusEngine interface {
	Mine(block *core.Block) error
	Verify(block *core.Block) bool
	DifficultyTarget() uint32
}

// PoUWEngine implements a Proof of Useful Work consensus engine.
type PoUWEngine struct {
	targetBits uint32
	target     *big.Int
}

// NewPoUWEngine creates a new Proof of Useful Work engine with a bit-based target.
func NewPoUWEngine(targetBits uint32) *PoUWEngine {
	engine := &PoUWEngine{targetBits: targetBits}
	engine.calculateTarget()
	return engine
}

func (pouw *PoUWEngine) DifficultyTarget() uint32 {
	return pouw.targetBits
}

func (pouw *PoUWEngine) calculateTarget() {
	pouw.target = big.NewInt(1)
	pouw.target.Lsh(pouw.target, uint(256-pouw.targetBits))
}

// Mine searches for a nonce and compute receipt that satisfy PoUW difficulty.
func (pouw *PoUWEngine) Mine(block *core.Block) error {
	var hashInt big.Int

	for nonce := uint64(0); nonce < ^uint64(0); nonce++ {
		block.Header.Nonce = nonce
		block.Header.ComputeReceipt = pouw.buildReceipt(block.Header, nonce)

		hashBytes := block.Header.Hash()
		hashInt.SetBytes(hashBytes[:])

		if hashInt.Cmp(pouw.target) == -1 {
			return nil
		}
	}

	return fmt.Errorf("mining failed: could not find a valid nonce")
}

func (pouw *PoUWEngine) buildReceipt(header *core.BlockHeader, nonce uint64) []byte {
	buffer := make([]byte, 8)
	binary.LittleEndian.PutUint64(buffer, nonce)
	payload := append(header.PrevBlockHash, header.MerkleRoot...)
	payload = append(payload, buffer...)
	receipt := sha256.Sum256(payload)
	return receipt[:]
}

// Verify validates that the block meets the Proof of Useful Work target.
func (pouw *PoUWEngine) Verify(block *core.Block) bool {
	var hashInt big.Int
	hashBytes := block.Header.Hash()
	hashInt.SetBytes(hashBytes[:])
	return hashInt.Cmp(pouw.target) == -1
}
