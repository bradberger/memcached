package memcached

import (
	"strconv"
	"strings"
)

// todo look into using fmt.Sscanf for parsing
type command string

func (c command) String() string {
	return strings.TrimSpace(string(c))
}

func (c command) Error() error {
	switch c.Name() {
	default:
		return ErrUnknownCommand
	case cmdGet:
		fallthrough
	case cmdGets:
		if c.Len() < 2 {
			return ErrInvalidCommand
		}
		return nil
	case cmdDel:
		if c.Len() != 2 {
			return ErrInvalidCommand
		}
		return nil
	case cmdFlushAll:
		return nil
	case cmdIncr:
		return nil
	case cmdDecr:
		return nil
	case cmdCas:
		return ErrNotImplemented
	case cmdAppend:
		fallthrough
	case cmdPrepend:
		fallthrough
	case cmdReplace:
		fallthrough
	case cmdAdd:
		fallthrough
	case cmdSet:
		if c.Len() < 5 || c.Len() > 6 {
			return ErrInvalidCommand
		}
		return nil
	}
}

func (c command) Parse() []string {
	return strings.Split(c.String(), " ")
}

func (c command) Len() int {
	return len(c.Parse())
}

func (c command) Name() string {
	return c.Parse()[0]
}

func (c command) Key() []string {
	if c.Len() < 2 {
		return nil
	}
	switch c.Name() {
	default:
		return []string{c.Parse()[1]}
	case cmdGets:
		fallthrough
	case cmdGet:
		return c.Parse()[1:]
	}
}

func (c command) Flags() int {
	if c.Len() < 3 {
		return 0
	}
	f, _ := strconv.ParseInt(c.Parse()[2], 10, 64)
	return int(f)
}

func (c command) Exp() int {
	if c.Len() < 4 {
		return 0
	}
	b, _ := strconv.ParseInt(c.Parse()[3], 10, 64)
	return int(b)
}

func (c command) Bytes() int {
	if c.Len() < 5 {
		return 0
	}
	b, _ := strconv.ParseInt(c.Parse()[4], 10, 64)
	return int(b)
}

func (c command) CasUnique() int {
	return 0
}

func (c command) NoReply() bool {
	switch c.Name() {
	default:
		return c.Len() == 6
	case cmdIncr:
		return c.Len() == 4
	case cmdDecr:
		return c.Len() == 4
	case cmdCas:
		return c.Len() == 7
	}
}

func (c command) GetDelta() (uint64, error) {
	delta, err := strconv.ParseInt(c.Parse()[2], 10, 64)
	return uint64(delta), err
}
