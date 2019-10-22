package cdn

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/ovh/cds/engine/cdn/objectstore"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) mirroring(object objectstore.Object, body io.Closer, reader io.Reader) {
	defer body.Close()

	for _, mirror := range s.MirrorDrivers {
		var buf bytes.Buffer
		tee := io.TeeReader(reader, &buf)
		_, err := mirror.Store(object, ioutil.NopCloser(tee))
		if err != nil {
			log.Error("Cannot mirror artifact : %v", err)
		}
		reader = &buf
	}
}
