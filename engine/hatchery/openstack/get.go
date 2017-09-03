package openstack

import (
	"fmt"
	"sync"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"

	"github.com/ovh/cds/sdk/log"
)

// Find image ID from image name
func (h *HatcheryOpenstack) imageID(img string) (string, error) {
	for _, i := range h.getImages() {
		if i.Name == img {
			return i.ID, nil
		}
	}
	return "", fmt.Errorf("imageID> image '%s' not found", img)
}

// Find flavor ID from flavor name
func (h *HatcheryOpenstack) flavorID(flavor string) (string, error) {
	for _, f := range h.flavors {
		if f.Name == flavor {
			return f.ID, nil
		}
	}
	return "", fmt.Errorf("flavorID> flavor '%s' not found", flavor)
}

//This a embeded cache for images list
var limages = struct {
	mu   sync.RWMutex
	list []images.Image
}{
	mu:   sync.RWMutex{},
	list: []images.Image{},
}

func (h *HatcheryOpenstack) getImages() []images.Image {
	t := time.Now()
	defer log.Debug("getImages(): %fs", time.Since(t).Seconds())

	limages.mu.RLock()
	nbImages := len(limages.list)
	limages.mu.RUnlock()

	if nbImages == 0 {
		all, err := images.ListDetail(h.openstackClient, nil).AllPages()
		if err != nil {
			log.Error("getImages> error on listDetail: %s", err)
			return limages.list
		}
		imgs, err := images.ExtractImages(all)
		if err != nil {
			log.Error("getImages> error on images.ExtractImages: %s", err)
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

//This a embeded cache for servers list
var lservers = struct {
	mu   sync.RWMutex
	list []servers.Server
}{
	mu:   sync.RWMutex{},
	list: []servers.Server{},
}

func (h *HatcheryOpenstack) getServers() []servers.Server {
	t := time.Now()
	defer log.Debug("getServers() : %fs", time.Since(t).Seconds())

	lservers.mu.RLock()
	nbServers := len(lservers.list)
	lservers.mu.RUnlock()

	if nbServers == 0 {
		all, err := servers.List(h.openstackClient, nil).AllPages()
		if err != nil {
			log.Error("getServers> error on servers.List: %s", err)
			return lservers.list
		}
		serverList, err := servers.ExtractServers(all)
		if err != nil {
			log.Error("getServers> error on servers.ExtractServers: %s", err)
			return lservers.list
		}

		srvs := []servers.Server{}
		for _, s := range serverList {
			_, worker := s.Metadata["worker"]
			if !worker {
				continue
			}
			workerHatcheryName, _ := s.Metadata["hatchery_name"]
			if workerHatcheryName == "" || workerHatcheryName != h.Hatchery().Name {
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
