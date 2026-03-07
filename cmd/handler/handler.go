package handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/gotsforge/redisman/cmd/constants"
	"github.com/gotsforge/redisman/cmd/parser"
	"github.com/gotsforge/redisman/cmd/protocol"
	"github.com/gotsforge/redisman/cmd/server"
)

type Handler func(*server.Server, parser.Command) protocol.Response

func HandlePing(s *server.Server, cmd parser.Command) protocol.Response {
	if len(cmd.Args) > 1 {
		return protocol.NewErrorResponse(constants.ERR_WRONG_NUMBER_OF_ARGS_PING)
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
	if len(cmd.Args) < 2 {
		return protocol.NewErrorResponse(constants.ERR_WRONG_NUMBER_OF_ARGS_SET)
	}

	key := cmd.Args[0]
	val := cmd.Args[1]

	// no expiry -> number of args is 2
	if len(cmd.Args) == 2 {
		s.Set(key, val, time.Time{})
	} else if len(cmd.Args) != 4 {
		return protocol.NewErrorResponse(constants.ERR_WRONG_NUMBER_OF_ARGS_SET)
	} else {
		// 4 arguments -> key, val, expiryType, expiryValue

		expiryType := cmd.Args[2]  // EX or PX
		expiryValue := cmd.Args[3] // seconds or milliseconds

		if strings.ToLower(expiryType) != "ex" && strings.ToLower(expiryType) != "px" {
			return protocol.NewErrorResponse(constants.ERR_SYNTAX_ERROR)
		}

		// parse expiryValue to be an int
		expiryValInt, err := strconv.Atoi(expiryValue)
		if err != nil {
			return protocol.NewErrorResponse("ERR value is not an integer or out of range")
		}

		switch strings.ToLower(expiryType) {
		case "px":
			expiry := time.Now().Add(time.Millisecond * time.Duration(expiryValInt))
			s.Set(key, val, expiry)
		case "ex":
			expiry := time.Now().Add(time.Second * time.Duration(expiryValInt))
			s.Set(key, val, expiry)
		}

	}

	resp, err := protocol.NewSimpleString("OK")
	if err != nil {
		return protocol.NewErrorResponse(err.Error())
	}

	return resp
}

func HandleGet(s *server.Server, cmd parser.Command) protocol.Response {
	if len(cmd.Args) != 1 {
		return protocol.NewErrorResponse(constants.ERR_WRONG_NUMBER_OF_ARGS_GET)
	}

	key := cmd.Args[0]

	value, exists := s.Get(key)
	if !exists {
		return protocol.NewNullBulkString()
	}

	return protocol.NewBulkString(value.Value)
}
