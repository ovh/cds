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

	bts, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Error("cannot read body to mirror: %v", err)
		return
	}

	for _, mirror := range s.MirrorDrivers {
		// TODO: check to duplicate stream
		_, err := mirror.Store(object, ioutil.NopCloser(bytes.NewBuffer(bts)))
		if err != nil {
			log.Error("Cannot mirror artifact : %v", err)
		}
	}
}
