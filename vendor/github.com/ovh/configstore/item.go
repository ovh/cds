package configstore

import (
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"github.com/ghodss/yaml"
)

// Item is a key/value pair with a priority attached.
// The initial priority is set by the provider, but can be modified (see Reorder).
type Item struct {
	key          string
	value        string
	priority     int64
	unmarshaled  interface{}
	unmarshalErr error
}

// Strictly used for unmarshaling, bypassing the fact that a Item properties are private
type jsonItem struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Priority int64  `json:"priority"`
}

func transformKey(k string) string {
	k = strings.ToLower(k)
	k = strings.Replace(k, "_", "-", -1)
	return k
}

// NewItem creates a item object from key / value / priority values.
// It is meant to be used by provider implementations.
func NewItem(key, value string, priority int64) Item {
	return Item{key: transformKey(key), value: value, priority: priority}
}

// UnmarshalJSON respects json.Unmarshaler
func (s *Item) UnmarshalJSON(b []byte) error {
	j := &jsonItem{}
	err := yaml.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	s.key = transformKey(j.Key)
	s.value = j.Value
	s.priority = j.Priority
	return nil
}

// Key returns the item key.
func (s *Item) Key() string {
	return s.key
}

// Value returns the item value, along with any error that was encountered in list processing (unmarshal, transform).
func (s Item) Value() (string, error) {
	return s.value, s.unmarshalErr
}

// ValueBool returns the item value, along with any error that was encountered in list processing (unmarshal, transform).
func (s Item) ValueBool() (bool, error) {
	if s.unmarshalErr != nil {
		return false, s.unmarshalErr
	}

	return strconv.ParseBool(s.value)
}

// ValueFloat returns the item value, along with any error that was encountered in list processing (unmarshal, transform).
func (s Item) ValueFloat() (float64, error) {
	if s.unmarshalErr != nil {
		return 0, s.unmarshalErr
	}

	return strconv.ParseFloat(s.value, 64)
}

// ValueInt returns the item value, along with any error that was encountered in list processing (unmarshal, transform).
func (s Item) ValueInt() (int64, error) {
	if s.unmarshalErr != nil {
		return 0, s.unmarshalErr
	}

	return strconv.ParseInt(s.value, 10, 64)
}

// ValueUint returns the item value, along with any error that was encountered in list processing (unmarshal, transform).
func (s Item) ValueUint() (uint64, error) {
	if s.unmarshalErr != nil {
		return 0, s.unmarshalErr
	}

	return strconv.ParseUint(s.value, 10, 64)
}

// ValueDuration returns the item value, along with any error that was encountered in list processing (unmarshal, transform).
func (s Item) ValueDuration() (time.Duration, error) {
	if s.unmarshalErr != nil {
		return time.Duration(0), s.unmarshalErr
	}

	return time.ParseDuration(s.value)
}

// ValueBytes returns the item value, along with any error that was encountered in list processing (unmarshal, transform).
// Data to be returned should be base64 encoded
func (s Item) ValueBytes() ([]byte, error) {
	if s.unmarshalErr != nil {
		return nil, s.unmarshalErr
	}

	return base64.StdEncoding.DecodeString(s.value)
}

// Priority returns the item priority.
func (s Item) Priority() int64 {
	return s.priority
}

// Tries to unmarshal (from JSON or YAML) the item value into i.
// The result and error are stored within the item object, to be handled later.
func (s *Item) storeUnmarshal(i interface{}) {
	if s.unmarshalErr != nil {
		return
	}
	err := yaml.Unmarshal([]byte(s.value), i)
	if err != nil {
		s.unmarshalErr = err
		return
	}
	s.unmarshaled = i
}

// Unmarshaled returns the unmarshaled object produced by ItemFilter.Unmarshal, along with any error
// that was encountered in list processing (unmarshal, transform).
func (s Item) Unmarshaled() (interface{}, error) {
	return s.unmarshaled, s.unmarshalErr
}
