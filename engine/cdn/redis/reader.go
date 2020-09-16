package redis

import (
	"io"
	"strings"
)

type Reader struct {
	ReadWrite
	nextIndex     uint
	From          uint // the offset that we want to use when reading lines from Redis
	Size          int  // the count of lines that we want to read (0 means to the end)
	currentBuffer []byte
}

func (r *Reader) loadMoreLines() error {
	// If its first read (next index == 0), init the next index with 'from' value
	if r.nextIndex == 0 {
		r.nextIndex = r.From // 'from' can be 0
	}

	// Read 100 lines if possible or only the missing lines if less than 100
	alreadyReadLinesLength := r.nextIndex - r.From
	var newNextIndex uint
	if r.Size > 0 {
		linesLeftToRead := uint(r.Size) - alreadyReadLinesLength
		if linesLeftToRead == 0 {
			return nil
		}
		if linesLeftToRead > 100 {
			newNextIndex = r.nextIndex + 100
		} else {
			newNextIndex = r.nextIndex + linesLeftToRead
		}
	} else {
		newNextIndex = r.nextIndex + 100
	}

	// Get new lines from Redis and append it to current buffer
	lines, err := r.get(r.nextIndex, newNextIndex-1)
	if err != nil {
		return err
	}
	if len(lines) > 0 {
		r.currentBuffer = append(r.currentBuffer, []byte(strings.Join(lines, ""))...)
	}
	r.nextIndex = newNextIndex

	return nil
}

func (r *Reader) Read(p []byte) (n int, err error) {
	lengthToRead := len(p)

	// If we don't have enough bytes in current buffer we will load some line from Redis
	if len(r.currentBuffer) < lengthToRead {
		if err := r.loadMoreLines(); err != nil {
			return 0, err
		}
	}

	// If not more data in the current buffer we should turn an EOF error
	if len(r.currentBuffer) == 0 {
		return 0, io.EOF
	}

	var buffer []byte
	if len(r.currentBuffer) > lengthToRead { // more data, return a subset of current buffer
		buffer = r.currentBuffer[:lengthToRead-1]
		r.currentBuffer = r.currentBuffer[lengthToRead:]
	} else { // return all the current buffer
		buffer = append([]byte{}, r.currentBuffer...)
		r.currentBuffer = nil
	}

	return copy(p, buffer), nil
}
