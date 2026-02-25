package openv

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/na4ma4/1password-direnv-tool/internal/itemref"
)

// ParseItemRef parses an item reference into vault and item components.
// Supported formats:
//   - op://vault-name-or-id/item-name-or-id
func ParseItemRef(itemRef itemref.Ref) (string, string, error) {
	u, err := url.Parse(itemRef.String())
	if err != nil {
		return "", "", fmt.Errorf("%w: %w", ErrInvalidItemRef, err)
	}

	if u.Scheme != "op" {
		return "", "", fmt.Errorf("%w: expected op:// scheme, got %q", ErrInvalidItemRef, u.Scheme)
	}

	vaultRef := u.Host
	itemRef2 := strings.TrimPrefix(u.Path, "/")

	if vaultRef == "" || itemRef2 == "" {
		return "", "", fmt.Errorf("%w: expected op://vault/item format", ErrInvalidItemRef)
	}

	// Remove any trailing path components (section/field) if present.
	parts := strings.SplitN(itemRef2, "/", 2) //nolint:mnd // 2 is the max number of parts
	itemRef2 = parts[0]

	return vaultRef, itemRef2, nil
}
