package protocol

import (
	"fmt"
	"strings"
)

type Response interface {
	Encode() []byte
}

type SimpleString struct {
	value string
}

func (s SimpleString) Encode() []byte {
	return []byte("+" + s.value + "\r\n")
}

func NewSimpleString(s string) (Response, error) {
	// validate first
	if strings.Contains(s, "\r") || strings.Contains(s, "\n") {
		return nil, fmt.Errorf("simple string cannot contain CR or LF")
	}

	return SimpleString{value: s}, nil
}

type BulkString struct {
	value string
}

func (b BulkString) Encode() []byte {
	formattedBulkString := fmt.Sprintf("$%d\r\n%s\r\n", len(b.value), b.value)
	return []byte(formattedBulkString)
}

func NewBulkString(s string) Response {
	return BulkString{value: s}
}

type ErrorResponse struct {
	message string
}

func (e ErrorResponse) Encode() []byte {
	return []byte("-" + e.message + "\r\n")
}

func NewErrorResponse(m string) Response {
	return ErrorResponse{message: m}
}

type NullBulkString struct{}

func (n NullBulkString) Encode() []byte {
	return []byte("$-1\r\n")
}

func NewNullBulkString() Response {
	return NullBulkString{}
}

type Integer struct {
	value int
}

func (i Integer) Encode() []byte {
	formattedInt := fmt.Sprintf(":%d\r\n", i.value)
	return []byte(formattedInt)
}

func NewInteger(i int) Response {
	return Integer{
		value: i,
	}
}

type BulkArray struct {
	values []string
}

func (a BulkArray) Encode() []byte {
	arrLen := len(a.values)

	bulkString := fmt.Sprintf("*%d\r\n", arrLen)
	for _, val := range a.values {
		formattedVal := fmt.Sprintf("$%d\r\n%s\r\n", len(val), val)
		bulkString += formattedVal
	}

	return []byte(bulkString)
}

func NewBulkArray(values []string) Response {
	return BulkArray{
		values: values,
	}
}
