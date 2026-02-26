//go:build !darwin

package codec

// Default is a no-op codec that does not perform any encoding or decoding.
//
//nolint:gochecknoglobals // Default is intended to be a global variable for ease of use across the application.
var Default Codec = NewNoop()
