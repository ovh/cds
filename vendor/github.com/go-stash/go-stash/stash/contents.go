package stash

import (
	"fmt"
	"strings"
)

type Lines struct {
	Text string `"json:text"`
}

type Content struct {
	Lines []Lines `"json:lines"`
}

type ContentResource struct {
	client *Client
}

// Get content data for file
func (r *ContentResource) Find(project, slug, path string) (string, error) {
	content := Content{}
	var file []string

	url_path := fmt.Sprintf("/projects/%s/repos/%s/browse/%s", project,
		slug, path)

	if err := r.client.do("GET", "core", url_path, nil, nil, &content); err != nil {
		return "", err
	}

	for i := range content.Lines {
		file = append(file, content.Lines[i].Text)
	}

	return strings.Join(file, "\n"), nil
}
