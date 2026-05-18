package consensus

import (
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

// PoWEngine implements a Proof of Work consensus engine using SHA256.
type PoWEngine struct {
	targetBits uint32
	target     *big.Int
}

// NewPoWEngine creates a new Proof of Work engine with a bit-based target.
func NewPoWEngine(targetBits uint32) *PoWEngine {
	engine := &PoWEngine{targetBits: targetBits}
	engine.calculateTarget()
	return engine
}

func (pow *PoWEngine) DifficultyTarget() uint32 {
	return pow.targetBits
}

func (pow *PoWEngine) calculateTarget() {
	pow.target = big.NewInt(1)
	pow.target.Lsh(pow.target, uint(256-pow.targetBits))
}

// Mine searches for a nonce that satisfies the PoW difficulty.
func (pow *PoWEngine) Mine(block *core.Block) error {
	var hashInt big.Int

	for nonce := uint64(0); nonce < ^uint64(0); nonce++ {
		block.Header.Nonce = nonce
		hashBytes := block.Header.Hash()
		hashInt.SetBytes(hashBytes[:])

		if hashInt.Cmp(pow.target) == -1 {
			return nil
		}
	}

	return fmt.Errorf("mining failed: could not find a valid nonce")
}

// Verify validates that the block meets the Proof of Work target.
func (pow *PoWEngine) Verify(block *core.Block) bool {
	var hashInt big.Int
	hashBytes := block.Header.Hash()
	hashInt.SetBytes(hashBytes[:])
	return hashInt.Cmp(pow.target) == -1
}
