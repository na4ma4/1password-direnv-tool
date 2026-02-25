//go:build !darwin

package codec

var Default Codec = NewNoop()
