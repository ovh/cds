package redis

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

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
	size, _ := s.store.SetCard(i.ID)
	return size > 0, nil
}

func (s *Redis) Add(i storage.ItemUnit, index uint, value string) error {
	value = strconv.Itoa(int(index)) + "#" + value
	return s.store.ScoredSetAdd(context.Background(), i.ItemID, value, float64(index))
}

func (s *Redis) Append(i storage.ItemUnit, value string) error {
	return s.store.ScoredSetAppend(context.Background(), i.ItemID, value)
}

func (s *Redis) Card(i storage.ItemUnit) (int, error) {
	return s.store.SetCard(i.ItemID)
}

func (s *Redis) Get(i storage.ItemUnit, from, to uint) ([]string, error) {
	var res = make([]string, to-from+1)
	if err := s.store.ScoredSetScan(context.Background(), i.ItemID, float64(from), float64(to), &res); err != nil {
		return res, err
	}
	for i := range res {
		res[i] = strings.TrimFunc(res[i], unicode.IsNumber)
		res[i] = strings.TrimPrefix(res[i], "#")
	}
	return res, nil
}

// NewReader instanciate a reader that it able to iterate over Redis storage unit
// with a score step of 100.0, starting at score 0
func (s *Redis) NewReader(i storage.ItemUnit) (io.ReadCloser, error) {
	return &reader{s: s, i: i}, nil
}

func (s *Redis) Read(i storage.ItemUnit, r io.Reader, w io.Writer) error {
	_, err := io.Copy(w, r)
	return err
}

type reader struct {
	s             *Redis
	i             storage.ItemUnit
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
