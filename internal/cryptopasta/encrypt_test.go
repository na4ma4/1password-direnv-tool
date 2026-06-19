// cryptopasta - basic cryptography examples
//
// Written in 2015 by George Tankersley <george.tankersley@gmail.com>
// Modified in 2026 by na4ma4 <https://github.com/na4ma4>
//
// To the extent possible under law, the author(s) have dedicated all copyright
// and related and neighboring rights to this software to the public domain
// worldwide. This software is distributed without any warranty.
//
// You should have received a copy of the CC0 Public Domain Dedication along
// with this software. If not, see // <http://creativecommons.org/publicdomain/zero/1.0/>.

package cryptopasta_test

import (
	"bytes"
	"crypto/rand"
	"io"
	"os"
	"testing"

	"golang.org/x/crypto/nacl/secretbox"

	"github.com/na4ma4/1password-direnv-tool/internal/cryptopasta"
)

func TestEncryptDecryptGCM(t *testing.T) {
	t.Parallel()

	randomKey := cryptopasta.NewEncryptionKey()

	gcmTests := []struct {
		plaintext []byte
		key       *cryptopasta.Key
	}{
		{
			plaintext: []byte("Hello, world!"),
			key:       randomKey,
		},
	}

	for _, tt := range gcmTests {
		ciphertext, err := cryptopasta.Encrypt(tt.plaintext, tt.key)
		if err != nil {
			t.Fatal(err)
		}

		plaintext, err := cryptopasta.Decrypt(ciphertext, tt.key)
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(plaintext, tt.plaintext) {
			t.Errorf("plaintexts don't match")
		}

		ciphertext[0] ^= 0xff
		plaintext, err = cryptopasta.Decrypt(ciphertext, tt.key)
		if err == nil {
			t.Errorf("gcmOpen should not have worked, but did")
		}

		if plaintext != nil {
			t.Errorf("gcmOpen should not have returned plaintext, but did")
		}
	}
}

func BenchmarkAESGCM(b *testing.B) {
	randomKey := cryptopasta.NewEncryptionKey()

	data, err := os.ReadFile("testdata/big")
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(data)))

	for range b.N {
		cryptopasta.Encrypt(data, randomKey)
	}
}

func BenchmarkSecretbox(b *testing.B) {
	randomKey := cryptopasta.NewEncryptionKey()

	nonce := &[24]byte{}
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		b.Fatal(err)
	}

	var data []byte
	{
		var err error
		data, err = os.ReadFile("testdata/big")
		if err != nil {
			b.Fatal(err)
		}
		b.SetBytes(int64(len(data)))
	}

	for range b.N {
		secretbox.Seal(nil, data, nonce, (*[32]byte)(randomKey))
	}
}
