package redis

import (
	"context"
	"io"
	"strconv"
	"strings"

	"github.com/ovh/cds/engine/cache"
)

var _ io.WriteCloser = new(Writer)

type Writer struct {
	Store         cache.ScoredSetStore
	ItemID        string
	PrefixKey     string
	currentScore  uint
	currentBuffer []byte
}

// Add new item in cache + update last usage
func (w *Writer) add(score uint, value string) error {
	itemKey := cache.Key(w.PrefixKey, w.ItemID)
	value = strconv.Itoa(int(score)) + "#" + value
	if err := w.Store.ScoredSetAdd(context.Background(), itemKey, value, float64(score)); err != nil {
		return err
	}
	return nil
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
		if err := w.add(w.currentScore, bufferSplitted[i]+"\n"); err != nil {
			return 0, err
		}
		w.currentScore++
	}

	// We directly return the length of the given buffer cause all given bytes will be stored in redis or in the current buffer
	return len(p), nil
}

// Close will write the end of the buffer to store in case the last line is not ended by \n
func (w *Writer) Close() error {
	if len(w.currentBuffer) > 0 {
		if err := w.add(w.currentScore, string(w.currentBuffer)+"\n"); err != nil {
			return err
		}
	}
	return nil
}
