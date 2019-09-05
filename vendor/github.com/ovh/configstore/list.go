package configstore

import (
	"fmt"
	"sort"
	"time"
)

// ItemList is a list of items which can be manipulated by an ItemFilter
type ItemList struct {
	Items   []Item
	indexed map[string][]Item
}

// Keys returns a list of the different keys present in the item list.
func (s *ItemList) Keys() []string {
	if s == nil {
		return nil
	}

	ret := []string{}
	for k := range s.indexed {
		ret = append(ret, k)
	}

	return ret
}

// GetItem returns a single item, by key.
// If 0 or >=2 items are present with that key, it will return an error.
func (s *ItemList) GetItem(key string) (Item, error) {

	if s == nil {
		return Item{}, ErrUninitializedItemList(fmt.Sprintf("configstore: get '%s': non-initialized item list", key))
	}

	l := (&ItemFilter{}).Slice(key).Apply(s)

	switch len(l.Items) {
	case 0:
		return Item{}, ErrItemNotFound(fmt.Sprintf("configstore: get '%s': no item found", key))
	case 1:
		return l.Items[0], nil

	}
	return Item{}, ErrAmbiguousItem(fmt.Sprintf("configstore: get '%s': ambiguous, %d items share that key", key, len(l.Items)))
}

// GetItemValue returns a single item value, by key.
// If 0 or >=2 items are present with that key, it will return an error.
func (s *ItemList) GetItemValue(key string) (string, error) {
	i, err := s.GetItem(key)
	if err != nil {
		return "", err
	}
	return i.Value()
}

// GetItemValueBool returns a single item value, by key.
// If 0 or >=2 items are present with that key, it will return an error.
func (s *ItemList) GetItemValueBool(key string) (bool, error) {
	i, err := s.GetItem(key)
	if err != nil {
		return false, err
	}
	return i.ValueBool()
}

// GetItemValueFloat returns a single item value, by key.
// If 0 or >=2 items are present with that key, it will return an error.
func (s *ItemList) GetItemValueFloat(key string) (float64, error) {
	i, err := s.GetItem(key)
	if err != nil {
		return 0, err
	}
	return i.ValueFloat()
}

// GetItemValueInt returns a single item value, by key.
// If 0 or >=2 items are present with that key, it will return an error.
func (s *ItemList) GetItemValueInt(key string) (int64, error) {
	i, err := s.GetItem(key)
	if err != nil {
		return 0, err
	}
	return i.ValueInt()
}

// GetItemValueUint returns a single item value, by key.
// If 0 or >=2 items are present with that key, it will return an error.
func (s *ItemList) GetItemValueUint(key string) (uint64, error) {
	i, err := s.GetItem(key)
	if err != nil {
		return 0, err
	}
	return i.ValueUint()
}

// GetItemValueDuration returns a single item value, by key.
// If 0 or >=2 items are present with that key, it will return an error.
func (s *ItemList) GetItemValueDuration(key string) (time.Duration, error) {
	i, err := s.GetItem(key)
	if err != nil {
		return time.Duration(0), err
	}
	return i.ValueDuration()
}

// Implements sort.Interface.
// NOT CONCURRENT SAFE.
func (s *ItemList) Len() int {
	return len(s.Items)
}

// Implements sort.Interface
// NOT CONCURRENT SAFE.
func (s *ItemList) Less(i, j int) bool {
	s1 := s.Items[i]
	s2 := s.Items[j]
	return s1.priority > s2.priority
}

// Implements sort.Interface
// NOT CONCURRENT SAFE.
func (s *ItemList) Swap(i, j int) {
	s1 := s.Items[i]
	s2 := s.Items[j]
	s.Items[i] = s2
	s.Items[j] = s1
}

// Indexes the items of the list by key for easy access.
func (s *ItemList) index() *ItemList {
	if s.indexed != nil {
		return s
	}
	sort.Sort(s)
	s.indexed = map[string][]Item{}
	for _, sec := range s.Items {
		s.indexed[sec.key] = append(s.indexed[sec.key], sec)
	}
	return s
}
