package artifact

import (
	"io"

	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// SaveWorkflowFile Insert file in db and write it in data directory
func SaveWorkflowFile(art *sdk.WorkflowNodeRunArtifact, content io.ReadCloser) error {
	objectPath, err := objectstore.Store(art, content)
	if err != nil {
		return sdk.WrapError(err, "Cannot store artifact")
	}
	log.Debug("objectpath=%s\n", objectPath)
	art.ObjectPath = objectPath
	return nil
}
