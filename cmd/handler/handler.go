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
	"github.com/gotsforge/redisman/cmd/utils"
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

	var elements []protocol.Response
	for _, elem := range arr {
		elements = append(elements, protocol.NewBulkString(elem))
	}

	return protocol.NewArray(elements)
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
	if len(cmd.Args) < 1 || len(cmd.Args) > 2 {
		return protocol.NewErrorResponse(constants.ERR_WRONG_NUMBER_OF_ARGS_LPOP)
	}

	key := cmd.Args[0]
	numberOfArgsToRemove := cmd.Args[1]
	numberOfArgsToRemoveInt := 1

	if len(numberOfArgsToRemove) != 0 {
		var err error
		numberOfArgsToRemoveInt, err = strconv.Atoi(numberOfArgsToRemove)
		if err != nil {
			return protocol.NewErrorResponse(constants.ERR_SYNTAX_ERROR)
		}
	}

	poppedVals, found, err := s.ListPop(key, numberOfArgsToRemoveInt)
	if err != nil {
		return protocol.NewErrorResponse(err.Error())
	}

	if !found {
		return protocol.NewNullBulkString()
	}

	var elements []protocol.Response
	for _, poppedVal := range poppedVals {
		elements = append(elements, protocol.NewBulkString(poppedVal))
	}

	return protocol.NewArray(elements)
}

func HandleBLPop(s *server.Server, cmd parser.Command) protocol.Response {
	if len(cmd.Args) < 1 {
		return protocol.NewErrorResponse(constants.ERR_WRONG_NUMBER_OF_ARGS_BLPOP)
	}

	argsLength := len(cmd.Args)

	keys := cmd.Args[:argsLength-1]

	timeout := cmd.Args[argsLength-1]
	timeoutInt := 0

	if len(timeout) != 0 {
		var err error
		timeoutInt, err = strconv.Atoi(timeout)
		if err != nil {
			return protocol.NewErrorResponse(constants.ERR_SYNTAX_ERROR)
		}
	}

	poppedVals, found, err := s.BListPop(keys, time.Duration(timeoutInt)*time.Second)
	if err != nil {
		return protocol.NewErrorResponse(err.Error())
	}

	if !found {
		return protocol.NewNullBulkString()
	}

	var elements []protocol.Response
	for _, poppedVal := range poppedVals {
		elements = append(elements, protocol.NewBulkString(poppedVal))
	}

	return protocol.NewArray(elements)
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

func HandleType(s *server.Server, cmd parser.Command) protocol.Response {
	if len(cmd.Args) != 1 {
		return protocol.NewErrorResponse(constants.ERR_WRONG_NUMBER_OF_ARGS_TYPE)
	}

	key := cmd.Args[0]

	_, exists := s.Get(key)
	if !exists {
		res, err := protocol.NewSimpleString("none")
		if err != nil {
			return protocol.NewErrorResponse(err.Error())

		}

		return res
	}

	res, err := protocol.NewSimpleString("string")
	if err != nil {
		return protocol.NewErrorResponse(err.Error())

	}

	return res
}

func HandleXAdd(s *server.Server, cmd parser.Command) protocol.Response {
	// Stream key, entry id, atleast one K-V pair
	if len(cmd.Args) < 4 {
		return protocol.NewErrorResponse(constants.ERR_WRONG_NUMBER_OF_ARGS_XADD)
	}

	argsLen := len(cmd.Args)

	if argsLen%2 != 0 {
		return protocol.NewErrorResponse(constants.ERR_WRONG_NUMBER_OF_ARGS_XADD)
	}

	key := cmd.Args[0]
	entryId := cmd.Args[1]

	restArgs := cmd.Args[2:] // [k,v,k1,v1...] -> {k:v,k1:v1...}
	var kvPairs map[string]string = make(map[string]string)

	for idx := 0; idx < len(restArgs); idx += 2 {
		currVal := restArgs[idx]
		nextVal := restArgs[idx+1]
		kvPairs[currVal] = nextVal
	}

	// should parse entryId (string here) and return the timestamp and sequenceNumber
	parsedEntryId := utils.ExtractDetailsFromEntryId(entryId)

	if !parsedEntryId.IsValid {
		return protocol.NewErrorResponse(constants.ERR_INVALID_ID_XADD)
	}

	res, err := s.XAdd(key, parsedEntryId, kvPairs)
	if err != nil {
		return protocol.NewErrorResponse(err.Error())
	}

	return protocol.NewBulkString(res.String())
}
