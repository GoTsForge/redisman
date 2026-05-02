package utils

import (
	"strconv"
	"strings"
)

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
