package swarm

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"

	types "github.com/docker/docker/api/types"
	context "golang.org/x/net/context"
)

func (h *HatcherySwarm) pullImage(dockerClient *dockerClient, img string, timeout time.Duration, model sdk.Model) error {
	t0 := time.Now()
	log.Debug("hatchery> swarm> pullImage> pulling image %s on %s", img, dockerClient.name)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	//Pull the worker image
	opts := types.ImageCreateOptions{}
	if model.ModelDocker.Private {
		registry := "index.docker.io"
		if model.ModelDocker.Registry != "" {
			urlParsed, errParsed := url.Parse(model.ModelDocker.Registry)
			if errParsed != nil {
				return sdk.WrapError(errParsed, "cannot parse registry url %s", registry)
			}
			if urlParsed.Host == "" {
				registry = urlParsed.Path
			} else {
				registry = urlParsed.Host
			}
		}
		auth := fmt.Sprintf(`{"username": "%s", "password": "%s", "serveraddress": "%s"}`, model.ModelDocker.Username, model.ModelDocker.Password, registry)
		opts.RegistryAuth = base64.StdEncoding.EncodeToString([]byte(auth))
	}
	res, err := dockerClient.ImageCreate(ctx, img, opts)
	if err != nil {
		log.Warning("hatchery> swarm> pullImage> Unable to pull image %s on %s: %s", img, dockerClient.name, err)
		return sdk.WithStack(err)
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
		return sdk.WrapError(err, "error closing the buffer")
	}

	log.Debug(buff.String())
	log.Info("hatchery> swarm> pullImage> pulling image %s on %s - %.3f seconds elapsed", img, dockerClient.name, time.Since(t0).Seconds())

	return nil
}
