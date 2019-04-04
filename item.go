package memcached

import "time"

type item struct {
	key   string
	value []byte
	flags int64
	exp   time.Time
	cas   int64
}

func (i item) expired() bool {
	if i.exp.IsZero() {
		return false
	}
	return i.exp.Before(time.Now())
}

func (i item) expires() time.Duration {
	if i.exp.IsZero() {
		return 0
	}
	return i.exp.Sub(time.Now())
}
