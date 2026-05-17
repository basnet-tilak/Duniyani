package crypto

import (
	"crypto/mldsa"
	"crypto/mlkem"
	"crypto/rand"
	"crypto/sha256"

	"github.com/mr-tron/base58"
	"golang.org/x/crypto/sha3"
)

// GenerateSignatureKeyPair uses ML-DSA-87 for NIST Level 5 post-quantum digital signatures.
func GenerateSignatureKeyPair() (*mldsa.PrivateKey87, error) {
	return mldsa.GenerateKey87(rand.Reader)
}

// GenerateKEMKeyPair uses ML-KEM-1024 for NIST Level 5 post-quantum key encapsulation.
func GenerateKEMKeyPair() (*mlkem.DecapsulationKey1024, error) {
	return mlkem.GenerateKey1024()
}

// PubKeyToAddress hashes an ML-DSA public key with SHA3-256 and Base58Check encodes it
// with a 'DQ' (Duniyani Quantum) prefix.
func PubKeyToAddress(pubKeyBytes []byte) string {
	// 1. Hash the public key with SHA3-256
	hash := sha3.Sum256(pubKeyBytes)

	// 2. Compute checksum using double SHA-256
	checksum := sha256.Sum256(hash[:])
	checksum = sha256.Sum256(checksum[:])

	// 3. Append first 4 bytes of checksum to the hash
	payload := append(hash[:], checksum[:4]...)

	// 4. Base58 encode and prepend DQ
	return "DQ" + base58.Encode(payload)
}

// VerifyMLDSASignature verifies an ML-DSA-87 signature.
func VerifyMLDSASignature(pubKeyBytes, msg, sig []byte) bool {
	pubKey, err := mldsa.ParsePublicKey87(pubKeyBytes)
	if err != nil {
		return false
	}

	// mldsa.Verify87 returns a non-nil error if the signature is invalid.
	if err := mldsa.Verify87(pubKey, msg, sig); err != nil {
		return false
	}
	return true
}
