package cdn

import (
	"context"
	"fmt"
	"io"

	"github.com/ovh/cds/engine/cdn/objectstore"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) mirroring(object objectstore.Object, reader io.Reader) {
	fmt.Println("Mirroring !", len(s.MirrorDrivers))
	writerClosers := make([]io.WriteCloser, 0, len(s.MirrorDrivers))
	for _, mirror := range s.MirrorDrivers {
		fileWriter, err := mirror.Open(context.Background(), object)
		if err != nil {
			log.Error("Cannot mirror artifact : %v", err)
			continue
		}
		writerClosers = append(writerClosers, fileWriter)
	}

	multiWriters := MultiWriteCloser(writerClosers...)

	_, err := io.Copy(multiWriters, reader)
	if err != nil {
		log.Error("cannot write to writers : %v", err)
		return
	}

	if err := multiWriters.Close(); err != nil {
		log.Error("cannot close multiWriteClosers : %v", err)
	}
}

func (s *Service) downloadFromMirrors(ctx context.Context, object objectstore.Object) (io.ReadCloser, error) {
	var content io.ReadCloser
	var err error
	for _, mirror := range s.MirrorDrivers {
		content, err = mirror.Fetch(ctx, object)
		if err == nil {
			return content, nil
		}
	}

	return nil, sdk.WrapError(err, "cannot download from mirrors")
}
