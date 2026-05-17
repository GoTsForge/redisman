package utils

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gotsforge/redisman/cmd/constants"
)

type EntryId struct {
	Timestamp      int
	SequenceNumber int
}

// Format the entryId as: `ts-seq`
func (e EntryId) String() string {
	return fmt.Sprintf("%d-%d", e.Timestamp, e.SequenceNumber)
}

// -1 -> e < f, 0 -> e = f and 1 -> e > f
func (e EntryId) Compare(f EntryId) int {
	if e.Timestamp < f.Timestamp {
		return -1
	}

	if e.Timestamp > f.Timestamp {
		return 1
	}

	if e.SequenceNumber == f.SequenceNumber {
		return 0
	}

	if e.SequenceNumber < f.SequenceNumber {
		return -1
	}

	return 1
}

type ParsedEntryId struct {
	Timestamp               int
	SequenceNumber          int
	IsTimestampAutoGen      bool
	IsSequenceNumberAutoGen bool
	IsValid                 bool
}

// Returns timestamp, sequenceNumber, isTimeStampAutoGen, isSequenceNumberAutoGen, isValid
func ExtractDetailsFromEntryId(entryId string) ParsedEntryId {
	if entryId == "*" {
		return ParsedEntryId{
			Timestamp:               0,
			SequenceNumber:          0,
			IsTimestampAutoGen:      true,
			IsSequenceNumberAutoGen: true,
			IsValid:                 true,
		}
	}

	parts := strings.Split(entryId, "-")
	if len(parts) != 2 {
		return ParsedEntryId{
			Timestamp:               0,
			SequenceNumber:          0,
			IsTimestampAutoGen:      false,
			IsSequenceNumberAutoGen: false,
			IsValid:                 false,
		}
	}

	timestamp := parts[0]
	sequenceNumber := parts[1]

	var timestampInt int
	var timestampErr error
	var sequenceNumberInt int
	var sequenceNumberError error

	// timestamp should always be a number here:
	timestampInt, timestampErr = strconv.Atoi(timestamp)
	if timestampErr != nil {
		return ParsedEntryId{
			Timestamp:               0,
			SequenceNumber:          0,
			IsTimestampAutoGen:      false,
			IsSequenceNumberAutoGen: false,
			IsValid:                 false,
		}
	}

	if sequenceNumber == "*" {
		return ParsedEntryId{
			Timestamp:               timestampInt,
			SequenceNumber:          0,
			IsTimestampAutoGen:      false,
			IsSequenceNumberAutoGen: true,
			IsValid:                 true,
		}
	}

	sequenceNumberInt, sequenceNumberError = strconv.Atoi(sequenceNumber)
	if sequenceNumberError != nil {
		return ParsedEntryId{
			Timestamp:               0,
			SequenceNumber:          0,
			IsTimestampAutoGen:      false,
			IsSequenceNumberAutoGen: false,
			IsValid:                 false,
		}
	}

	if timestampInt < 0 {
		return ParsedEntryId{
			Timestamp:               0,
			SequenceNumber:          0,
			IsTimestampAutoGen:      false,
			IsSequenceNumberAutoGen: false,
			IsValid:                 false,
		}
	}

	if sequenceNumberInt < 0 {
		return ParsedEntryId{
			Timestamp:               0,
			SequenceNumber:          0,
			IsTimestampAutoGen:      false,
			IsSequenceNumberAutoGen: false,
			IsValid:                 false,
		}
	}

	return ParsedEntryId{
		Timestamp:               timestampInt,
		SequenceNumber:          sequenceNumberInt,
		IsTimestampAutoGen:      false,
		IsSequenceNumberAutoGen: false,
		IsValid:                 true,
	}
}

func ParseRangeStart(startId string) (EntryId, error) {
	if startId == "-" {
		return EntryId{
			Timestamp:      0,
			SequenceNumber: 0,
		}, nil
	}

	if startId == "+" {
		return EntryId{
			Timestamp:      constants.MaxInt,
			SequenceNumber: constants.MaxInt,
		}, nil
	}

	// split startId by "-"
	startIdParts := strings.Split(startId, "-")

	if len(startIdParts) > 2 {
		return EntryId{}, fmt.Errorf(constants.ERR_INVALID_STREAM_ID)
	}

	if len(startIdParts) == 1 {
		startIdTs, err := strconv.Atoi(startIdParts[0])
		if err != nil {
			return EntryId{}, fmt.Errorf(constants.ERR_INVALID_STREAM_ID)
		}

		return EntryId{
			Timestamp:      startIdTs,
			SequenceNumber: 0,
		}, nil
	}

	startIdTs, err := strconv.Atoi(startIdParts[0])
	if err != nil {
		return EntryId{}, fmt.Errorf(constants.ERR_INVALID_STREAM_ID)
	}

	startIdSeq, err := strconv.Atoi(startIdParts[1])
	if err != nil {
		return EntryId{}, fmt.Errorf(constants.ERR_INVALID_STREAM_ID)
	}

	return EntryId{
		Timestamp:      startIdTs,
		SequenceNumber: startIdSeq,
	}, nil
}

func ParseRangeEnd(endId string) (EntryId, error) {
	if endId == "-" {
		return EntryId{
			Timestamp:      0,
			SequenceNumber: 0,
		}, nil
	}

	if endId == "+" {
		return EntryId{
			Timestamp:      constants.MaxInt,
			SequenceNumber: constants.MaxInt,
		}, nil
	}

	// split endId by "-"
	endIdParts := strings.Split(endId, "-")

	if len(endIdParts) > 2 {
		return EntryId{}, fmt.Errorf(constants.ERR_INVALID_STREAM_ID)
	}

	if len(endIdParts) == 1 {
		endIdTs, err := strconv.Atoi(endIdParts[0])
		if err != nil {
			return EntryId{}, fmt.Errorf(constants.ERR_INVALID_STREAM_ID)
		}

		return EntryId{
			Timestamp:      endIdTs,
			SequenceNumber: constants.MaxInt,
		}, nil
	}

	endIdTs, err := strconv.Atoi(endIdParts[0])
	if err != nil {
		return EntryId{}, fmt.Errorf(constants.ERR_INVALID_STREAM_ID)
	}

	endIdSeq, err := strconv.Atoi(endIdParts[1])
	if err != nil {
		return EntryId{}, fmt.Errorf(constants.ERR_INVALID_STREAM_ID)
	}

	return EntryId{
		Timestamp:      endIdTs,
		SequenceNumber: endIdSeq,
	}, nil
}
