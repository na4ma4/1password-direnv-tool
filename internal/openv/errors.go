package openv

import "errors"

// ErrInvalidItemRef is returned when the item reference is invalid.
var ErrInvalidItemRef = errors.New("invalid item reference")

// ErrVaultNotFound is returned when the vault cannot be found.
var ErrVaultNotFound = errors.New("vault not found")

// ErrItemNotFound is returned when the item cannot be found.
var ErrItemNotFound = errors.New("item not found")
