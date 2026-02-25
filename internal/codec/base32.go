package codec

import "encoding/base32"

// Base32SNoPadding is a base32 encoding with no padding, used for encoding
// cache keys in the Disk cache implementation.
//
//nolint:gochecknoglobals // global variable is acceptable here since it's just a constant encoding definition
var Base32SNoPadding = base32.StdEncoding.WithPadding(base32.NoPadding)
