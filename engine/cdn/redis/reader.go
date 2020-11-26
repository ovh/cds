package redis

import (
	"io"
	"sort"

	"github.com/ovh/cds/sdk"
)

var _ io.ReadCloser = new(Reader)

type Reader struct {
	ReadWrite
	nextIndex     uint
	From          int64 // the offset that we want to use when reading lines from Redis, allows negative value to get last lines
	Size          uint  // the count of lines that we want to read (0 means to the end)
	currentBuffer []byte
	Format        sdk.CDNReaderFormat
	readEOF       bool
	Sort          int64 // < 0 for latest logs first, >= 0 for older logs first
}

func (r *Reader) loadMoreLines() error {
	// Use max score to get line max count instead of card in case of missing rows
	maxScore, err := r.maxScore()
	if err != nil {
		return err
	}
	lineCount := uint(maxScore) + 1

	// If from is less than 0 try to substract given value to lines count
	if r.From < 0 {
		r.From = int64(lineCount) + r.From
		if r.From < 0 {
			r.From = 0
		}
	}

	maxLinesToRead := uint(lineCount) - uint(r.From)
	if r.Size == 0 || r.Size > maxLinesToRead {
		r.Size = maxLinesToRead
	}

	isFirstRead := r.nextIndex == 0
	// If its first read, init the next index with 'from' value
	if isFirstRead {
		r.nextIndex = uint(r.From) // 'from' can be 0 but not < 0 at this point
	}

	// If first read and json format init json list, also define formatter to exec before append lines and at read end
	if isFirstRead && r.Format == sdk.CDNReaderFormatJSON {
		r.currentBuffer = []byte("[")
	}
	formatBeforeLine := func() {
		// For json format, if not first read we should add a comma before each line object
		if r.Format == sdk.CDNReaderFormatJSON {
			r.currentBuffer = append(r.currentBuffer, []byte(",")...)
		}
	}
	formatEnd := func() {
		if r.Format == sdk.CDNReaderFormatJSON {
			r.currentBuffer = append(r.currentBuffer, []byte("]")...)
		}
	}

	// Read 100 lines if possible or only the missing lines if less than 100
	alreadyReadLinesLength := r.nextIndex - uint(r.From)
	linesLeftToRead := uint(r.Size) - alreadyReadLinesLength
	if linesLeftToRead == 0 {
		if !r.readEOF {
			r.readEOF = true
			formatEnd()
		}
		return nil
	}
	var newNextIndex uint
	if linesLeftToRead > 100 {
		newNextIndex = r.nextIndex + 100
	} else {
		newNextIndex = r.nextIndex + linesLeftToRead
	}

	// Get new lines from Redis and append it to current buffer
	var from, to uint
	if r.Sort >= 0 {
		from = r.nextIndex
		to = newNextIndex - 1
	} else {
		if uint(lineCount) < newNextIndex {
			from = 0
		} else {
			from = uint(lineCount) - newNextIndex
		}
		if uint(lineCount) < r.nextIndex {
			to = uint(lineCount) - 1
		} else {
			to = uint(lineCount) - (r.nextIndex + 1)
		}
	}
	lines, err := r.get(from, to)
	if err != nil {
		return err
	}

	if r.Sort < 0 {
		sort.Slice(lines, func(i, j int) bool { return lines[i].Number > lines[j].Number })
	}

	for i := range lines {
		buf, err := lines[i].Format(r.Format)
		if err != nil {
			return err
		}
		if !(isFirstRead && i == 0) {
			formatBeforeLine()
		}
		r.currentBuffer = append(r.currentBuffer, buf...)
	}
	if len(lines) == 0 && !r.readEOF {
		r.readEOF = true
		formatEnd()
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
		buffer = r.currentBuffer[:lengthToRead]
		r.currentBuffer = r.currentBuffer[lengthToRead:]
	} else { // return all the current buffer
		buffer = append([]byte{}, r.currentBuffer...)
		r.currentBuffer = nil
	}

	return copy(p, buffer), nil
}
