package git

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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

	_, err := Clone(repo, myrepo, nil, opts, output)
	assert.NoError(t, err)

	info := ExtractInfo(myrepo, &CloneOpts{ForceGetGitDescribe: true})
	assert.NotEmpty(t, info.GitDescribe)
	assert.Equal(t, "0.41.0-119-gf57e4c840", info.GitDescribe)
}
