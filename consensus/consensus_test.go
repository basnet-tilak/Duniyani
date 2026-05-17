package consensus

import (
	"testing"

	"github.com/basnet-tilak/Duniyani/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoUWEngine_MineAndVerify(t *testing.T) {
	t.Parallel()

	// Use a low difficulty for fast testing
	const testDifficulty = 8
	cEng := NewPoUWEngine(testDifficulty)

	// 1. Test Mining a block
	block := core.NewBlock([]*core.Transaction{}, []byte{}, 1, testDifficulty)

	err := cEng.Mine(block)
	require.NoError(t, err, "Mining should not produce an error")

	// 2. Test Verifying the valid, mined block
	assert.True(t, cEng.Verify(block), "A correctly mined block should be valid")

	// 3. Test Verifying an invalid block
	// Create another engine with a higher difficulty
	higherDifficultyCEng := NewPoUWEngine(testDifficulty + 4)
	assert.False(t, higherDifficultyCEng.Verify(block), "Block should be invalid against a higher difficulty")

	// Tamper with the block after mining
	block.Header.Nonce++
	assert.False(t, cEng.Verify(block), "A tampered block should be invalid")
}
