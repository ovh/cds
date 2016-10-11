package stash

import (
	"os"
	"reflect"
	"testing"
)

func TestPullRequests(t *testing.T) {
	fromRef := os.Getenv("STASH_FROM_REF")
	if fromRef == "" {
		t.Skipf("STASH_FROM_REF not specified, skipping")
	}

	prTitle := "Pull Request Test"
	pr, err := client.PullRequests.Create(testProject, testRepo, prTitle, fromRef, "refs/heads/master", []string{})
	if err != nil {
		t.Fatalf("unexpected error creating a pull request: %s", err)
	}

	if pr.Title != prTitle {
		t.Fatalf("expected title %q, got %q", prTitle, pr.Title)
	}

	if pr.FromRef.Id != fromRef {
		t.Fatalf("expected reference %q, got %q", fromRef, pr.FromRef.Id)
	}

	prs, err := client.PullRequests.List(testProject, testRepo, "", "", "", "", false, false)
	if err != nil {
		t.Fatalf("unexpected error listing pull requests: %s", err)
	}

	if len(prs) != 1 {
		t.Fatalf("expected %d pull requests, got %d", 1, len(prs))
	}

	if !reflect.DeepEqual(pr, prs[0]) {
		t.Fatalf("expected pull request %+v, got %+v", pr, prs[0])
	}

	gotPr, err := client.PullRequests.Get(testProject, testRepo, pr.Id)
	if err != nil {
		t.Fatalf("unexpected error getting pull request: %s", err)
	}

	if !reflect.DeepEqual(pr, gotPr) {
		t.Fatalf("expected pull request %+v, got %+v", pr, gotPr)
	}
}
