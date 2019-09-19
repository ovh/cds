package configstore

import (
	"encoding/json"
	"fmt"
	"time"
)

// ItemFilter holds a list of manipulation steps to operate on an item list.
// It can be declared globally then used/applied on specific item lists later.
// By declaring it before actual use, you can make it available to other packages
// which can then use it to describe your filters / configuration (e.g. main for usage).
// See String().
type ItemFilter struct {
	funcs           []func(*ItemList) *ItemList
	initialKeySlice string
	unmarshalType   interface{}
	store           *Store
}

// Filter creates a new empty filter object.
func Filter() *ItemFilter {
	return &ItemFilter{store: DefaultStore}
}

// String returns a description of the filter.
func (s *ItemFilter) String() string {
	if s == nil || s.initialKeySlice == "" {
		return ""
	}

	typeStr := "string"
	if s.unmarshalType != nil {
		j, _ := json.Marshal(s.unmarshalType)
		typeStr = string(j)
	}

	return fmt.Sprintf("%s: %s", s.initialKeySlice, typeStr)
}

/*
 ** GETTERS
 */

func (s *ItemFilter) getStore() *Store {
	if s.store == nil {
		return DefaultStore
	}
	return s.store
}

// GetItem fetches the full item list, applies the filter, then returns a single item by key.
func (s *ItemFilter) GetItem(key string) (Item, error) {
	items, err := s.GetItemList()
	if err != nil {
		return Item{}, err
	}
	return items.GetItem(key)
}

// MustGetItem is similar to GetItem, but always returns an Item object.
// The eventual error is stored inside, and returned when accessing the item's value.
// Useful for chaining calls.
func (s *ItemFilter) MustGetItem(key string) Item {
	i, err := s.GetItem(key)
	if err != nil {
		return Item{unmarshalErr: err}
	}
	return i
}

// GetFirstItem fetches the full item list, applies the filter, then returns the first item of the list.
func (s *ItemFilter) GetFirstItem() (Item, error) {
	items, err := s.GetItemList()
	if err != nil {
		return Item{}, err
	}
	if len(items.Items) == 0 {
		sliceKey := s.initialKeySlice
		if sliceKey == "" {
			sliceKey = "[NONE]"
		}
		return Item{}, ErrItemNotFound(fmt.Sprintf("configstore: get first item (slice: %s): no item found", sliceKey))
	}
	return items.Items[0], nil
}

// MustGetFirstItem is similar to GetFirstItem, but always returns an Item object.
// The eventual error is stored inside, and returned when accessing the item's value.
// Useful for chaining calls.
func (s *ItemFilter) MustGetFirstItem() Item {
	i, err := s.GetFirstItem()
	if err != nil {
		return Item{unmarshalErr: err}
	}
	return i
}

// GetItemValue fetches the full item list, applies the filter, then returns a single item's value by key.
func (s *ItemFilter) GetItemValue(key string) (string, error) {
	i, err := s.GetItem(key)
	if err != nil {
		return "", err
	}
	return i.Value()
}

// GetItemValueBool fetches the full item list, applies the filter, then returns a single item's value by key.
func (s *ItemFilter) GetItemValueBool(key string) (bool, error) {
	i, err := s.GetItem(key)
	if err != nil {
		return false, err
	}
	return i.ValueBool()
}

// GetItemValueFloat fetches the full item list, applies the filter, then returns a single item's value by key.
func (s *ItemFilter) GetItemValueFloat(key string) (float64, error) {
	i, err := s.GetItem(key)
	if err != nil {
		return 0, err
	}
	return i.ValueFloat()
}

// GetItemValueInt fetches the full item list, applies the filter, then returns a single item's value by key.
func (s *ItemFilter) GetItemValueInt(key string) (int64, error) {
	i, err := s.GetItem(key)
	if err != nil {
		return 0, err
	}
	return i.ValueInt()
}

// GetItemValueUint fetches the full item list, applies the filter, then returns a single item's value by key.
func (s *ItemFilter) GetItemValueUint(key string) (uint64, error) {
	i, err := s.GetItem(key)
	if err != nil {
		return 0, err
	}
	return i.ValueUint()
}

// GetItemValueDuration fetches the full item list, applies the filter, then returns a single item's value by key.
func (s *ItemFilter) GetItemValueDuration(key string) (time.Duration, error) {
	i, err := s.GetItem(key)
	if err != nil {
		return time.Duration(0), err
	}
	return i.ValueDuration()
}

// GetItemList fetches the full item list, applies the filter, and returns the result.
func (s *ItemFilter) GetItemList() (*ItemList, error) {
	items, err := s.getStore().GetItemList()
	if err != nil {
		return nil, err
	}
	return s.Apply(items), nil
}

// Apply applies the filter on an existing item list.
func (s *ItemFilter) Apply(items *ItemList) *ItemList {
	if s == nil {
		return items
	}
	filtered := items
	for _, f := range s.funcs {
		filtered = f(filtered)
	}
	return filtered
}

/*
 ** LIST MANIPULATION
 */

// chained calls return a copy of the filter.
// all objects (filter + list + item) are immutable.
func copyItemFilter(s *ItemFilter) *ItemFilter {
	ret := Filter()
	if s != nil {
		ret.funcs = s.funcs
		ret.unmarshalType = s.unmarshalType
		ret.initialKeySlice = s.initialKeySlice
		ret.store = s.store
	}
	return ret
}

// Store lets you specify which store instance to use for functions GetItemList, GetItem, ...
func (s *ItemFilter) Store(st *Store) *ItemFilter {

	if st == nil {
		return s
	}

	s = copyItemFilter(s)
	s.store = st

	return s
}

// Slice filters the list items, keeping only those matching key.
// You can optionally pass a list of modifier functions, to be invoked when applying the filter.
func (s *ItemFilter) Slice(key string, keyF ...func(string) string) *ItemFilter {

	key = transformKey(key)

	s = copyItemFilter(s)

	if s.initialKeySlice == "" {
		s.initialKeySlice = key
	}

	s.funcs = append(s.funcs, func(s *ItemList) *ItemList {
		keyLocal := key
		for _, f := range keyF {
			keyLocal = f(keyLocal)
		}
		newList := make([]Item, len(s.indexed[keyLocal]))
		copy(newList, s.indexed[keyLocal])
		return (&ItemList{Items: newList}).index()
	})

	return s
}

// Rekey modifies item keys. The function parameter is called for each item in the item list, and the returned string
// is used as the new key.
func (s *ItemFilter) Rekey(rekeyF func(*Item) string) *ItemFilter {
	return s.mapFunc(func(sec *Item) Item {
		return Item{
			key:          transformKey(rekeyF(sec)),
			value:        sec.value,
			priority:     sec.priority,
			unmarshaled:  sec.unmarshaled,
			unmarshalErr: sec.unmarshalErr,
		}
	})
}

// Reorder modifies item priority. The function parameter is called for each item in the item list, and the returned integer
// is used as the new priority.
func (s *ItemFilter) Reorder(reorderF func(*Item) int64) *ItemFilter {
	return s.mapFunc(func(sec *Item) Item {
		return Item{
			key:          sec.key,
			value:        sec.value,
			priority:     reorderF(sec),
			unmarshaled:  sec.unmarshaled,
			unmarshalErr: sec.unmarshalErr,
		}
	})
}

// Transform modifies item values. The function parameter is called for each item in the item list, and the returned string + error
// are the values which will be returned by item.Value().
func (s *ItemFilter) Transform(transformF func(*Item) (string, error)) *ItemFilter {
	return s.mapFunc(func(sec *Item) Item {
		if sec.unmarshalErr != nil {
			return *sec
		}
		tr, err := transformF(sec)
		return Item{
			key:          sec.key,
			value:        tr,
			priority:     sec.priority,
			unmarshaled:  sec.unmarshaled,
			unmarshalErr: err,
		}
	})
}

// Unmarshal tries to unmarshal (from JSON or YAML) all the items in the item list into objects returned by the factory f().
// The results and errors will be stored to be handled later. See item.Unmarshaled().
func (s *ItemFilter) Unmarshal(f func() interface{}) *ItemFilter {

	s = copyItemFilter(s)

	if f == nil {
		return s
	}

	if s.unmarshalType == nil {
		s.unmarshalType = f()
	}

	s.funcs = append(s.funcs, func(s *ItemList) *ItemList {
		ret := &ItemList{Items: make([]Item, 0, len(s.Items))}
		for i := range s.Items {
			ret.Items = append(ret.Items, s.Items[i])
			ret.Items[i].storeUnmarshal(f())
		}
		return ret.index()
	})

	return s
}

// Implementation of the map logic for Rekey/Reorder/... public functions
func (s *ItemFilter) mapFunc(mapF func(*Item) Item) *ItemFilter {

	s = copyItemFilter(s)

	s.funcs = append(s.funcs, func(s *ItemList) *ItemList {
		ret := &ItemList{}

		for _, sec := range s.Items {
			ret.Items = append(ret.Items, mapF(&sec))
		}

		return ret.index()
	})

	return s
}

// Squash filters the items in the item list, keeping only the items with the highest priority for each key.
func (s *ItemFilter) Squash() *ItemFilter {

	s = copyItemFilter(s)

	s.funcs = append(s.funcs, func(s *ItemList) *ItemList {
		ret := &ItemList{}
		for _, l := range s.indexed {
			highest := int64(0)
			if len(l) > 0 {
				highest = l[0].priority
			}
			for _, sec := range l {
				if sec.priority >= highest {
					ret.Items = append(ret.Items, sec)
				}
			}
		}
		return ret.index()
	})

	return s
}
