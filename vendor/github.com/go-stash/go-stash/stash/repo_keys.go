package stash

import (
	"encoding/json"
	"fmt"
)

type RepoKey struct {
	Key        Key    `"json:key"`
	Permission string `"json:permission"`
}

type Result struct {
	Keys []RepoKey `"json:values"`
}

type RepoKeyResource struct {
	client *Client
}

func (r *RepoKeyResource) Create(project, slug, key, label string) (*Key, error) {
	keyInit := Key{Text: key}
	repoKey := RepoKey{Key: keyInit, Permission: "REPO_READ"}
	values, err := json.Marshal(repoKey)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/projects/%s/repos/%s/ssh", project, slug)
	if err := r.client.do("POST", "keys", path, nil, values, &repoKey); err != nil {
		return nil, err
	}

	return &repoKey.Key, nil
}

func (r *RepoKeyResource) Find(project, slug, key string) (*Key, error) {
	result := Result{}
	path := fmt.Sprintf("/projects/%s/repos/%s/ssh", project, slug)
	if err := r.client.do("GET", "keys", path, nil, nil, &result); err != nil {
		return nil, err
	}

	for _, k := range result.Keys {
		if k.Key.Text == key {
			return &k.Key, nil
		}
	}

	return nil, ErrNotFound
}

func (r *RepoKeyResource) CreateUpdate(project, slug, key, label string) (*Key, error) {
	if found, err := r.Find(project, slug, key); err == nil {
		return found, nil
	}

	return r.Create(project, slug, key, label)
}
