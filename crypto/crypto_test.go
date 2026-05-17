package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddressGeneration(t *testing.T) {
	t.Parallel()

	// 1. Create a new key pair
	_, pubKey := NewKeyPair()

	// 2. Generate an address
	address := PubKeyToAddress(pubKey)

	// 3. Check that the address has the correct prefix
	require.True(t, len(address) > 0, "Address should not be empty")
	assert.Equal(t, "D", string(address[0]), "Duniyani addresses should start with 'D'")

	// 4. Decode the address back to the public key hash
	pubKeyHashFromAddr := AddressToPubKeyHash(address)

	// 5. Verify that the decoded hash matches the original
	expectedPubKeyHash := HashPubKey(SerializeCompressed(pubKey))
	assert.Equal(t, expectedPubKeyHash, pubKeyHashFromAddr, "Decoded public key hash should match the original")
}

func TestInvalidAddress(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		address string
	}{
		{"Invalid Base58", "D123abcxyz@#$"},
		{"Invalid Checksum", "D6U1sy39L8z4wzH26T3nZzYd2n9XwXvYq"}, // Valid format, wrong checksum
		{"Wrong Version", "16U1sy39L8z4wzH26T3nZzYd2n9XwXvYq"},    // Bitcoin address
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// We expect the function to panic for invalid addresses
			assert.Panics(t, func() {
				AddressToPubKeyHash(tc.address)
			}, "Should panic for an invalid address")
		})
	}
}
