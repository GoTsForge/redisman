package handler

import (
	"slices"
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

func HandleLPush(s *server.Server, cmd parser.Command) protocol.Response {
	if len(cmd.Args) < 2 {
		return protocol.NewErrorResponse(constants.ERR_WRONG_NUMBER_OF_ARGS_LPUSH)
	}

	key := cmd.Args[0]
	vals := cmd.Args[1:]

	// reverse the slice first since this is LPUSH
	slices.Reverse(vals)

	listLength, err := s.LPush(key, vals)

	if err != nil {
		return protocol.NewErrorResponse(err.Error())
	}

	return protocol.NewInteger(listLength)
}

func HandleLRange(s *server.Server, cmd parser.Command) protocol.Response {
	if len(cmd.Args) < 3 {
		return protocol.NewErrorResponse(constants.ERR_WRONG_NUMBER_OF_ARGS_RPUSH)
	}

	key := cmd.Args[0]
	startIndex := cmd.Args[1]
	stopIndex := cmd.Args[2]

	startIndexInt, err := strconv.Atoi(startIndex)
	if err != nil {
		return protocol.NewErrorResponse(constants.ERR_SYNTAX_ERROR)
	}

	stopIndexInt, err := strconv.Atoi(stopIndex)
	if err != nil {
		return protocol.NewErrorResponse(constants.ERR_SYNTAX_ERROR)
	}

	// call the LRange method on the store
	arr, err := s.LRange(key, startIndexInt, stopIndexInt)
	if err != nil {
		return protocol.NewErrorResponse(err.Error())
	}

	return protocol.NewBulkArray(arr)
}

func HandleLLen(s *server.Server, cmd parser.Command) protocol.Response {
	if len(cmd.Args) != 1 {
		return protocol.NewErrorResponse(constants.ERR_WRONG_NUMBER_OF_ARGS_LLEN)
	}

	key := cmd.Args[0]
	listLength, err := s.ListLen(key)
	if err != nil {
		return protocol.NewErrorResponse(err.Error())
	}

	return protocol.NewInteger(listLength)
}

func HandleLPop(s *server.Server, cmd parser.Command) protocol.Response {
	key := cmd.Args[0]

	poppedVal, found, err := s.ListPop(key)
	if err != nil {
		return protocol.NewErrorResponse(err.Error())
	}

	if !found {
		return protocol.NewNullBulkString()
	}

	return protocol.NewBulkString(poppedVal)
}

func HandleRPush(s *server.Server, cmd parser.Command) protocol.Response {
	if len(cmd.Args) < 2 {
		return protocol.NewErrorResponse(constants.ERR_WRONG_NUMBER_OF_ARGS_RPUSH)
	}

	key := cmd.Args[0]
	vals := cmd.Args[1:]

	listLength, err := s.RPush(key, vals)

	if err != nil {
		return protocol.NewErrorResponse(err.Error())
	}

	return protocol.NewInteger(listLength)
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

	if value.Type != server.TypeString {
		return protocol.NewErrorResponse("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	return protocol.NewBulkString(value.StringValue)
}
