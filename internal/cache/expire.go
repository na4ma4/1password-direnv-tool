package cache

import (
	"time"

	"github.com/na4ma4/1password-direnv-tool/model"
)

type DeleteError struct {
	Key string
}

func (e DeleteError) Error() string {
	return "delete: " + e.Key
}

func ExpireFunc(expiry time.Duration) IterateFunc {
	return func(key string, _ *model.FileList, age time.Time, _ string) error {
		if time.Since(age) > expiry {
			return DeleteError{Key: key}
		}
		return nil
	}
}
