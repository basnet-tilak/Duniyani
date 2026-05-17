package crypto

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
)

const (
	addressVersion     = byte(0x1e)
	addressChecksumLen = 4
)

var secp256k1 elliptic.Curve = elliptic.P256()

func NewKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	privKey, err := ecdsa.GenerateKey(secp256k1, rand.Reader)
	if err != nil {
		panic(err)
	}
	return privKey, &privKey.PublicKey
}

func SerializePublicKey(pubKey *ecdsa.PublicKey) []byte {
	byteLen := (pubKey.Curve.Params().BitSize + 7) >> 3
	ret := make([]byte, 1+2*byteLen)
	ret[0] = 4 // uncompressed point format
	pubKey.X.FillBytes(ret[1 : 1+byteLen])
	pubKey.Y.FillBytes(ret[1+byteLen:])
	return ret
}

func SerializeCompressed(pubKey *ecdsa.PublicKey) []byte {
	byteLen := (pubKey.Curve.Params().BitSize + 7) >> 3
	compressed := make([]byte, 1+byteLen)
	if pubKey.Y.Bit(0) == 1 {
		compressed[0] = 0x03
	} else {
		compressed[0] = 0x02
	}
	pubKey.X.FillBytes(compressed[1:])
	return compressed
}

func ParsePublicKey(data []byte) (*ecdsa.PublicKey, error) {
	// Use ecdh to safely validate the uncompressed SEC 1 point without deprecated APIs
	if _, err := ecdh.P256().NewPublicKey(data); err != nil {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}

	byteLen := (secp256k1.Params().BitSize + 7) >> 3
	x := new(big.Int).SetBytes(data[1 : 1+byteLen])
	y := new(big.Int).SetBytes(data[1+byteLen:])
	return &ecdsa.PublicKey{Curve: secp256k1, X: x, Y: y}, nil
}

func HashPubKey(pubKey []byte) []byte {
	publicSHA256 := sha256.Sum256(pubKey)
	// ripemd160 is deprecated; replacing with double-SHA256 for a modern baseline
	hash2 := sha256.Sum256(publicSHA256[:])
	return hash2[:]
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
		panic("invalid address: decoded length is too short")
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
