package openstack

import (
	"context"
	"fmt"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
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

func (h *HatcheryOpenstack) getImages(ctx context.Context) []images.Image {
	t := time.Now()
	defer log.Debug(ctx, "getImages(): %fs", time.Since(t).Seconds())

	is, expired := h.cache.Getimages()
	if len(is) > 0 && !expired {
		return is
	}

	if err := h.refreshImagesCache(ctx); err != nil {
		log.ErrorWithStackTrace(ctx, err)
	}

	is, _ = h.cache.Getimages()
	return is
}

func (h *HatcheryOpenstack) refreshImagesCache(ctx context.Context) error {
	all, err := images.ListDetail(h.openstackClient, nil).AllPages()
	if err != nil {
		return sdk.WrapError(err, "cannot list openstack images")
	}
	imgs, err := images.ExtractImages(all)
	if err != nil {
		return sdk.WrapError(err, "cannot extract openstack images")
	}

	activeImages := make([]images.Image, 0, len(imgs))
	for i := range imgs {
		if imgs[i].Status == "ACTIVE" {
			activeImages = append(activeImages, imgs[i])
		}
	}

	h.cache.SetImages(activeImages)
	return nil
}

func (h *HatcheryOpenstack) getServers(ctx context.Context) []servers.Server {
	t := time.Now()
	defer log.Debug(ctx, "getServers() : %fs", time.Since(t).Seconds())

	srvs, expired := h.cache.GetServers()
	if len(srvs) > 0 && !expired {
		return srvs
	}

	if err := h.refreshServersCache(ctx); err != nil {
		log.ErrorWithStackTrace(ctx, err)
	}

	srvs, _ = h.cache.GetServers()
	return srvs
}

func (h *HatcheryOpenstack) refreshServersCache(ctx context.Context) error {
	all, err := servers.List(h.openstackClient, nil).AllPages()
	if err != nil {
		return sdk.WrapError(err, "cannot list openstack servers")
	}
	serverList, err := servers.ExtractServers(all)
	if err != nil {
		return sdk.WrapError(err, "cannot extract openstack servers")
	}

	filteredServerList := make([]servers.Server, 0, len(serverList))
	for _, s := range serverList {
		if _, worker := s.Metadata["worker"]; !worker {
			continue
		}
		workerHatcheryName := s.Metadata["hatchery_name"]
		if workerHatcheryName == "" || workerHatcheryName != h.Name() {
			continue
		}
		filteredServerList = append(filteredServerList, s)
	}

	h.cache.SetServers(filteredServerList)
	return nil
}

func (h *HatcheryOpenstack) getConsoleLog(s servers.Server) (string, error) {
	result := servers.ShowConsoleOutput(h.openstackClient, s.ID, servers.ShowConsoleOutputOpts{})
	info, err := result.Extract()
	return info, sdk.WrapError(err, "unable to get console log from %s", s.ID)
}
