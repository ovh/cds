package swarm

import (
	"io/ioutil"
	"time"

	types "github.com/docker/docker/api/types"
	context "golang.org/x/net/context"

	"github.com/ovh/cds/sdk/log"
)

func (h *HatcherySwarm) pullImage(img string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	//Pull the worker image
	opts := types.ImagePullOptions{}
	log.Info("pullImage> pulling image %s", img)
	res, err := h.dockerClient.ImagePull(ctx, img, opts)
	if err != nil {
		log.Warning("pullImage> Unable to pull image %s : %s", img, err)
		return err
	}

	btes, _ := ioutil.ReadAll(res)
	log.Debug("pullImage> %s", string(btes))
	return res.Close()
}
