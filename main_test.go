package memcached

import (
	"os"
	"testing"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/stretchr/testify/assert"
)

var (
	c      *memcache.Client
	s      *Server
	listen = ":11211"
)

func TestMain(m *testing.M) {

	c = memcache.New(listen)
	go func(s *Server) {
		if err := ListenAndServe(listen); err != nil {
			panic(err)
		}
	}(s)

	time.Sleep(1 * time.Second)

	os.Exit(m.Run())
}

func TestSet(t *testing.T) {
	assert.NoError(t, c.Set(&memcache.Item{Key: "foo", Value: []byte("bar")}))
	assert.NoError(t, c.Delete("foo"))
}

func TestMultiSet(t *testing.T) {
	assert.NoError(t, c.Set(&memcache.Item{Key: "foo", Value: []byte("bar")}))
	assert.NoError(t, c.Set(&memcache.Item{Key: "foo", Value: []byte("bar")}))
	assert.NoError(t, c.Delete("foo"))
}

func TestGet(t *testing.T) {
	assert.NoError(t, c.Set(&memcache.Item{Key: "foo", Value: []byte("bar")}))

	itm, err := c.Get("foo")
	assert.NoError(t, err)
	assert.Equal(t, "bar", string(itm.Value))
	assert.NoError(t, c.Delete("foo"))
}

func TestMultiGet(t *testing.T) {
	assert.NoError(t, c.Set(&memcache.Item{Key: "foo", Value: []byte("foo")}))
	assert.NoError(t, c.Set(&memcache.Item{Key: "bar", Value: []byte("bar")}))
	items, err := c.GetMulti([]string{"foo", "bar"})

	assert.NoError(t, err)
	assert.Equal(t, 2, len(items))
	assert.Equal(t, []byte("foo"), items["foo"].Value)
	assert.Equal(t, []byte("bar"), items["bar"].Value)
	assert.NoError(t, c.Delete("foo"))
	assert.NoError(t, c.Delete("bar"))
}

func TestMiss(t *testing.T) {
	itm, err := c.Get("foo")
	assert.Nil(t, itm)
	assert.Error(t, err)
}

func TestAdd(t *testing.T) {
	assert.NoError(t, c.Add(&memcache.Item{Key: "foo", Value: []byte("bar")}))
}

func TestReplace(t *testing.T) {

	itm := memcache.Item{Key: "foo", Value: []byte("bar")}
	assert.NoError(t, c.Replace(&itm))
	assert.NoError(t, c.Delete("foo"))
	assert.Equal(t, memcache.ErrNotStored, c.Replace(&itm))
}

func TestAppend(t *testing.T) {
	t.Skipf("Client doesn't support apppend")
	// assert.NoError(t, c.Append(&memcache.Item{Key: "foo", Value: []byte("bar")}))
}

func TestPrepend(t *testing.T) {
	t.Skipf("Client doesn't support prepend")
	// assert.NoError(t, c.Prepend(&memcache.Item{Key: "foo", Value: []byte("bar")}))
}

func TestFlushAll(t *testing.T) {
	t.Skipf("flush_all doesn't work yet")
	// assert.NoError(t, c.FlushAll())
}

func TestDeleteAll(t *testing.T) {
	t.Skipf("flush_all doesn't work yet")
	// assert.NoError(t, c.DeleteAll())
}

func TestIncrement(t *testing.T) {

	// Set initial value.
	assert.NoError(t, c.Delete("foo"))
	assert.NoError(t, c.Set(&memcache.Item{Key: "foo", Value: []byte("1")}))

	// Check increment response.
	delta, err := c.Increment("foo", 1)
	assert.NoError(t, err)
	assert.Equal(t, uint64(2), delta)

	// Check get after increment
	itm, err := c.Get("foo")
	if assert.NoError(t, err) {
		assert.NotNil(t, itm)
		assert.Equal(t, []byte("2"), itm.Value)
	}
}

func TestDecrement(t *testing.T) {

	// Set up initial value.
	assert.NoError(t, c.Delete("foo"))
	assert.NoError(t, c.Set(&memcache.Item{Key: "foo", Value: []byte("100")}))

	// Check the decrement response
	delta, err := c.Decrement("foo", 1)
	assert.NoError(t, err)
	assert.Equal(t, uint64(99), delta)

	// Check get matches the value just returned
	itm, err := c.Get("foo")
	assert.NoError(t, err)
	assert.Equal(t, []byte("99"), itm.Value)
}

func TestCompareAndSwap(t *testing.T) {
	t.Skipf("TODO")
}
