package memcached

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/bradberger/gocache/cache"
	"golang.org/x/sync/errgroup"
)

type session struct {
	srv *Server
	rwc net.Conn
	br  *bufio.Reader
	bw  *bufio.Writer
}

func (s *session) serve() {

	defer s.rwc.Close()

	defer func() {
		if r := recover(); r != nil {
			s.srv.Logger.Errorf("Panic: %v", r)
			s.sendservererror(fmt.Errorf("%v", r))
		}
	}()

	for {

		sl, err := s.br.ReadSlice('\n')
		if err != nil {
			if err == io.EOF {
				s.rwc.Close()
				return
			}

			s.srv.Logger.Warningf("read error: %v", err)
			return
		}

		cmd := command(strings.TrimSpace(string(sl)))
		if err := cmd.Error(); err != nil {
			s.sendservererror(err)
			continue
		}

		s.handle(cmd)
	}
}

func (s *session) senditems(items []*item) {
	for i := range items {
		s.senditem(items[i])
	}
	s.sendlinef("END")
}

func (s *session) senditem(i *item) {
	s.sendlinef("VALUE %s %v %v %v", i.key, i.flags, len(string(i.value)), i.cas)
	s.sendlinef("%s", string(i.value))
}

func (s *session) sendf(format string, args ...interface{}) {
	fmt.Fprintf(s.bw, format, args...)
	if err := s.bw.Flush(); err != nil {
		s.srv.Logger.Errorf("Couldn't write message to network: %v", err)
	}
}

func (s *session) sendlinef(format string, args ...interface{}) {
	s.sendf(format+"\r\n", args...)
}

func (s *session) sendclienterror(e error) {
	s.sendlinef("CLIENT_ERROR %v", e)
}

func (s *session) sendservererror(e error) {
	switch e {
	case cache.ErrKeyExists:
		s.sendlinef("EXISTS")
	case cache.ErrNotStored:
		s.sendlinef("NOT_STORED")
	case cache.ErrNotFound:
		s.sendlinef("NOT_FOUND")
	case ErrUnknownCommand:
		s.sendlinef("ERROR")
	default:
		s.sendlinef("SERVER_ERROR %v", e)
	}
}

func (s *session) getItem(cmd command) (*item, error) {

	buf := make([]byte, cmd.Bytes())
	if _, err := io.ReadFull(s.br, buf); err != nil {
		return nil, err
	}

	if _, err := s.br.ReadSlice('\n'); err != nil {
		return nil, err
	}

	return &item{
		key:   cmd.Key()[0],
		value: buf,
		exp:   getExpiration(cmd.Exp()),
	}, nil
}

func (s *session) handleSet(cmd command) {

	// Read the value
	val, err := s.getItem(cmd)
	if err != nil {
		s.sendservererror(err)
		return
	}

	if err := s.srv.Cache.Set(val.key, val, val.expires()); err != nil {
		s.sendservererror(err)
		return
	}

	s.sendlinef("STORED")
}

func (s *session) handleAdd(cmd command) {
	a, ok := s.srv.Cache.(cache.Add)
	if !ok {
		s.sendservererror(ErrNotImplemented)
		return
	}

	val, err := s.getItem(cmd)
	if err != nil {
		s.sendservererror(err)
		return
	}

	if err := a.Add(val.key, val, val.expires()); err != nil {
		s.sendservererror(err)
		return
	}

	s.sendlinef("STORED")

}

func (s *session) handleReplace(cmd command) {
	rp, ok := s.srv.Cache.(cache.Replace)
	if !ok {
		s.sendservererror(ErrNotImplemented)
		return
	}

	val, err := s.getItem(cmd)
	if err != nil {
		s.sendservererror(err)
		return
	}

	if err := rp.Replace(val.key, val); err != nil {
		if err == cache.ErrNotStored {
			s.sendlinef("NOT_STORED")
			return
		}
		s.sendservererror(err)
		return
	}

	s.sendlinef("STORED")

}

func (s *session) handleAppend(cmd command) {

	key := cmd.Key()[0]

	if !s.srv.Cache.Exists(key) {
		s.sendservererror(cache.ErrNotStored)
		return
	}

	itm := item{}
	if err := s.srv.Cache.Get(key, &itm); err != nil {
		s.sendservererror(err)
		return
	}

	val, err := s.getItem(cmd)
	if err != nil {
		s.sendservererror(err)
		return
	}

	// TODO Revisit this... it's essentially doing a "touch" now.
	// Might have to do store time set and time expires explicitly.
	itm.value = append(itm.value, val.value...)
	if err := s.srv.Cache.Set(itm.key, itm, val.expires()); err != nil {
		s.sendservererror(err)
		return
	}

	s.sendlinef("STORED")
}

func (s *session) handlePrepend(cmd command) {
	key := cmd.Key()[0]

	if !s.srv.Cache.Exists(key) {
		s.sendservererror(cache.ErrKeyExists)
		return
	}

	itm := item{}
	if err := s.srv.Cache.Get(key, &itm); err != nil {
		s.sendservererror(err)
		return
	}

	val, err := s.getItem(cmd)
	if err != nil {
		s.sendservererror(err)
		return
	}

	// TODO Revisit this... it's essentially doing a "touch" now.
	// Might have to do store time set and time expires explicitly.
	itm.value = append(val.value, itm.value...)
	if err := s.srv.Cache.Set(itm.key, itm, itm.expires()); err != nil {
		s.sendservererror(err)
		return
	}

	s.sendlinef("STORED")

}

func (s *session) handleDelete(cmd command) {
	key := cmd.Key()[0]
	if err := s.srv.Cache.Del(key); err != nil {
		s.sendservererror(err)
		return
	}

	s.sendlinef("DELETED")
}

func (s *session) handleFlushAll(cmd command) {

	kl, ok := s.srv.Cache.(cache.KeyList)
	if !ok {
		s.sendservererror(ErrNotImplemented)
		return
	}

	var eg errgroup.Group
	keys := kl.Keys()
	for _, key := range keys {
		eg.Go(func() error {
			return s.srv.Cache.Del(key)
		})
	}

	if err := eg.Wait(); err != nil {
		s.sendservererror(err)
		return
	}

	s.sendlinef("OK")

}

func (s *session) handleGet(cmd command) {
	var mux sync.Mutex
	var wg sync.WaitGroup

	items := []*item{}
	keys := cmd.Key()
	wg.Add(len(keys))

	for i := range keys {

		go func(key string) {

			defer wg.Done()
			if !s.srv.Cache.Exists(key) {
				return
			}

			itm := item{}
			if err := s.srv.Cache.Get(key, &itm); err != nil {
				s.srv.Logger.Warningf("Cache get error: %v", err)
				return
			}

			mux.Lock()
			items = append(items, &itm)
			mux.Unlock()

		}(keys[i])
	}

	wg.Wait()
	s.senditems(items)

}

func (s *session) handle(cmd command) {

	// s.srv.Logger.Infof("Handling %s %+v", cmd.Name(), cmd.Key())

	switch cmd.Name() {

	default:
		s.sendservererror(ErrUnknownCommand)

	case cmdSet:
		s.handleSet(cmd)

	case cmdAdd:
		s.handleAdd(cmd)

	case cmdReplace:
		s.handleReplace(cmd)

	case cmdAppend:
		s.handleAppend(cmd)

	case cmdPrepend:
		s.handlePrepend(cmd)

	case cmdDel:
		s.handleDelete(cmd)

	case cmdFlushAll:
		s.handleFlushAll(cmd)

	case cmdQuit:
		s.handleQuit()

	case cmdCas:
		s.sendservererror(ErrNotImplemented)

	case cmdIncr:
		s.handleIncrement(cmd)

	case cmdDecr:
		s.handleDecrement(cmd)

	case cmdTouch:
		s.sendservererror(ErrNotImplemented)

	case cmdGat:
		s.sendservererror(ErrNotImplemented)

	case cmdGats:
		s.sendservererror(ErrNotImplemented)

	case cmdCacheMemlimit:
		s.sendservererror(ErrNotImplemented)

	case cmdVersion:
		s.sendservererror(ErrNotImplemented)

	case cmdStat:
		s.sendservererror(ErrNotImplemented)

	case cmdStats:
		s.sendservererror(ErrNotImplemented)

	case cmdGet:
		s.handleGet(cmd)

	case cmdGets:
		s.handleGet(cmd)
	}
}

func (s *session) handleQuit() {
	s.rwc.Close()
}

func (s *session) handleIncrement(cmd command) {

	delta, err := cmd.GetDelta()
	if err != nil {
		s.sendservererror(err)
		return
	}

	var itm item
	if err := s.srv.Cache.Get(cmd.Key()[0], &itm); err != nil {
		s.sendservererror(err)
		return
	}

	cur, err := strconv.ParseUint(string(itm.value), 10, 64)
	if err != nil {
		s.sendservererror(err)
		return
	}

	// TODO store this.
	s.sendlinef("%v", cur+delta)
}

func (s *session) handleDecrement(cmd command) {

	delta, err := cmd.GetDelta()
	if err != nil {
		s.sendservererror(err)
		return
	}

	var itm item
	if err := s.srv.Cache.Get(cmd.Key()[0], &itm); err != nil {
		s.sendservererror(err)
		return
	}

	cur, err := strconv.ParseUint(string(itm.value), 10, 64)
	if err != nil {
		s.sendservererror(err)
		return
	}

	// TODO store this.
	s.sendlinef("%v", cur-delta)
}
