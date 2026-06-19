package codec_test

import (
	"testing"

	"github.com/na4ma4/1password-direnv-tool/internal/codec"
)

func TestKeychainCodec(t *testing.T) {
	t.Skip("This test creates and binds an option in the keychain to the test binary")

	t.Parallel()

	k := codec.NewKeychain()

	key, err := k.GetKey()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Key: %q", key)

	encoded, err := k.Encode("hello world")
	if err != nil {
		t.Fatal(err)
	}

	if encoded == "hello world" {
		t.Errorf("encoded value should not be the same as the original value")
	}

	decoded, err := k.Decode(encoded)
	if err != nil {
		t.Fatal(err)
	}

	if decoded != "hello world" {
		t.Errorf("decoded value should be the same as the original value")
	}
}
