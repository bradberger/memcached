package memcached

import (
	"bufio"
	"fmt"
	"strings"
)

type command struct {
	keys                               []string
	delta                              uint64
	name, key                          string
	flags, exptime, byteLen, casUnique int
	noReply                            bool
}

func parseCmd(cmdName string, r *bufio.Reader) (cmd *command, err error) {

	cmd = &command{name: cmdName}
	ln, err := r.ReadString('\n')
	if err != nil {
		return
	}

	ln = strings.TrimSpace(ln)

	switch cmdName {
	case cmdGet:
		fallthrough
	case cmdGets:
		cmd.keys = strings.Split(ln, " ")
	case cmdDel:
		cmd.noReply, err = parseLine(ln, "%s", &cmd.key)
	case cmdFlushAll:
	case cmdIncr:
		fallthrough
	case cmdDecr:
		cmd.noReply, err = parseLine(ln, "%s %d", &cmd.key, &cmd.delta)
	case cmdCas:
		cmd.noReply, err = parseLine(ln, "%s %d %d %d", &cmd.key, &cmd.exptime, &cmd.byteLen, &cmd.casUnique)
	case cmdTouch:
		cmd.noReply, err = parseLine(ln, "%s %d", &cmd.key, &cmd.exptime)
	case cmdAppend:
		fallthrough
	case cmdPrepend:
		fallthrough
	case cmdReplace:
		fallthrough
	case cmdAdd:
		fallthrough
	case cmdSet:
		cmd.noReply, err = parseLine(ln, "%s %d %d %d", &cmd.key, &cmd.flags, &cmd.exptime, &cmd.byteLen)
	default:
		err = ErrUnknownCommand
	}

	return
}

func isNoReply(lineFmt string, actual string) bool {
	return strings.Count(lineFmt, " ") > strings.Count(actual, " ")
}

func parseLine(ln string, lineFmt string, vars ...interface{}) (noReply bool, err error) {

	defer func() {
		if r := recover(); r != nil {
			err = ErrInvalidCommand
		}
	}()

	if _, err = fmt.Sscanf(ln, lineFmt, vars...); err != nil {
		return
	}

	noReply = strings.Count(ln, " ") > 3
	return
}
