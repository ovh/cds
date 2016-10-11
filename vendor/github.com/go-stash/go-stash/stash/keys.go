package stash

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Key struct {
	Id    int64  `"json:id"`
	Text  string `"json:text"`
	Label string `"json:label"`
}

type Keys struct {
	Values []Key `"json:values"`
}

type KeyResource struct {
	client *Client
}

func (r *KeyResource) Create(key string) (*Key, error) {
	newKey := map[string]string{"text": key}
	values, err := json.Marshal(newKey)
	if err != nil {
		return nil, err
	}

	k := Key{}
	path := fmt.Sprintf("/keys")
	if err := r.client.do("POST", "ssh", path, nil, values, &k); err != nil {
		return nil, err
	}

	return &k, nil
}

func (r *KeyResource) Find(key string) (*Key, error) {
	keys := Keys{}
	path := fmt.Sprintf("/keys")
	if err := r.client.do("GET", "ssh", path, nil, nil, &keys); err != nil {
		return nil, err
	}

	for _, k := range keys.Values {
		if strings.TrimSpace(k.Text) == strings.TrimSpace(key) {
			return &k, nil
		}
	}

	return nil, ErrNotFound
}

func (r *KeyResource) CreateUpdate(key string) (*Key, error) {
	if found, err := r.Find(key); err == nil {
		return found, nil
	}

	return r.Create(key)
}
