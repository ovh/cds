package sdk

import (
	"fmt"
	"net/url"
	"strings"
)

// Cache define a file needed to be save for cache
type Cache struct {
	ID      int64  `json:"id" cli:"id"`
	Project string `json:"project"`
	Name    string `json:"name" cli:"name"`
	Tag     string `json:"tag"`

	DownloadHash     string   `json:"download_hash" cli:"download_hash"`
	Size             int64    `json:"size,omitempty" cli:"size"`
	Perm             uint32   `json:"perm,omitempty"`
	MD5sum           string   `json:"md5sum,omitempty" cli:"md5sum"`
	ObjectPath       string   `json:"object_path,omitempty"`
	TempURL          string   `json:"temp_url,omitempty"`
	TempURLSecretKey string   `json:"-"`
	Files            []string `json:"files"`
}

//GetName returns the name the artifact
func (c *Cache) GetName() string {
	return c.Name
}

//GetPath returns the path of the artifact
func (c *Cache) GetPath() string {
	container := fmt.Sprintf("%s-%s", c.Project, c.Tag)
	container = url.QueryEscape(container)
	container = strings.Replace(container, "/", "-", -1)
	return container
}
