package configstore

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
)

/*
** DEFAULT PROVIDERS IMPLEMENTATION
 */

func errorProvider(s *Store, name string, err error) {
	s.RegisterProvider(name, newErrorProvider(err))
}

func newErrorProvider(err error) Provider {
	return func() (ItemList, error) {
		return ItemList{}, err
	}
}

func fileProvider(s *Store, filename string) {
	file(s, filename, false, nil)
}

func fileRefreshProvider(s *Store, filename string) {
	file(s, filename, true, nil)
}

func fileCustomProvider(s *Store, filename string, fn func([]byte) ([]Item, error)) {
	file(s, filename, false, fn)
}

func fileCustomRefreshProvider(s *Store, filename string, fn func([]byte) ([]Item, error)) {
	file(s, filename, true, fn)
}

func file(s *Store, filename string, refresh bool, fn func([]byte) ([]Item, error)) {

	if filename == "" {
		return
	}

	providername := fmt.Sprintf("file:%s", filename)

	last := time.Now()
	vals, err := readFile(filename, fn)
	if err != nil {
		errorProvider(s, providername, err)
		return
	}
	inmem := inMemoryProvider(s, providername)
	logrus.Infof("Configuration from file: %s", filename)
	inmem.Add(vals...)

	if refresh {
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			for range ticker.C {
				finfo, err := os.Stat(filename)
				if err != nil {
					continue
				}
				if finfo.ModTime().After(last) {
					last = finfo.ModTime()
				} else {
					continue
				}
				vals, err := readFile(filename, fn)
				if err != nil {
					continue
				}
				inmem.mut.Lock()
				inmem.items = vals
				inmem.mut.Unlock()
				s.NotifyWatchers()
			}
		}()
	}
}

func fileTreeProvider(s *Store, dirname string) {

	if dirname == "" {
		return
	}

	providername := fmt.Sprintf("filetree:%s", dirname)

	files, err := ioutil.ReadDir(dirname)
	if err != nil {
		errorProvider(s, providername, err)
		return
	}

	items := []Item{}

	for _, f := range files {
		filename := filepath.Join(dirname, f.Name())

		if f.IsDir() {
			items, err = browseDir(items, filename, f.Name())
			if err != nil {
				errorProvider(s, providername, err)
				return
			}
		} else {
			it, err := readItem(filename, f.Name(), f.Name())
			if err != nil {
				errorProvider(s, providername, err)
				return
			}
			items = append(items, it)
		}
	}

	inmem := inMemoryProvider(s, providername)
	for _, it := range items {
		inmem.Add(it)
	}
}

func fileListProvider(s *Store, dirname string) {

	if dirname == "" {
		return
	}

	files, err := ioutil.ReadDir(dirname)
	if err != nil {
		errorProvider(s, fmt.Sprintf("filelist:%s", dirname), err)
		return
	}

	for _, file := range files {
		if !file.IsDir() {
			fileProvider(s, filepath.Join(dirname, file.Name()))
		}
	}
}

func browseDir(items []Item, path, basename string) ([]Item, error) {

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return items, err
	}

	for _, f := range files {
		filename := filepath.Join(path, f.Name())
		if f.IsDir() {
			return items, fmt.Errorf("subdir %s: encountered nested directory %s, max 1 level of nesting", basename, f.Name())
		}
		it, err := readItem(filename, f.Name(), basename)
		if err != nil {
			return items, err
		}
		items = append(items, it)
	}

	return items, nil
}

func readItem(path, basename, itemKey string) (Item, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return Item{}, err
	}
	priority := int64(5)
	first, _ := utf8.DecodeRuneInString(basename)
	if unicode.IsUpper(first) {
		priority = 10
	}
	return NewItem(itemKey, string(content), priority), nil
}

func readFile(filename string, fn func([]byte) ([]Item, error)) ([]Item, error) {
	vals := []Item{}
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	if fn != nil {
		return fn(b)
	}
	err = yaml.Unmarshal(b, &vals)
	if err != nil {
		return nil, err
	}
	return vals, nil
}

func inMemoryProvider(s *Store, name string) *InMemoryProvider {
	inmem := &InMemoryProvider{}
	s.RegisterProvider(name, inmem.Items)
	return inmem
}

// InMemoryProvider implements an in-memory configstore provider.
type InMemoryProvider struct {
	items []Item
	mut   sync.Mutex
}

// Add appends an item to the in-memory list.
func (inmem *InMemoryProvider) Add(s ...Item) *InMemoryProvider {
	inmem.mut.Lock()
	defer inmem.mut.Unlock()
	inmem.items = append(inmem.items, s...)
	return inmem
}

// Items returns the in-memory item list. This is the function that gets called by configstore.
func (inmem *InMemoryProvider) Items() (ItemList, error) {
	inmem.mut.Lock()
	defer inmem.mut.Unlock()
	return ItemList{Items: inmem.items}, nil
}

func envProvider(s *Store, prefix string) {

	if prefix != "" && !strings.HasSuffix(prefix, "_") {
		prefix += "_"
	}

	prefixName := strings.ToUpper(prefix)
	if prefixName == "" {
		prefixName = "all"
	}
	inmem := inMemoryProvider(s, fmt.Sprintf("env:%s", prefixName))

	prefix = transformKey(prefix)

	for _, e := range os.Environ() {
		ePair := strings.SplitN(e, "=", 2)
		if len(ePair) <= 1 {
			continue
		}
		eTr := transformKey(ePair[0])
		if strings.HasPrefix(eTr, prefix) {
			inmem.Add(NewItem(strings.TrimPrefix(eTr, prefix), ePair[1], 15))
		}
	}
}
