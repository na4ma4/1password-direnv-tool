package cache

import "time"

type DeleteError struct {
	Key string
}

func (e DeleteError) Error() string {
	return "delete: " + e.Key
}

func ExpireFunc(expiry time.Duration) IterateFunc {
	return func(key string, age time.Time, _ string) error {
		if time.Since(age) > expiry {
			return DeleteError{Key: key}
		}
		return nil
	}
}
