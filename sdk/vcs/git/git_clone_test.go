package git

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractInfo(t *testing.T) {
	repo := "https://github.com/ovh/cds.git"
	myrepo, _ := ioutil.TempDir(os.TempDir(), "cds_TestExtractInfo")
	defer os.RemoveAll(myrepo)
	opts := &CloneOpts{
		CheckoutCommit: "f57e4c8405d5b6ffddc33755c105f73c64ed89da",
		Verbose:        true,
		// Depth can be set to 100 locally, but be sure that the sha1 is well downloaded
		Depth: 10000,
		Quiet: false,
	}

	out := new(bytes.Buffer)
	errb := new(bytes.Buffer)

	output := &OutputOpts{
		Stdout: out,
		Stderr: errb,
	}

	LogFunc = t.Logf
	verbose = true

	_, err := Clone(repo, myrepo, ".", nil, opts, output)
	assert.NoError(t, err)

	info, err := ExtractInfo(context.TODO(), myrepo, &CloneOpts{ForceGetGitDescribe: true})
	require.NoError(t, err)
	assert.NotEmpty(t, info.GitDescribe)
	assert.Equal(t, "0.41.0-119-gf57e4c840", info.GitDescribe)
}
func TestExtractInfoAbsPath(t *testing.T) {
	repo := "https://github.com/ovh/cds.git"
	myrepo, _ := ioutil.TempDir(os.TempDir(), "cds_TestExtractInfoAbsPath")
	defer os.RemoveAll(myrepo)
	opts := &CloneOpts{
		CheckoutCommit: "f57e4c8405d5b6ffddc33755c105f73c64ed89da",
		Verbose:        true,
		// Depth can be set to 100 locally, but be sure that the sha1 is well downloaded
		Depth: 10000,
		Quiet: false,
	}

	out := new(bytes.Buffer)
	errb := new(bytes.Buffer)

	output := &OutputOpts{
		Stdout: out,
		Stderr: errb,
	}

	LogFunc = t.Logf
	verbose = true

	_, err := Clone(repo, myrepo, myrepo, nil, opts, output) // equals to {{.cds.workspace}} in path attribute
	assert.NoError(t, err)

	info, err := ExtractInfo(context.TODO(), myrepo, &CloneOpts{ForceGetGitDescribe: true})
	require.NoError(t, err)
	assert.NotEmpty(t, info.GitDescribe)
	assert.Equal(t, "0.41.0-119-gf57e4c840", info.GitDescribe)
}
func TestExtractInfoEmptyPath(t *testing.T) {
	repo := "https://github.com/ovh/cds.git"
	myrepo, _ := ioutil.TempDir(os.TempDir(), "cds_TestExtractInfoEmptyPath")
	defer os.RemoveAll(myrepo)
	opts := &CloneOpts{
		CheckoutCommit: "f57e4c8405d5b6ffddc33755c105f73c64ed89da",
		Verbose:        true,
		// Depth can be set to 100 locally, but be sure that the sha1 is well downloaded
		Depth: 10000,
		Quiet: false,
	}

	out := new(bytes.Buffer)
	errb := new(bytes.Buffer)

	output := &OutputOpts{
		Stdout: out,
		Stderr: errb,
	}

	LogFunc = t.Logf
	verbose = true

	_, err := Clone(repo, myrepo, "", nil, opts, output) // equals to "" in path attribute, this will create a cds directory
	assert.NoError(t, err)

	info, err := ExtractInfo(context.TODO(), filepath.Join(myrepo, "cds"), &CloneOpts{ForceGetGitDescribe: true})
	require.NoError(t, err)
	assert.NotEmpty(t, info.GitDescribe)
	assert.Equal(t, "0.41.0-119-gf57e4c840", info.GitDescribe)
}
