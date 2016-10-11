package stash

import (
	"testing"
)

func TestContentFind(t *testing.T) {
	content, err := client.Contents.Find(testProject, testRepo, "README.md")
	if err != nil {
		t.Errorf("Unexpected error on `client.Contents.Find()`, got %v", err)
	}
	if content == "" {
		t.Errorf("Expected some content, got nothing")
	}
}

func TestContentFindEmpty(t *testing.T) {
	content, err := client.Contents.Find(testProject, testRepo, "random")
	if err == nil {
		t.Errorf("Expected an error on `client.Contents.Find()`")
	}
	if content != "" {
		t.Errorf("Did not expect any content, got %v", content)
	}
}

func TestContentFindInvalid(t *testing.T) {
	content, err := client.Contents.Find("wrong", testRepo, "README.md")
	if err == nil {
		t.Error("Expected an error on `client.Contents.Find()`")
	}
	if content != "" {
		t.Errorf("Did not expect any content, got %v", content)
	}
}
