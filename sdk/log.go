package sdk

import (
	"time"

	"github.com/golang/protobuf/ptypes"
)

// NewLog returns a log struct
func NewLog(pbJobID int64, value string, pipelineBuildID int64, stepOrder int) *Log {
	//There cant be any error since we are using time.Now which is obviously a real and valid timestamp
	now, _ := ptypes.TimestampProto(time.Now())
	l := &Log{
		PipelineBuildJobID: pbJobID,
		PipelineBuildID:    pipelineBuildID,
		Start:              now,
		StepOrder:          int64(stepOrder),
		Val:                value,
		LastModified:       now,
	}

	return l
}
