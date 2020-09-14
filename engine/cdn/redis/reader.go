package redis

import (
	"io"
	"strings"
)

type Reader struct {
	ReadWrite
	LastIndex     uint
	From          uint
	Size          int
	CurrentBuffer string
}

func (r *Reader) Read(p []byte) (n int, err error) {
	lenght := len(p)
	var buffer string
	if len(r.CurrentBuffer) > 0 {
		if len(r.CurrentBuffer) <= lenght {
			buffer = r.CurrentBuffer
		}
	}

	var newFromIndex uint
	var newToIndex uint

	// First read
	if r.From > 0 && r.LastIndex == 0 {
		r.LastIndex = r.From
	}
	if r.Size >= 0 && newFromIndex+100 > (r.From+uint(r.Size-1)) {
		newToIndex = r.From + uint(r.Size-1)
	} else {
		newToIndex = newFromIndex + 100
	}

	lines, err := r.get(r.LastIndex, newToIndex)
	if err != nil {
		return 0, err
	}

	if len(lines) > 0 {
		r.CurrentBuffer += strings.Join(lines, "")
	}

	if len(buffer) < lenght && len(r.CurrentBuffer) > 0 {
		x := lenght - len(buffer)
		if x < len(r.CurrentBuffer) {
			buffer += r.CurrentBuffer[:x]
			r.CurrentBuffer = r.CurrentBuffer[x:]
		} else {
			buffer += r.CurrentBuffer
			r.CurrentBuffer = ""
		}
	}

	r.LastIndex = newToIndex
	err = nil
	if len(lines) == 0 || (r.Size-1 >= 0 && r.LastIndex == (r.From+uint(r.Size-1))) {
		err = io.EOF
	}

	return copy(p, buffer), err
}
