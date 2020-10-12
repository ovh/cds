package openstack

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Find image ID from image name
func (h *HatcheryOpenstack) imageID(ctx context.Context, img string) (string, error) {
	for _, i := range h.getImages(ctx) {
		if i.Name == img {
			return i.ID, nil
		}
	}
	return "", sdk.WithStack(fmt.Errorf("image '%s' not found", img))
}

// Find flavor ID from flavor name
func (h *HatcheryOpenstack) flavor(flavor string) (flavors.Flavor, error) {
	for i := range h.flavors {
		if h.flavors[i].Name == flavor {
			return h.flavors[i], nil
		}
	}
	return flavors.Flavor{}, sdk.WithStack(fmt.Errorf("flavor '%s' not found", flavor))
}

// Find flavor ID from flavor name
func (h *HatcheryOpenstack) getSmallerFlavorThan(flavor flavors.Flavor) flavors.Flavor {
	var smaller *flavors.Flavor
	for i := range h.flavors {
		// If the flavor is not the given one and need less CPUs its
		if h.flavors[i].ID != flavor.ID && h.flavors[i].VCPUs < flavor.VCPUs && (smaller == nil || smaller.VCPUs < h.flavors[i].VCPUs) {
			smaller = &h.flavors[i]
		}
	}
	if smaller == nil {
		return flavor
	}
	return *smaller
}

//This a embedded cache for images list
var limages = struct {
	mu   sync.RWMutex
	list []images.Image
}{
	mu:   sync.RWMutex{},
	list: []images.Image{},
}

func (h *HatcheryOpenstack) getImages(ctx context.Context) []images.Image {
	t := time.Now()
	defer log.Debug("getImages(): %fs", time.Since(t).Seconds())

	limages.mu.RLock()
	nbImages := len(limages.list)
	limages.mu.RUnlock()

	if nbImages == 0 {
		all, err := images.ListDetail(h.openstackClient, nil).AllPages()
		if err != nil {
			log.Error(ctx, "getImages> error on listDetail: %s", err)
			return limages.list
		}
		imgs, err := images.ExtractImages(all)
		if err != nil {
			log.Error(ctx, "getImages> error on images.ExtractImages: %s", err)
			return limages.list
		}

		activeImages := []images.Image{}
		for i := range imgs {
			log.Debug("getImages> image %s status %s progress %d all:%+v", imgs[i].Name, imgs[i].Status, imgs[i].Progress, imgs[i])
			if imgs[i].Status == "ACTIVE" {
				log.Debug("getImages> add %s to activeImages", imgs[i].Name)
				activeImages = append(activeImages, imgs[i])
			}
		}

		limages.mu.Lock()
		limages.list = activeImages
		limages.mu.Unlock()
		//Remove data from the cache after 2 seconds
		go func() {
			time.Sleep(10 * time.Minute)
			h.resetImagesCache()
		}()
	}

	return limages.list
}

func (h *HatcheryOpenstack) resetImagesCache() {
	limages.mu.Lock()
	limages.list = []images.Image{}
	limages.mu.Unlock()
}

//This a embedded cache for servers list
var lservers = struct {
	mu   sync.RWMutex
	list []servers.Server
}{
	mu:   sync.RWMutex{},
	list: []servers.Server{},
}

func (h *HatcheryOpenstack) getServers(ctx context.Context) []servers.Server {
	t := time.Now()
	defer log.Debug("getServers() : %fs", time.Since(t).Seconds())

	lservers.mu.RLock()
	nbServers := len(lservers.list)
	lservers.mu.RUnlock()

	if nbServers == 0 {
		var serverList []servers.Server
		var isOk bool
		for i := 0; i <= 5; i++ {
			all, err := servers.List(h.openstackClient, nil).AllPages()
			if err != nil {
				log.Error(ctx, "getServers> error on servers.List: %s", err)
				continue
			}
			serverList, err = servers.ExtractServers(all)
			if err != nil {
				log.Error(ctx, "getServers> error on servers.ExtractServers: %s", err)
				continue
			}
			isOk = true
			break
		}
		if !isOk {
			return lservers.list
		}

		srvs := []servers.Server{}
		for _, s := range serverList {
			_, worker := s.Metadata["worker"]
			if !worker {
				continue
			}
			workerHatcheryName := s.Metadata["hatchery_name"]
			if workerHatcheryName == "" || workerHatcheryName != h.Name() {
				continue
			}
			srvs = append(srvs, s)
		}

		lservers.mu.Lock()
		lservers.list = srvs
		lservers.mu.Unlock()
		//Remove data from the cache after 2 seconds
		go func() {
			time.Sleep(2 * time.Second)
			lservers.mu.Lock()
			lservers.list = []servers.Server{}
			lservers.mu.Unlock()
		}()
	}

	return lservers.list
}

func (h *HatcheryOpenstack) getConsoleLog(s servers.Server) (string, error) {
	result := servers.ShowConsoleOutput(h.openstackClient, s.ID, servers.ShowConsoleOutputOpts{})
	info, err := result.Extract()
	return info, sdk.WrapError(err, "unable to get console log from %s", s.ID)
}
