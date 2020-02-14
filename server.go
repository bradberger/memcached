package memcached

import (
	"bufio"
	"log"
	"net"

	"github.com/bradberger/gocache/cache"
)

// Server defines parameters for running a memcache server.
type Server struct {
	Logger Logger
	Cache  cache.Cache
	Addr   string
}

// ListenAndServe listens on the configured network address
// and then calls Serve to handle memcache requests
func (srv *Server) ListenAndServe() error {

	ln, e := net.Listen("tcp", srv.Addr)
	if e != nil {
		return e
	}
	return srv.Serve(ln)
}

// Serve accepts incoming connections on the Listener l, creating a new service goroutine for each.
func (srv *Server) Serve(l net.Listener) error {
	defer l.Close()
	for {
		rw, e := l.Accept()
		if e != nil {
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				log.Printf("memcached: error: %v", e)
				continue
			}
			return e
		}
		sess, err := srv.newSession(rw)
		if err != nil {
			continue
		}
		go sess.serve()
	}
}

func (srv *Server) newSession(rwc net.Conn) (s *session, err error) {
	s = &session{
		srv: srv,
		rwc: rwc,
		br:  bufio.NewReader(rwc),
		bw:  bufio.NewWriter(rwc),
	}
	return
}
