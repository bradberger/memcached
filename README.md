# Go Memcache Server

A Go implementation of the memcached protocol, designed to be easy to switch
storage mechanisms depending on your specific needs.

## Usage

```go
import (
  "log"

  "github.com/bradberger/memcached"
)

func main() {
  srv := memcached.New()
  log.Fatal(srv.ListenAndServe())
}
```

## Configuration

You can configure the storage engine, the listening address, and the logging interface.
The server currently implements a subset of the [memcached protocol](https://github.com/memcached/memcached/blob/master/doc/protocol.txt). The extent to which it implements all the storage methods
are dependent on the storage engine you choose.
*(I'll add details about what this means in the future)*

```go
import (
  "log"

  "github.com/bradberger/memcached"
  "github.com/bradberger/gocache/drivers/lru"
)

func main() {

  srv := &memcached.Server{
    // Set the cache engine.
    // Must implement "github.com/bradberger/gocache/cache.Cache" interface.
		Cache:  lru.NewBasic(lru.Gigabyte, 100000),
    // Set the logger.
    // Must implement "github.com/bradberger/memcached".Logger interface.
		Logger: logger.Init("Memcached", true, false, os.Stdout),
    // Set the listen address. Default is :11211
		Addr:   ":11211",
	}

  log.Fatal(srv.ListenAndServe())
}

```

## TODO

- [X] `increment` command support
- [X] `decrement` command support
- [X] `touch` command support
- [-] `append` command support
- [-] `prepend` command support
- [ ] `gat` command support
- [ ] `gats` command support
- [ ] `cas` command support
- [ ] UDP support
- [ ] Binary protocol support
- [X] handle `noreply` properly
