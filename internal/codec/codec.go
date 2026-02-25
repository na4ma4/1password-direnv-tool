package codec

type Codec interface {
	// Encode takes a plaintext string and returns an encoded version of it.
	Encode(value string) (string, error)

	// Decode takes an encoded string and returns the original value.
	Decode(value string) (string, error)

	// ExportKey returns the key used for encoding and decoding, if applicable.
	ExportKey() (string, error)

	// ImportKey allows importing a key for encoding and decoding, if applicable.
	ImportKey(key any) error
}
