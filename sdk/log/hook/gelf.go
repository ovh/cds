// Copyright 2012 SocialCode. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package hook

import (
	"encoding/json"
	"io"
	"runtime"
	"strings"
)

// What compression type the writer should use when sending messages
// to the graylog2 server
type CompressType int

const (
	CompressGzip CompressType = iota
	CompressZlib
)

// Message represents the contents of the GELF message.  It is gzipped
// before sending.
type Message struct {
	Version  string                 `json:"version"`
	Host     string                 `json:"host"`
	Short    string                 `json:"short_message"`
	Full     string                 `json:"full_message,omitempty"`
	Time     float64                `json:"timestamp"`
	Level    int32                  `json:"level"`
	Pid      int                    `json:"_pid,omitempty"`
	Facility string                 `json:"_facility,omitempty"` // optional, deprecated, send as additional field
	File     string                 `json:"_file,omitempty"`     // optional, deprecated, send as additional field
	Line     int                    `json:"_line,omitempty"`     // optional, deprecated, send as additional field
	Prefix   string                 `json:"_prefix,omitempty"`
	Extra    map[string]interface{} `json:"-"`
}

type innerMessage Message //against circular (Un)MarshalJSON

type Writer interface {
	io.Writer
	WriteMessage(*Message) error
}

func (m *Message) MarshalJSON() ([]byte, error) {
	var err error
	var b, eb []byte

	extra := m.Extra
	b, err = json.Marshal((*innerMessage)(m))
	m.Extra = extra
	if err != nil {
		return nil, err
	}

	if len(extra) == 0 {
		return b, nil
	}

	if eb, err = json.Marshal(extra); err != nil {
		return nil, err
	}

	// merge serialized message + serialized extra map
	b[len(b)-1] = ','
	return append(b, eb[1:len(eb)]...), nil
}

func (m *Message) UnmarshalJSON(data []byte) error {
	i := make(map[string]interface{}, 16)
	if err := json.Unmarshal(data, &i); err != nil {
		return err
	}
	for k, v := range i {
		switch k {
		case "version":
			m.Version = v.(string)
		case "host":
			m.Host = v.(string)
		case "short_message":
			m.Short = v.(string)
		case "full_message":
			m.Full = v.(string)
		case "timestamp":
			m.Time = v.(float64)
		case "level":
			m.Level = int32(v.(float64))
		case "_pid":
			m.Pid = int(v.(float64))
		case "_facility":
			m.Facility = v.(string)
		case "_file":
			m.File = v.(string)
		case "_line":
			m.Line = int(v.(float64))
		case "_prefix":
			m.Prefix = v.(string)
		default:
			if k[0] == '_' {
				if m.Extra == nil {
					m.Extra = make(map[string]interface{}, 1)
				}
				m.Extra[k] = v
			}
		}
	}
	return nil
}

// getCaller returns the filename and the line info of a function
// further down in the call stack.  Passing 0 in as callDepth would
// return info on the function calling getCallerIgnoringLog, 1 the
// parent function, and so on.  Any suffixes passed to getCaller are
// path fragments like "/pkg/log/log.go", and functions in the call
// stack from that file are ignored.
func getCaller(callDepth int, suffixesToIgnore ...string) (file string, line int) {
	// bump by 1 to ignore the getCaller (this) stackframe
	callDepth++
outer:
	for {
		var ok bool
		_, file, line, ok = runtime.Caller(callDepth)
		if !ok {
			file = "???"
			line = 0
			break
		}

		for _, s := range suffixesToIgnore {
			if strings.HasSuffix(file, s) {
				callDepth++
				continue outer
			}
		}
		break
	}
	return
}

func getCallerIgnoringLogMulti(callDepth int) (string, int) {
	// the +1 is to ignore this (getCallerIgnoringLogMulti) frame
	return getCaller(callDepth+1, "logrus/hooks.go", "logrus/entry.go", "logrus/logger.go", "logrus/exported.go", "asm_amd64.s", "log/log.go")
}
