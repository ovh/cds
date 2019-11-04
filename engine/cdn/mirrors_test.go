package cdn

import (
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/ovh/cds/engine/cdn/objectstore"
	"github.com/ovh/cds/sdk"

	"github.com/stretchr/testify/assert"
)

func Test_mirroring(t *testing.T) {
	reader := strings.NewReader("test")
	srv, err := newTestService(t)
	assert.NoError(t, err)
	art := sdk.WorkflowNodeRunArtifact{
		Name: sdk.RandomString(10),
		Tag:  "test",
		Ref:  "test",
	}
	firstMirror, _ := objectstore.NewFilesystemStore(sdk.ProjectIntegration{Name: sdk.DefaultStorageIntegrationName}, objectstore.ConfigOptionsFilesystem{BaseDirectory: defaultBaseDir})
	secondMirror, _ := objectstore.NewFilesystemStore(sdk.ProjectIntegration{Name: sdk.DefaultStorageIntegrationName}, objectstore.ConfigOptionsFilesystem{BaseDirectory: "/tmp/cdstest"})
	defer os.RemoveAll(defaultBaseDir)
	defer os.RemoveAll("/tmp/cdstest")
	srv.MirrorDrivers = append(srv.MirrorDrivers, firstMirror, secondMirror)

	srv.mirroring(&art, reader)

	btes, err := ioutil.ReadFile(path.Join(defaultBaseDir, art.GetPath(), art.GetName()))
	assert.NoError(t, err, "cannot read file")
	assert.Equal(t, "test", string(btes))
	btes, err = ioutil.ReadFile(path.Join("/tmp/cdstest", art.GetPath(), art.GetName()))
	assert.NoError(t, err, "cannot read file")
	assert.Equal(t, "test", string(btes))
}
