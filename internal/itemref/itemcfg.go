package itemref

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/na4ma4/1password-direnv-tool/internal/codec"
)

type RefVersion string

const (
	RefVersionPlain       RefVersion = "plain"
	RefVersionEncryptedV1 RefVersion = "encrypted-v1"
)

type Ref interface {
	Version() RefVersion
	IsEmpty() bool
	String() string
}

var ErrNoValidRef = errors.New("no valid reference found")

func GetRef(ctx context.Context, c codec.Codec) (Ref, error) {
	var r Ref
	{
		var err error
		r, err = getRawRef(ctx)
		if err != nil {
			return nil, err
		}
	}

	switch r.Version() {
	case RefVersionEncryptedV1:
		val, err := c.Decode(strings.TrimPrefix(r.String(), "encv1://"))
		if err != nil {
			return nil, fmt.Errorf("decrypting reference: %w", err)
		}

		return &encrypted{val: val}, nil
	case RefVersionPlain:
		return r, nil
	default:
		return r, nil
	}
}

func getRawRef(ctx context.Context) (Ref, error) {
	if v, err := getEnv(); err == nil {
		return v, nil
	}

	if v, err := getGitConfig(ctx, "1password.envrc-item"); err == nil {
		return v, nil
	}

	if v, err := getViper("item"); err == nil {
		return v, nil
	}

	return nil, ErrNoValidRef
}
