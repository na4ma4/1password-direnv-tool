package itemref

import (
	"errors"
	"os"
)

type env struct {
	val string
}

func (e *env) Version() RefVersion {
	return coreRefType(e)
}

func (e *env) IsEmpty() bool {
	return coreIsEmpty(e)
}

func (e *env) String() string {
	return e.val
}

func getEnv() (*env, error) {
	v := os.Getenv("OP_ITEM_UUID")
	if v == "" {
		return nil, errors.New("env: missing OP_ITEM_UUID")
	}

	return &env{val: v}, nil
}
