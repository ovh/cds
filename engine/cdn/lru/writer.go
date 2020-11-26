package lru

import (
	"context"
	"io"
	"time"

	"github.com/ovh/cds/engine/cdn/redis"
)

var _ io.WriteCloser = new(writer)

type writer struct {
	redis.Writer
}

func (w *writer) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// Close the writer will update or set the last usage date for item in lru
func (w *writer) Close() error {
	if err := w.Store.ScoredSetAdd(context.Background(), redisLruKeyCacheKey, w.ItemID, float64(time.Now().UnixNano())); err != nil {
		return err
	}
	return w.Writer.Close()
}
