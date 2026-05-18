package state

// StateDB defines a minimal state persistence interface.
type StateDB interface {
	Get(key []byte) ([]byte, error)
	Put(key, value []byte) error
	Delete(key []byte) error
}
