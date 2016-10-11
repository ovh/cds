package stash

import (
	"testing"
)

func TestReposList(t *testing.T) {
	repos, err := client.Repos.List()
	if err != nil {
		t.Errorf("Did not expect an error, got: %v", err)
	}
	if len(repos) != 1 {
		t.Errorf("Expected 1 repo back, got: %v", len(repos))
	}
	repo := repos[0]
	if repo.Slug != testRepo {
		t.Errorf("repo slug [%v]; want [%v]", repo.Slug, testRepo)
	}
	if repo.Project.Key != testProject {
		t.Errorf("repo project [%v]; want [%v]", repo.Project, testProject)
	}
}

func TestReposFind(t *testing.T) {
	repo, err := client.Repos.Find(testProject, testRepo)
	if err != nil {
		t.Error(err)
	}
	if repo.Slug != testRepo {
		t.Errorf("repo slug [%v]; want [%v]", repo.Slug, testRepo)
	}
	if repo.Project.Key != testProject {
		t.Errorf("repo project [%v]; want [%v]", repo.Project, testProject)
	}
}

func TestReposFindNot(t *testing.T) {
	repo, err := client.Repos.Find(testProject, "wrong")
	if err == nil {
		t.Error("Expected an error")
	}
	if repo != nil {
		t.Errorf("Did not expect a repo, got %v", repo)
	}
}
