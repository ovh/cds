package swarm

import (
	"bytes"
	"io"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"

	types "github.com/docker/docker/api/types"
	context "golang.org/x/net/context"
)

func (h *HatcherySwarm) pullImage(dockerClient *dockerClient, img string, timeout time.Duration) error {
	t0 := time.Now()
	log.Debug("hatchery> swarm> pullImage> pulling image %s on %s", img, dockerClient.name)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	//Pull the worker image
	opts := types.ImageCreateOptions{}
	res, err := dockerClient.ImageCreate(ctx, img, opts)
	if err != nil {
		log.Warning("hatchery> swarm> pullImage> Unable to pull image %s on %s: %s", img, dockerClient.name, err)
		return err
	}

	p := make([]byte, 4)
	buff := new(bytes.Buffer)
	for {
		if _, err := res.Read(p); err == io.EOF {
			break
		} else if err != nil {
			break
		}
		_, _ = buff.Write(p)
	}
	if err := res.Close(); err != nil {
		return sdk.WrapError(err, "hatchery> swarm> pullImage> error closing the buffer")
	}

	log.Debug(buff.String())
	log.Info("hatchery> swarm> pullImage> pulling image %s on %s - %.3f seconds elapsed", img, dockerClient.name, time.Since(t0).Seconds())

	return nil
}
