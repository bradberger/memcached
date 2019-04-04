package memcached

import (
	"errors"
	"os"
	"time"

	"github.com/bradberger/gocache/drivers/lru"
	"github.com/google/logger"
)

var (
	ErrNotImplemented = errors.New("not implemented")
	ErrInvalidCommand = errors.New("invalid command")
	ErrUnknownCommand = errors.New("unknown command")
)

const (
	cmdCas           = "cas"
	cmdSet           = "set"
	cmdAdd           = "add"
	cmdReplace       = "replace"
	cmdAppend        = "append"
	cmdPrepend       = "prepend"
	cmdGet           = "get"
	cmdGets          = "gets"
	cmdIncr          = "incr"
	cmdDecr          = "decr"
	cmdDel           = "delete"
	cmdTouch         = "touch"
	cmdGat           = "gat"
	cmdGats          = "gats"
	cmdFlushAll      = "flush_all"
	cmdCacheMemlimit = "cache_memlimit"
	cmdVersion       = "version"
	cmdQuit          = "quit"
	cmdStat          = "stat"
	cmdStats         = "stats"
)

func New() *Server {
	return &Server{
		Cache:  lru.NewBasic(lru.Gigabyte, 100000),
		Logger: logger.Init("Memcached", true, false, os.Stdout),
		Addr:   ":11211",
	}
}

// getExpiration caculates the actual time of the expiration based on
// the given int. Per the memcached spec the actual value sent may either be
// Unix time (number of seconds since January 1, 1970, as a 32-bit
// value), or a number of seconds starting from current time. In the
// latter case, this number of seconds may not exceed 60*60*24*30 (number
// of seconds in 30 days); if the number sent by a client is larger than
// that, the server will consider it to be real Unix time value rather
// than an offset from current time.
func getExpiration(dur int) time.Time {
	switch {
	case dur > 2592000:
		return time.Unix(int64(dur), 0)
	case dur > 0:
		return time.Now().Add(time.Duration(dur) * time.Second)
	default:
		return time.Time{}
	}
}
