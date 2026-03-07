package handler

import (
	"github.com/gotsforge/redisman/cmd/parser"
	"github.com/gotsforge/redisman/cmd/protocol"
	"github.com/gotsforge/redisman/cmd/server"
)

type Handler func(*server.Server, parser.Command) protocol.Response

func HandlePing(s *server.Server, cmd parser.Command) protocol.Response {
	if len(cmd.Args) > 1 {
		return protocol.NewErrorResponse("too many arguments for PING")
	}

	if len(cmd.Args) == 0 {
		// new simple string with message PONG
		r, err := protocol.NewSimpleString("PONG")
		if err != nil {
			// should never happen!
			return nil
		}

		return r
	}

	arg := cmd.Args[0]
	return protocol.NewBulkString(arg)
}

func HandleSet(s *server.Server, cmd parser.Command) protocol.Response {
	if len(cmd.Args) != 2 {
		return protocol.NewErrorResponse("invalid number of arguments for SET")
	}

	key := cmd.Args[0]
	val := cmd.Args[1]

	s.Set(key, val)

	resp, err := protocol.NewSimpleString("OK")
	if err != nil {
		return protocol.NewErrorResponse(err.Error())
	}

	return resp
}

func HandleGet(s *server.Server, cmd parser.Command) protocol.Response {
	key := cmd.Args[0]

	value, exists := s.Get(key)
	if !exists {
		return protocol.NewNullBulkString()
	}

	return protocol.NewBulkString(value)
}
