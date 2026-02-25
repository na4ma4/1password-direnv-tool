package codec

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"lds.li/keychain"

	"github.com/na4ma4/1password-direnv-tool/internal/cryptopasta"
)

// passphraseQuery defines the query for the keychain item that stores the encryption key.
//
//nolint:gochecknoglobals // global variable is acceptable here since it's just a constant query definition
var passphraseQuery = keychain.GenericPasswordQuery{
	Account: "1password-direnv-tool",
	Service: "encryption",
}

type Keychain struct{}

func NewKeychain() *Keychain {
	return &Keychain{}
}

func (k *Keychain) generateNewKey() (*cryptopasta.Key, error) {
	passphrase := []byte(cryptopasta.NewEncryptionKey().String())
	err := keychain.CreateGenericPassword(keychain.GenericPassword{
		Account: passphraseQuery.Account,
		Service: passphraseQuery.Service,
		Label:   "1password-direnv-tool encryption key",
		Value:   passphrase,
	})
	if err != nil {
		return nil, err
	}

	key := &cryptopasta.Key{}
	copy(key[:], passphrase)

	return key, nil
}

func (k *Keychain) GetKey() (*cryptopasta.Key, error) {
	var passphrase []byte
	{
		var err error
		passphrase, err = keychain.GetGenericPassword(passphraseQuery)
		if err != nil {
			if keychainError, ok := errors.AsType[*keychain.Error](err); ok &&
				keychainError.Code() == keychain.ErrorCodeItemNotFound {
				return k.generateNewKey()
			} else if ok {
				return nil, keychainError
			}

			return nil, err
		}
	}

	key := &cryptopasta.Key{}
	copy(key[:], passphrase)

	return key, nil
}

func (k *Keychain) ExportKey() (string, error) {
	key, err := k.GetKey()
	if err != nil {
		return "", err
	}

	return Base32SNoPadding.EncodeToString(key.Bytes()), nil
}

func (k *Keychain) ImportKey(key any) error {
	switch v := key.(type) {
	case string:
		var code string
		{
			var ok bool
			code, ok = strings.CutPrefix(v, "encv1-key://")
			if !ok {
				return errors.New("invalid key format: missing 'encv1://' prefix")
			}
		}

		var decoded []byte
		{
			var err error
			decoded, err = Base32SNoPadding.DecodeString(code)
			if err != nil {
				return fmt.Errorf("failed to decode key: %w", err)
			}
		}

		if len(decoded) != cryptopasta.KeySize {
			return fmt.Errorf("decoded key has invalid length: expected 32 bytes, got %d bytes", len(decoded))
		}

		cpKey := &cryptopasta.Key{}
		copy(cpKey[:], decoded)

		if err := keychain.DeleteGenericPassword(passphraseQuery); err != nil {
			// If the error is "item not found", we can ignore it since we're going to create a new item anyway.
			if keychainError, ok := errors.AsType[*keychain.Error](err); !ok ||
				keychainError.Code() != keychain.ErrorCodeItemNotFound {
				return fmt.Errorf("failed to delete existing keychain item: %w", err)
			}
		}

		return keychain.CreateGenericPassword(keychain.GenericPassword{
			Account: passphraseQuery.Account,
			Service: passphraseQuery.Service,
			Label:   "1password-direnv-tool encryption key",
			Value:   cpKey.Bytes(),
		})
	default:
		return fmt.Errorf("unsupported key type: %s", reflect.TypeOf(key).String())
	}
}

func (k *Keychain) Encode(value string) (string, error) {
	var key *cryptopasta.Key
	{
		var err error
		key, err = k.GetKey()
		if err != nil {
			return "", err
		}
	}

	var data []byte
	{
		var err error
		data, err = cryptopasta.Encrypt([]byte(value), key)
		if err != nil {
			return "", err
		}
	}

	var dst []byte
	{
		dst = make([]byte, Base32SNoPadding.EncodedLen(len(data)))
		Base32SNoPadding.Encode(dst, data)
	}

	return string(dst), nil
}

func (k *Keychain) Decode(value string) (string, error) {
	var key *cryptopasta.Key
	{
		var err error
		key, err = k.GetKey()
		if err != nil {
			return "", err
		}
	}

	var data []byte
	{
		var err error
		data, err = Base32SNoPadding.DecodeString(value)
		if err != nil {
			return "", err
		}
	}

	var dst []byte
	{
		var err error
		dst, err = cryptopasta.Decrypt(data, key)
		if err != nil {
			return "", err
		}
	}

	return string(dst), nil
}
