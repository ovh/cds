package cdn

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"

	"github.com/ovh/cds/engine/cdn/objectstore"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) mirroring(object objectstore.Object, body io.Closer, reader io.Reader) {
	defer body.Close()

	for _, mirror := range s.MirrorDrivers {
		var buf bytes.Buffer
		tee := io.TeeReader(reader, &buf)
		_, err := mirror.Store(object, ioutil.NopCloser(tee))
		if err != nil {
			log.Error(context.Background(), "Cannot mirror artifact : %v", err)
		}
		reader = &buf
	}
}

func (s *Service) downloadFromMirrors(object objectstore.Object) (io.ReadCloser, error) {
	var content io.ReadCloser
	var err error
	for _, mirror := range s.MirrorDrivers {
		content, err = mirror.Fetch(object)
		if err == nil {
			return content, nil
		}
	}

	return nil, sdk.WrapError(err, "cannot download from mirrors")
}
