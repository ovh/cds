package redis

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/sdk"
)

type Redis struct {
	storage.AbstractUnit
	config storage.RedisBufferConfiguration
	store  cache.ScoredSetStore
}

var _ storage.BufferUnit = new(Redis)

func init() {
	storage.RegisterDriver("redis", new(Redis))
}

func (s *Redis) Init(cfg interface{}) error {
	config, is := cfg.(storage.RedisBufferConfiguration)
	if !is {
		return sdk.WithStack(fmt.Errorf("invalid configuration: %T", cfg))
	}
	s.config = config
	var err error
	s.store, err = cache.New(s.config.Host, s.config.Password, 60)
	return err
}

func (s *Redis) ItemExists(i index.Item) (bool, error) {
	return false, nil
}

func (s *Redis) Add(i index.Item, index uint, value string) error {
	return s.store.ScoredSetAdd(context.Background(), i.ID, value, float64(index))
}

func (s *Redis) Append(i index.Item, value string) error {
	return s.store.ScoredAppend(context.Background(), i.ID, value)
}

func (s *Redis) Get(i index.Item, from, to uint) ([]string, error) {
	var res = make([]string, to-from+1)
	err := s.store.ScoredSetScan(context.Background(), i.ID, float64(from), float64(to), &res)
	return res, err
}

// NewReader instanciate a reader that it able to iterate over Redis storage unit
// with a score step of 100.0, starting at score 0
func (s *Redis) NewReader(i index.Item) (io.ReadCloser, error) {
	return &reader{s: s, i: i}, nil
}

type reader struct {
	s             *Redis
	i             index.Item
	lastIndex     uint
	currentBuffer string
}

func (r *reader) Read(p []byte) (n int, err error) {
	size := len(p)
	var buffer string
	if len(r.currentBuffer) > 0 {
		if len(r.currentBuffer) <= size {
			buffer = r.currentBuffer
		}
	}

	newIndex := r.lastIndex + 100
	lines, err := r.s.Get(r.i, r.lastIndex, newIndex)
	if err != nil {
		return 0, err
	}
	if len(lines) > 0 {
		if r.currentBuffer == "" {
			r.currentBuffer += strings.Join(lines, "\n")
		} else {
			r.currentBuffer += "\n" + strings.Join(lines, "\n")
		}
	}

	if len(buffer) < size && len(r.currentBuffer) > 0 {
		x := size - len(buffer)
		if x < len(r.currentBuffer) {
			buffer += r.currentBuffer[:x]
			r.currentBuffer = r.currentBuffer[x:]
		} else {
			buffer += r.currentBuffer
			r.currentBuffer = ""
		}
	}

	r.lastIndex = newIndex
	err = nil
	if len(lines) == 0 {
		err = io.EOF
	}

	return copy(p, buffer), err
}

func (r *reader) Close() error {
	return nil
}
