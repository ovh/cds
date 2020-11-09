package redis

import (
	"io"
	"strings"
)

var _ io.WriteCloser = new(Writer)

type Writer struct {
	ReadWrite
	currentScore  uint
	currentBuffer []byte
}

func (w *Writer) Write(p []byte) (int, error) {
	// Append cnew buffer to current one
	w.currentBuffer = append(w.currentBuffer, p...)

	// Split current buffer by lines
	bufferString := string(w.currentBuffer)
	bufferSplitted := strings.Split(bufferString, "\n")

	// Save all lines except the last one
	for i := 0; i < len(bufferSplitted); i++ {
		// For last part we add the bytes to the current buffer as it can be a partial line
		if i == len(bufferSplitted)-1 {
			w.currentBuffer = []byte(bufferSplitted[i])
			break
		}
		if err := w.ReadWrite.add(w.currentScore, bufferSplitted[i]+"\n"); err != nil {
			return 0, err
		}
		w.currentScore++
	}

	// We directly return the length of the given buffer cause all given bytes will be stored in redis or in the current buffer
	return len(p), nil
}

func (w *Writer) Close() error {
	if len(w.currentBuffer) > 0 {
		if err := w.ReadWrite.add(w.currentScore, string(w.currentBuffer)+"\n"); err != nil {
			return err
		}
	}

	return w.ReadWrite.Close()
}
