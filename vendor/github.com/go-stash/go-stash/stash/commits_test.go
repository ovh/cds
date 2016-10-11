package stash

import (
	"testing"
)

func TestGet(t *testing.T) {
	branches, err := client.Branches.List(testProject, testRepo)
	if err != nil {
		t.Errorf("Unexpected error on `client.Branches.List()`, got %v", err)
	}
	branch := branches[0]

	// test valid commit
	commit, err := client.Commits.Get(
		testProject,
		testRepo,
		branch.LatestHash,
	)
	if err != nil {
		t.Errorf("Unexpected error on `client.Commits.Get()`, got %v", err)
	}
	if commit.Hash != branch.LatestHash {
		t.Errorf(
			"Hash does not match, expected %v, got %v",
			branch.LatestHash,
			commit.Hash,
		)
	}

	// test invalid commmit
	commit2, err := client.Commits.Get(testProject, testRepo, "wrong")
	if err != nil {
		t.Errorf("Unexpected error on `client.Commits.Get()`, got %v", err)
	}
	if commit2.Hash != "" {
		t.Errorf("Did not expect commit data, got %v for hash", commit2.Hash)
	}

	// test invalid project
	commit3, err := client.Commits.Get("wrong", testRepo, branch.LatestHash)
	if err == nil {
		t.Errorf("Expected error on `client.Commits.Get()`")
	}
	if commit3 != nil {
		t.Errorf("Did not expect commit data, got %v", commit3)
	}
}
