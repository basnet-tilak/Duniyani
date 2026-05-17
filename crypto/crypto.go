package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"math/big"
)

const (
	addressVersion     = byte(0x1e)
	addressChecksumLen = 4
)

var secp256k1 elliptic.Curve = elliptic.P256()

var p256SPKIPrefix []byte

func init() {
	// Dynamically generate the SPKI prefix for P-256 to avoid using deprecated ecdsa/elliptic fields.
	priv, err := ecdsa.GenerateKey(secp256k1, rand.Reader)
	if err != nil {
		panic(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		panic(err)
	}
	// The uncompressed SEC 1 point is always 65 bytes for P-256
	prefixLen := len(der) - 65
	p256SPKIPrefix = der[:prefixLen:prefixLen] // force cap = len to prevent append races
}

func NewKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	privKey, err := ecdsa.GenerateKey(secp256k1, rand.Reader)
	if err != nil {
		panic(err)
	}
	return privKey, &privKey.PublicKey
}

func SerializePublicKey(pubKey *ecdsa.PublicKey) []byte {
	ecdhKey, err := pubKey.ECDH()
	if err != nil {
		panic(err)
	}
	return ecdhKey.Bytes()
}

func SerializeCompressed(pubKey *ecdsa.PublicKey) []byte {
	ecdhKey, err := pubKey.ECDH()
	if err != nil {
		panic(err)
	}
	bytes := ecdhKey.Bytes()
	byteLen := (len(bytes) - 1) / 2
	compressed := make([]byte, 1+byteLen)

	// The last byte of the uncompressed serialization is the last byte of Y.
	// We use it to determine the sign (odd/even).
	if bytes[len(bytes)-1]&1 == 1 {
		compressed[0] = 0x03
	} else {
		compressed[0] = 0x02
	}
	copy(compressed[1:], bytes[1:1+byteLen])
	return compressed
}

func ParsePublicKey(data []byte) (*ecdsa.PublicKey, error) {
	if len(data) != 65 || data[0] != 4 {
		return nil, fmt.Errorf("invalid public key length or format")
	}

	der := append(p256SPKIPrefix, data...)
	pub, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	ecdsaPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an ECDSA public key")
	}
	return ecdsaPub, nil
}

func HashPubKey(pubKey []byte) []byte {
	publicSHA256 := sha256.Sum256(pubKey)
	// ripemd160 is deprecated; replacing with double-SHA256 for a modern baseline
	hash2 := sha256.Sum256(publicSHA256[:])
	return hash2[:20] // Truncate to 20 bytes to preserve 25-byte address length and "D" prefix
}

func checksum(payload []byte) []byte {
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])
	return secondSHA[:addressChecksumLen]
}

func PubKeyToAddress(pubKey *ecdsa.PublicKey) string {
	pubKeyHash := HashPubKey(SerializeCompressed(pubKey))
	versionedPayload := append([]byte{addressVersion}, pubKeyHash...)
	chksum := checksum(versionedPayload)
	fullPayload := append(versionedPayload, chksum...)
	return encodeBase58(fullPayload)
}

func AddressToPubKeyHash(address string) []byte {
	decoded := decodeBase58(address)
	if len(decoded) < addressChecksumLen+1 {
		panic("invalid address: the decoded length is too short")
	}

	version := decoded[0]
	pubKeyHash := decoded[1 : len(decoded)-addressChecksumLen]
	actualChecksum := decoded[len(decoded)-addressChecksumLen:]

	if version != addressVersion {
		panic("invalid address: wrong version byte")
	}

	targetChecksum := checksum(append([]byte{version}, pubKeyHash...))
	if !equalBytes(targetChecksum, actualChecksum) {
		panic("invalid address: checksum does not match")
	}

	return pubKeyHash
}

func encodeBase58(input []byte) string {
	alphabet := []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")
	intData := new(big.Int).SetBytes(input)
	if intData.Sign() == 0 {
		return string(alphabet[0])
	}
	var encoded []byte
	base := big.NewInt(58)
	zero := big.NewInt(0)
	for intData.Cmp(zero) > 0 {
		dvd := new(big.Int)
		intData.DivMod(intData, base, dvd)
		encoded = append([]byte{alphabet[dvd.Int64()]}, encoded...)
	}
	for _, b := range input {
		if b != 0 {
			break
		}
		encoded = append([]byte{alphabet[0]}, encoded...)
	}
	return string(encoded)
}

func decodeBase58(input string) []byte {
	alphabet := []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")
	result := big.NewInt(0)
	base := big.NewInt(58)
	for _, r := range []byte(input) {
		digit := indexByte(alphabet, r)
		if digit < 0 {
			panic("invalid base58 character")
		}
		result.Mul(result, base)
		result.Add(result, big.NewInt(int64(digit)))
	}

	decoded := result.Bytes()
	for i := 0; i < len(input) && input[i] == alphabet[0]; i++ {
		decoded = append([]byte{0}, decoded...)
	}
	return decoded
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func indexByte(slice []byte, value byte) int {
	for i, b := range slice {
		if b == value {
			return i
		}
	}
	return -1
}
