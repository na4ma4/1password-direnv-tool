package cryptopasta

import (
	"crypto/rand"
	"io"
)

const KeySize = 32

type Key [KeySize]byte

// NewEncryptionKey generates a random 256-bit key for Encrypt() and
// Decrypt(). It panics if the source of randomness fails.
func NewEncryptionKey() *Key {
	key := Key{}
	_, err := io.ReadFull(rand.Reader, key[:])
	if err != nil {
		panic(err)
	}
	return &key
}

func (k *Key) String() string {
	return string(k[:])
}

func (k *Key) Bytes() []byte {
	return k[:]
}
