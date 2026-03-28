package utils

import (
	"strconv"
	"strings"
)

func ExtractDetailsFromEntryId(entryId string) (int, int, bool) {
	parts := strings.Split(entryId, "-")
	if len(parts) != 2 {
		return 0, 0, false
	}

	timestamp, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false
	}

	sequenceNumber, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, false
	}

	return timestamp, sequenceNumber, true
}
