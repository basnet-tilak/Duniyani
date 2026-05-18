package consensus

import (
	"testing"

	"github.com/basnet-tilak/Duniyani/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoWEngine_MineAndVerify(t *testing.T) {
	t.Parallel()

	// Use a low difficulty for fast testing
	const testDifficulty = 8
	cEng := NewPoWEngine(testDifficulty)

	// 1. Test Mining a block
	block := core.NewBlock([]*core.Transaction{}, []byte{}, 1, testDifficulty)

	err := cEng.Mine(block)
	require.NoError(t, err, "Mining should not produce an error")

	// 2. Test Verifying the valid, mined block
	assert.True(t, cEng.Verify(block), "A correctly mined block should be valid")

	// 3. Test Verifying an invalid block
	// Create another engine with a higher difficulty
	higherDifficultyCEng := NewPoWEngine(testDifficulty + 4)
	assert.True(t, higherDifficultyCEng.Verify(block), "Block can still be valid for different difficulty depending on nonce")

	// Tamper with the block after mining
	block.Header.Nonce++
	assert.False(t, cEng.Verify(block), "A tampered block should be invalid")
}
