package repositories

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

// execGitIn runs git in dir with a fixed identity and no host config, so
// fixtures do not depend on the developer's git environment.
func execGitIn(t *testing.T, dir string, args ...string) string {
	allArgs := append([]string{"-C", dir,
		"-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false",
		"-c", "init.defaultbranch=master"}, args...)
	cmd := exec.Command("git", allArgs...)
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.local",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.local",
		"EMAIL=test@test.local",
	)
	out, err := cmd.CombinedOutput()
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

	// With an explicit branch the default branch is not resolved (remote round-trip)
	opBranch := sdk.Operation{URL: "file://" + fixture, RepoFullName: "test/fixture"}
	opBranch.Setup.Checkout.Branch = "master"
	_, err = s.processGitCloneBare(context.TODO(), &opBranch)
	require.NoError(t, err)
	assert.Empty(t, opBranch.RepositoryInfo.DefaultBranch)

	// A directory left by an interrupted clone is not a bare repository: it must
	// be discarded and cloned again instead of failing forever
	require.NoError(t, os.RemoveAll(filepath.Join(path, "refs")))
	require.NoError(t, os.WriteFile(filepath.Join(path, "leftover"), []byte("x"), os.FileMode(0644)))
	op.RepositoryInfo = nil
	_, err = s.processGitCloneBare(context.TODO(), &op)
	require.NoError(t, err)
	assert.Equal(t, "true", execGitIn(t, path, "rev-parse", "--is-bare-repository"))
	assert.NoFileExists(t, filepath.Join(path, "leftover"))
}

func Test_processAnalyzeBare(t *testing.T) {
	fixture := createFixtureRepo(t)
	execGitIn(t, fixture, "tag", "v1.0.0")
	// feat, first commit: one rename, one addition
	execGitIn(t, fixture, "checkout", "-q", "-b", "feat")
	execGitIn(t, fixture, "mv", "main.go", "renamed.go")
	require.NoError(t, os.WriteFile(filepath.Join(fixture, "handler.go"), []byte("package handler"), os.FileMode(0644)))
	execGitIn(t, fixture, "add", ".")
	execGitIn(t, fixture, "commit", "-q", "-m", "rework handlers")
	// feat, second commit: one modification
	require.NoError(t, os.WriteFile(filepath.Join(fixture, "README.md"), []byte("# modified"), os.FileMode(0644)))
	execGitIn(t, fixture, "add", ".")
	execGitIn(t, fixture, "commit", "-q", "-m", "update readme")

	featSha := execGitIn(t, fixture, "rev-parse", "feat")
	sinceSha := execGitIn(t, fixture, "rev-parse", "master")
	// A tag named like the target branch, on an older commit: diffs must use the branch
	execGitIn(t, fixture, "tag", "master", "master~1")

	s := &Service{}
	s.Cfg.Basedir = t.TempDir()

	newOp := func() sdk.Operation {
		op := sdk.Operation{URL: "file://" + fixture, RepoFullName: "test/fixture"}
		op.Setup.Checkout.Branch = "feat"
		op.Setup.Checkout.Commit = featSha
		return op
	}

	t.Run("message, semver, signature and push changeset", func(t *testing.T) {
		op := newOp()
		op.Setup.Checkout.GetMessage = true
		op.Setup.Checkout.ProcessSemver = true
		op.Setup.Checkout.CheckSignature = true
		op.Setup.Checkout.GetChangeSet = true
		op.Setup.Checkout.ChangeSetCommitSince = sinceSha

		require.NoError(t, s.processAnalyzeBare(context.TODO(), &op))

		assert.Equal(t, "update readme", op.Setup.Checkout.Result.CommitMessage)
		assert.Equal(t, "test", op.Setup.Checkout.Result.Author)
		assert.Equal(t, "test@test.local", op.Setup.Checkout.Result.AuthorEmail)

		assert.Regexp(t, `^1\.0\.0\+2\.g[0-9a-f]+$`, op.Setup.Checkout.Result.Semver.Current)
		assert.Equal(t, "1.1.0", op.Setup.Checkout.Result.Semver.Next)

		assert.False(t, op.Setup.Checkout.Result.CommitVerified)
		assert.Equal(t, "commit not signed", op.Setup.Checkout.Result.Msg)

		require.Len(t, op.Setup.Checkout.Result.Files, 4)
		assert.Equal(t, "M", op.Setup.Checkout.Result.Files["README.md"].Status)
		assert.Equal(t, "D", op.Setup.Checkout.Result.Files["main.go"].Status)
		assert.Equal(t, "A", op.Setup.Checkout.Result.Files["renamed.go"].Status)
		assert.Equal(t, "A", op.Setup.Checkout.Result.Files["handler.go"].Status)
	})

	t.Run("pull request changeset", func(t *testing.T) {
		op := newOp()
		op.Setup.Checkout.GetChangeSet = true
		// Hooks send the target branch as a full ref
		op.Setup.Checkout.ChangeSetBranchTo = "refs/heads/master"

		require.NoError(t, s.processAnalyzeBare(context.TODO(), &op))
		require.Len(t, op.Setup.Checkout.Result.Files, 4, "diff against the branch head, not the homonymous tag")
		assert.Equal(t, "M", op.Setup.Checkout.Result.Files["README.md"].Status)
		assert.Equal(t, "D", op.Setup.Checkout.Result.Files["main.go"].Status, "main.go only exists on the branch head")
	})

	t.Run("changeset falls back to last commit files on unknown since commit", func(t *testing.T) {
		op := newOp()
		op.Setup.Checkout.GetChangeSet = true
		op.Setup.Checkout.ChangeSetCommitSince = strings.Repeat("0", 40)

		require.NoError(t, s.processAnalyzeBare(context.TODO(), &op))
		require.Len(t, op.Setup.Checkout.Result.Files, 1)
		assert.Equal(t, "M", op.Setup.Checkout.Result.Files["README.md"].Status)
	})

	t.Run("semver and signature on tag", func(t *testing.T) {
		op := sdk.Operation{URL: "file://" + fixture, RepoFullName: "test/fixture"}
		op.Setup.Checkout.Tag = "v1.0.0"
		op.Setup.Checkout.ProcessSemver = true
		op.Setup.Checkout.CheckSignature = true

		require.NoError(t, s.processAnalyzeBare(context.TODO(), &op))
		assert.Equal(t, "1.0.0", op.Setup.Checkout.Result.Semver.Current)
		assert.False(t, op.Setup.Checkout.Result.CommitVerified)
		assert.Equal(t, "commit not signed", op.Setup.Checkout.Result.Msg)
	})

	t.Run("message and semver on an annotated tag", func(t *testing.T) {
		execGitIn(t, fixture, "tag", "-a", "v1.1.0", "-m", "release 1.1.0", "feat")
		op := sdk.Operation{URL: "file://" + fixture, RepoFullName: "test/fixture"}
		op.Setup.Checkout.Tag = "v1.1.0"
		op.Setup.Checkout.GetMessage = true
		op.Setup.Checkout.ProcessSemver = true

		require.NoError(t, s.processAnalyzeBare(context.TODO(), &op))
		assert.Equal(t, "update readme", op.Setup.Checkout.Result.CommitMessage, "the tag must be peeled to its commit")
		assert.Equal(t, "1.1.0", op.Setup.Checkout.Result.Semver.Current)
	})

	t.Run("unknown tag is an error", func(t *testing.T) {
		op := sdk.Operation{URL: "file://" + fixture, RepoFullName: "test/fixture"}
		op.Setup.Checkout.Tag = "v9.9.9"
		op.Setup.Checkout.ProcessSemver = true

		err := s.processAnalyzeBare(context.TODO(), &op)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tag v9.9.9 not found")
	})

	t.Run("changeset between branches on a tag analysis", func(t *testing.T) {
		s := &Service{}
		s.Cfg.Basedir = t.TempDir() // fresh cache: no branch fetched yet
		op := sdk.Operation{URL: "file://" + fixture, RepoFullName: "test/fixture"}
		op.Setup.Checkout.Tag = "v1.1.0"
		op.Setup.Checkout.Branch = "feat"
		op.Setup.Checkout.GetChangeSet = true
		op.Setup.Checkout.ChangeSetBranchTo = "master"

		require.NoError(t, s.processAnalyzeBare(context.TODO(), &op))
		require.Len(t, op.Setup.Checkout.Result.Files, 4, "both branches must be fetched for the diff")
		assert.Equal(t, "D", op.Setup.Checkout.Result.Files["main.go"].Status)
	})

	t.Run("no blob was ever fetched in the bare cache", func(t *testing.T) {
		clonePath := s.BareRepo(newOp()).Basedir
		missing := execGitIn(t, clonePath, "rev-list", "--objects", "--missing=print", "refs/heads/feat")
		assert.NotZero(t, strings.Count(missing, "?"), "expected filtered blobs to still be missing")
	})
}

func Test_doWithBareAnalysisCache(t *testing.T) {
	fixture := createFixtureRepo(t)
	execGitIn(t, fixture, "tag", "v1.0.0")
	sinceSha := execGitIn(t, fixture, "rev-parse", "master")
	require.NoError(t, os.WriteFile(filepath.Join(fixture, "handler.go"), []byte("package handler"), os.FileMode(0644)))
	execGitIn(t, fixture, "add", ".")
	execGitIn(t, fixture, "commit", "-q", "-m", "add handler")
	headSha := execGitIn(t, fixture, "rev-parse", "master")

	s, err := newTestService(t)
	require.NoError(t, err)
	s.Cfg.Basedir = t.TempDir()
	s.Cfg.BareAnalysisCache = true
	s.Cfg.RepositoriesRetention = "6h"
	s.localCache = gocache.New(10*time.Minute, 10*time.Minute)
	ctx := context.TODO()

	newOp := func() sdk.Operation {
		op := sdk.Operation{UUID: sdk.UUID(), URL: "file://" + fixture, RepoFullName: "test/fixture"}
		op.Setup.Checkout.Branch = "master"
		op.Setup.Checkout.Commit = headSha
		op.Setup.Checkout.GetMessage = true
		op.Setup.Checkout.ProcessSemver = true
		op.Setup.Checkout.CheckSignature = true
		op.Setup.Checkout.GetChangeSet = true
		op.Setup.Checkout.ChangeSetCommitSince = sinceSha
		return op
	}
	repoID := s.Repo(newOp()).ID()
	t.Cleanup(func() {
		_ = s.Cache.Delete(s.dao.lastAccessKey(cacheRootFull.cloneKey(repoID)))
		_ = s.Cache.Delete(s.dao.lastAccessKey(cacheRootBare.cloneKey(repoID)))
	})

	t.Run("analysis operation is routed to the bare cache", func(t *testing.T) {
		op := newOp()
		require.NoError(t, s.do(ctx, op))

		saved := s.dao.loadOperation(ctx, op.UUID)
		require.NotNil(t, saved)
		t.Cleanup(func() { _ = s.dao.deleteOperation(saved) })
		assert.Equal(t, sdk.OperationStatusDone, saved.Status)
		assert.Nil(t, saved.Error)
		assert.Equal(t, "add handler", saved.Setup.Checkout.Result.CommitMessage)
		assert.Regexp(t, `^1\.0\.0\+1\.g[0-9a-f]+$`, saved.Setup.Checkout.Result.Semver.Current)
		assert.Equal(t, "commit not signed", saved.Setup.Checkout.Result.Msg)
		require.Len(t, saved.Setup.Checkout.Result.Files, 1)
		assert.Equal(t, "A", saved.Setup.Checkout.Result.Files["handler.go"].Status)

		assert.DirExists(t, filepath.Join(s.rootDir(cacheRootBare), repoID), "the clone must live in the bare namespace")
		assert.NoDirExists(t, filepath.Join(s.rootDir(cacheRootFull), repoID), "the full clones namespace must stay untouched")

		var protectedUntil time.Time
		found, err := s.Cache.Get(s.dao.lastAccessKey(cacheRootBare.cloneKey(repoID)), &protectedUntil)
		require.NoError(t, err)
		assert.True(t, found, "the bare scoped lastAccess key must be written")
		found, err = s.Cache.Get(s.dao.lastAccessKey(cacheRootFull.cloneKey(repoID)), &protectedUntil)
		require.NoError(t, err)
		assert.False(t, found, "the full clone lastAccess key must not be written")
	})

	t.Run("failed analysis ends in error status", func(t *testing.T) {
		op := newOp()
		op.Setup.Checkout.Branch = "doesnotexist"
		require.NoError(t, s.do(ctx, op))

		saved := s.dao.loadOperation(ctx, op.UUID)
		require.NotNil(t, saved)
		t.Cleanup(func() { _ = s.dao.deleteOperation(saved) })
		assert.Equal(t, sdk.OperationStatusError, saved.Status)
		require.NotNil(t, saved.Error)
	})

	t.Run("disabled flag keeps the current path", func(t *testing.T) {
		s.Cfg.BareAnalysisCache = false
		defer func() { s.Cfg.BareAnalysisCache = true }()

		op := newOp()
		require.NoError(t, s.do(ctx, op))

		saved := s.dao.loadOperation(ctx, op.UUID)
		require.NotNil(t, saved)
		t.Cleanup(func() { _ = s.dao.deleteOperation(saved) })
		assert.Equal(t, sdk.OperationStatusDone, saved.Status)
		assert.DirExists(t, filepath.Join(s.rootDir(cacheRootFull), repoID), "the operation must run on the full clones cache")

		var protectedUntil time.Time
		found, err := s.Cache.Get(s.dao.lastAccessKey(cacheRootFull.cloneKey(repoID)), &protectedUntil)
		require.NoError(t, err)
		assert.True(t, found, "the full clone lastAccess key must be written")
	})
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
	assert.Equal(t, execGitIn(t, fixture, "rev-parse", "v1.0.0^{commit}"), target, "the tag is resolved to its commit")
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
