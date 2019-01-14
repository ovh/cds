package workflow

import (
	"github.com/ovh/cds/sdk"
)

// max size of a log in database in bytes
const (
	defaultMaxLogSize = 15*1024 ^ 2 // 15MB
	maxLogMarker      = "... truncated\n"
)

func truncateLogs(maxSize, existingSize int64, logs *sdk.Log) bool {
	if maxSize == 0 {
		maxSize = defaultMaxLogSize
	}
	if existingSize >= maxSize {
		return true
	}

	// calculate length to add
	sizeToAdd := int64(len(logs.Val))
	maxReached := existingSize+sizeToAdd > maxSize
	if maxReached {
		sizeToAdd = maxSize - existingSize
		logs.Val = logs.Val[0:sizeToAdd] + maxLogMarker
	}

	return false
}

func truncateServiceLogs(maxSize, existingSize int64, logs *sdk.ServiceLog) bool {
	if maxSize == 0 {
		maxSize = defaultMaxLogSize
	}
	if existingSize >= maxSize {
		return true
	}

	// calculate length to add
	sizeToAdd := int64(len(logs.Val))
	maxReached := existingSize+sizeToAdd > maxSize
	if maxReached {
		sizeToAdd = maxSize - existingSize
		logs.Val = logs.Val[0:sizeToAdd] + maxLogMarker
	}

	return false
}
