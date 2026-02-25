//go:build darwin

package codec

// Default is the default Codec implementation used by the application. On Darwin, this is the Keychain codec.
//
//nolint:gochecknoglobals // Default is intended to be a global variable for ease of use across the application.
var Default Codec = NewKeychain()
