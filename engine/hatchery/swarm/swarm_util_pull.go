package swarm

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/url"
	"regexp"
	"time"

	"github.com/docker/distribution/reference"
	types "github.com/docker/docker/api/types"
	"github.com/rockbears/log"
	context "golang.org/x/net/context"

	"github.com/ovh/cds/sdk"
)

func (h *HatcherySwarm) pullImage(dockerClient *dockerClient, img string, timeout time.Duration, model sdk.WorkerStarterWorkerModel) error {
	t0 := time.Now()
	log.Debug(context.TODO(), "hatchery> swarm> pullImage> pulling image %s on %s", img, dockerClient.name)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	//Pull the worker image
	var authConfig *types.AuthConfig
	if model.IsPrivate() {
		registry := "index.docker.io"
		if model.ModelV1.ModelDocker.Registry != "" {
			urlParsed, err := url.Parse(model.ModelV1.ModelDocker.Registry)
			if err != nil {
				return sdk.WrapError(err, "cannot parse registry url %q", registry)
			}
			if urlParsed.Host == "" {
				registry = urlParsed.Path
			} else {
				registry = urlParsed.Host
			}
		}
		authConfig = &types.AuthConfig{
			Username:      model.GetDockerUsername(),
			Password:      model.GetDockerPassword(),
			ServerAddress: registry,
		}
	} else {
		ref, err := reference.ParseNormalizedNamed(img)
		if err != nil {
			return sdk.WithStack(err)
		}
		domain := reference.Domain(ref)
		var credentials *RegistryCredential
		if model.GetDockerUsername() == "" {
			// Check if credentials match current domain
			for i := range h.Config.RegistryCredentials {
				if h.Config.RegistryCredentials[i].Domain == domain {
					credentials = &h.Config.RegistryCredentials[i]
					break
				}
			}
			if credentials == nil {
				// Check if regex credentials match current domain
				for i := range h.Config.RegistryCredentials {
					reg := regexp.MustCompile(h.Config.RegistryCredentials[i].Domain)
					if reg.MatchString(domain) {
						credentials = &h.Config.RegistryCredentials[i]
						break
					}
				}
			}
		} else {
			credentials.Username = model.GetDockerUsername()
			credentials.Password = model.GetDockerPassword()
			credentials.Domain = domain
		}

		if credentials != nil {
			authConfig = &types.AuthConfig{
				Username:      credentials.Username,
				Password:      credentials.Password,
				ServerAddress: domain,
			}
			log.Debug(context.TODO(), "found credentials %q to pull image %q", credentials.Domain, img)
		}
	}

	opts := types.ImageCreateOptions{}
	if authConfig != nil {
		config, err := json.Marshal(authConfig)
		if err != nil {
			return sdk.WithStack(err)
		}
		opts.RegistryAuth = base64.StdEncoding.EncodeToString(config)
		log.Debug(context.TODO(), "pulling image %q on %q with login on %q", img, dockerClient.name, authConfig.ServerAddress)
	}

	res, err := dockerClient.ImageCreate(ctx, img, opts)
	if err != nil {
		ctx = sdk.ContextWithStacktrace(ctx, err)
		log.Warn(ctx, "unable to pull image %s on %s: %s", img, dockerClient.name, err)
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

	log.Debug(ctx, buff.String())
	log.Info(ctx, "hatchery> swarm> pullImage> pulling image %s on %s - %.3f seconds elapsed", img, dockerClient.name, time.Since(t0).Seconds())

	return nil
}
