package parser

import (
	"bytes"
	"strconv"
)

type Command struct {
	Name string
	Args []string
}

type Value struct {
	value string
}

// Returns command, bytesConsumed, status
func TryParse(buffer []byte) (Command, int, string) {
	idx := 0

	var respArr []string
	numCommands, bytesConsumedHeader, status := parseArrayHeader(buffer)

	if status != "success" {
		return Command{}, 0, status
	}

	idx += bytesConsumedHeader

	if status == "success" {
		for range numCommands {
			value, bytesConsumedBulk, status := parseBulk(buffer[idx:])

			if status == "error" || status == "incomplete" {
				return Command{}, 0, status
			}

			idx += bytesConsumedBulk

			if status == "success" && value != nil {
				respArr = append(respArr, value.value)
			}
		}
	}

	if len(respArr) == 0 {
		return Command{}, idx, "success"
	}

	return Command{
		Name: respArr[0],
		Args: respArr[1:],
	}, idx, "success"
}

func parseBulk(buffer []byte) (*Value, int, string) {
	if len(buffer) == 0 {
		return nil, 0, "incomplete"
	}

	if buffer[0] != byte('$') {
		return nil, 0, "error"
	}

	clrfIndex := bytes.Index(buffer, []byte("\r\n"))
	if clrfIndex == -1 {
		// no clrf present, the input is still incomplete
		return nil, 0, "incomplete"
	}

	slicedBuffer := buffer[1:clrfIndex]
	length, err := strconv.Atoi(string(slicedBuffer))
	if err != nil {
		return nil, 0, "error"
	}

	payloadStartIndex := clrfIndex + 2
	payloadEndIndex := payloadStartIndex + length

	payloadBuffer := buffer[payloadStartIndex:]
	if len(payloadBuffer) < length+2 {
		return nil, 0, "incomplete"
	}

	// last two elements aren't \r\n:
	if buffer[payloadEndIndex] != '\r' || buffer[payloadEndIndex+1] != '\n' {
		return nil, 0, "error"
	}

	value := string(buffer[payloadStartIndex:payloadEndIndex])
	bytesConsumed := clrfIndex + 2 + length + 2
	return &Value{value}, bytesConsumed, "success"
}

func parseArrayHeader(buffer []byte) (int, int, string) {
	if len(buffer) == 0 {
		return 0, 0, "incomplete"
	}

	if buffer[0] != byte('*') {
		return 0, 0, "error"
	}

	clrfIndex := bytes.Index(buffer, []byte("\r\n"))
	if clrfIndex == -1 {
		// no clrf present, the input is still incomplete
		return 0, 0, "incomplete"
	}

	slicedBuffer := buffer[1:clrfIndex]

	length, err := strconv.Atoi(string(slicedBuffer))
	if err != nil {
		return 0, 0, "error"
	}

	return length, clrfIndex + 2, "success"
}
