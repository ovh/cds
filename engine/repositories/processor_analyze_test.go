package repositories

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

// execGitIn runs a git command in dir with a neutral identity and signing
// disabled, so tests do not depend on the developer's global git config.
func execGitIn(t *testing.T, dir string, args ...string) string {
	allArgs := append([]string{"-C", dir,
		"-c", "user.name=test", "-c", "user.email=test@test.local",
		"-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false",
		"-c", "init.defaultbranch=master"}, args...)
	out, err := exec.Command("git", allArgs...).CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, string(out))
	return strings.TrimSpace(string(out))
}

// createFixtureRepo builds a small local repository allowing partial clone,
// used as the origin of bare analysis cache tests.
func createFixtureRepo(t *testing.T) string {
	dir := filepath.Join(t.TempDir(), "fixture")
	require.NoError(t, os.MkdirAll(dir, os.FileMode(0755)))
	execGitIn(t, dir, "init", "-q")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# fixture"), os.FileMode(0644)))
	execGitIn(t, dir, "add", ".")
	execGitIn(t, dir, "commit", "-q", "-m", "initial commit")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), os.FileMode(0644)))
	execGitIn(t, dir, "add", ".")
	execGitIn(t, dir, "commit", "-q", "-m", "second commit")
	execGitIn(t, dir, "config", "uploadpack.allowFilter", "true")
	return dir
}

func Test_processGitCloneBare(t *testing.T) {
	fixture := createFixtureRepo(t)
	s := &Service{}
	s.Cfg.Basedir = t.TempDir()
	op := sdk.Operation{URL: "file://" + fixture, RepoFullName: "test/fixture"}

	_, err := s.processGitCloneBare(context.TODO(), &op)
	require.NoError(t, err)

	require.NotNil(t, op.RepositoryInfo)
	assert.Equal(t, "test/fixture", op.RepositoryInfo.Name)
	assert.Equal(t, "master", op.RepositoryInfo.DefaultBranch)
	assert.Contains(t, op.RepositoryInfo.FetchURL, fixture)

	path := s.BareRepo(op).Basedir
	assert.Equal(t, "true", execGitIn(t, path, "rev-parse", "--is-bare-repository"))
	assert.Equal(t, "blob:none", execGitIn(t, path, "config", "--get", "remote.origin.partialclonefilter"))

	// Second call must reopen the cached clone, not fail on a non-empty directory
	op.RepositoryInfo = nil
	_, err = s.processGitCloneBare(context.TODO(), &op)
	require.NoError(t, err)
	require.NotNil(t, op.RepositoryInfo)
}

func Test_fetchAnalysisTarget(t *testing.T) {
	fixture := createFixtureRepo(t)
	s := &Service{}
	s.Cfg.Basedir = t.TempDir()
	op := sdk.Operation{URL: "file://" + fixture, RepoFullName: "test/fixture"}

	gitRepo, err := s.processGitCloneBare(context.TODO(), &op)
	require.NoError(t, err)
	clonePath := s.BareRepo(op).Basedir

	// Branch created after the clone, with a changeset targeting master
	execGitIn(t, fixture, "checkout", "-q", "-b", "feat")
	require.NoError(t, os.WriteFile(filepath.Join(fixture, "feat.go"), []byte("package feat"), os.FileMode(0644)))
	execGitIn(t, fixture, "add", ".")
	execGitIn(t, fixture, "commit", "-q", "-m", "feat commit")

	op.Setup.Checkout.Branch = "feat"
	op.Setup.Checkout.GetChangeSet = true
	op.Setup.Checkout.ChangeSetBranchTo = "master"
	target, err := s.fetchAnalysisTarget(context.TODO(), gitRepo, &op)
	require.NoError(t, err)
	assert.Equal(t, "refs/heads/feat", target)
	assert.Equal(t, execGitIn(t, fixture, "rev-parse", "feat"), execGitIn(t, clonePath, "rev-parse", "refs/heads/feat"))
	assert.NotEmpty(t, execGitIn(t, clonePath, "rev-parse", "refs/heads/master"), "changeset target branch must be fetched too")

	// An explicit commit takes priority over the branch ref
	sha := execGitIn(t, fixture, "rev-parse", "feat")
	op.Setup.Checkout.Commit = sha
	target, err = s.fetchAnalysisTarget(context.TODO(), gitRepo, &op)
	require.NoError(t, err)
	assert.Equal(t, sha, target)

	// Tag operation
	execGitIn(t, fixture, "tag", "v1.0.0")
	opTag := sdk.Operation{URL: op.URL, RepositoryInfo: op.RepositoryInfo}
	opTag.Setup.Checkout.Tag = "v1.0.0"
	target, err = s.fetchAnalysisTarget(context.TODO(), gitRepo, &opTag)
	require.NoError(t, err)
	assert.Equal(t, "refs/tags/v1.0.0", target)
	assert.NotEmpty(t, execGitIn(t, clonePath, "rev-parse", "refs/tags/v1.0.0"))

	// Empty branch falls back to the default branch
	opDefault := sdk.Operation{URL: op.URL, RepositoryInfo: op.RepositoryInfo}
	opDefault.Setup.Checkout.ProcessSemver = true
	target, err = s.fetchAnalysisTarget(context.TODO(), gitRepo, &opDefault)
	require.NoError(t, err)
	assert.Equal(t, "refs/heads/master", target)
	assert.Equal(t, "master", opDefault.Setup.Checkout.Branch)
}

func Test_useBareAnalysisCache(t *testing.T) {
	analysisOp := func() sdk.Operation {
		op := sdk.Operation{}
		op.Setup.Checkout.Branch = "master"
		op.Setup.Checkout.CheckSignature = true
		op.Setup.Checkout.ProcessSemver = true
		op.Setup.Checkout.GetChangeSet = true
		return op
	}

	tests := []struct {
		name   string
		flag   bool
		mutate func(op *sdk.Operation)
		want   bool
	}{
		{name: "analysis operation on branch", flag: true, want: true},
		{name: "flag disabled keeps current path", flag: false, want: false},
		{name: "analysis operation on tag", flag: true, mutate: func(op *sdk.Operation) {
			op.Setup.Checkout.Branch = ""
			op.Setup.Checkout.Tag = "v1.0.0"
		}, want: true},
		{name: "single analysis flag is enough", flag: true, mutate: func(op *sdk.Operation) {
			op.Setup.Checkout.CheckSignature = false
			op.Setup.Checkout.GetChangeSet = false
		}, want: true},
		{name: "loadfiles operation keeps current path", flag: true, mutate: func(op *sdk.Operation) {
			op.LoadFiles.Pattern = ".cds/*.yml"
		}, want: false},
		{name: "push operation keeps current path", flag: true, mutate: func(op *sdk.Operation) {
			op.Setup.Push.FromBranch = "cds/update"
		}, want: false},
		{name: "no branch nor tag keeps current path", flag: true, mutate: func(op *sdk.Operation) {
			op.Setup.Checkout.Branch = ""
		}, want: false},
		{name: "no analysis flag keeps current path", flag: true, mutate: func(op *sdk.Operation) {
			op.Setup.Checkout.CheckSignature = false
			op.Setup.Checkout.ProcessSemver = false
			op.Setup.Checkout.GetChangeSet = false
			op.Setup.Checkout.GetMessage = true
		}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{}
			s.Cfg.BareAnalysisCache = tt.flag
			op := analysisOp()
			if tt.mutate != nil {
				tt.mutate(&op)
			}
			assert.Equal(t, tt.want, s.useBareAnalysisCache(op))
		})
	}
}
