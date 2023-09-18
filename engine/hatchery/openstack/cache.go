package openstack

import (
	"sync"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
)

func NewCache(imagesExpirationDelay, serversExpirationDelay int) *cache {
	if imagesExpirationDelay == 0 {
		imagesExpirationDelay = 30
	}
	if serversExpirationDelay == 0 {
		imagesExpirationDelay = 2
	}
	var c cache
	c.images.expirationDelay = time.Duration(imagesExpirationDelay) * time.Second
	c.servers.expirationDelay = time.Duration(serversExpirationDelay) * time.Second
	return &c
}

// cache struct will not be reset automatically after expiration delay to prevent problems if the data cannot be retrieved from Openstack API
type cache struct {
	images struct {
		mu              sync.RWMutex
		list            []images.Image
		refreshDate     time.Time
		expirationDelay time.Duration
	}
	servers struct {
		mu              sync.RWMutex
		list            []servers.Server
		refreshDate     time.Time
		expirationDelay time.Duration
	}
}

func (c *cache) Getimages() ([]images.Image, bool) {
	c.images.mu.RLock()
	defer c.images.mu.RUnlock()
	expired := c.images.refreshDate.Add(c.images.expirationDelay).Before(time.Now())
	tmp := make([]images.Image, len(c.images.list))
	copy(tmp, c.images.list)
	return tmp, expired
}

func (c *cache) SetImages(l []images.Image) {
	c.images.mu.Lock()
	defer c.images.mu.Unlock()
	tmp := make([]images.Image, len(l))
	copy(tmp, l)
	c.images.list = tmp
	c.images.refreshDate = time.Now()
}

func (c *cache) GetServers() ([]servers.Server, bool) {
	c.servers.mu.RLock()
	defer c.servers.mu.RUnlock()
	expired := c.servers.refreshDate.Add(c.servers.expirationDelay).Before(time.Now())
	tmp := make([]servers.Server, len(c.servers.list))
	copy(tmp, c.servers.list)
	return tmp, expired
}

func (c *cache) SetServers(l []servers.Server) {
	c.servers.mu.Lock()
	defer c.servers.mu.Unlock()
	tmp := make([]servers.Server, len(l))
	copy(tmp, l)
	c.servers.list = tmp
	c.servers.refreshDate = time.Now()
}
