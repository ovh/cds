package openstack_test

import (
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/hatchery/openstack"
)

func TestCache_images(t *testing.T) {
	cache := openstack.NewCache(1, 1)
	is, expired := cache.Getimages()
	require.Len(t, is, 0)
	require.True(t, expired)
	cache.SetImages([]images.Image{{}, {}})
	is, expired = cache.Getimages()
	require.Len(t, is, 2)
	require.False(t, expired)
	time.Sleep(1 * time.Second)
	is, expired = cache.Getimages()
	require.Len(t, is, 2)
	require.True(t, expired)
}

func TestCache_servers(t *testing.T) {
	cache := openstack.NewCache(1, 1)
	srvs, expired := cache.GetServers()
	require.Len(t, srvs, 0)
	require.True(t, expired)
	cache.SetServers([]servers.Server{{}, {}})
	srvs, expired = cache.GetServers()
	require.Len(t, srvs, 2)
	require.False(t, expired)
	time.Sleep(1 * time.Second)
	srvs, expired = cache.GetServers()
	require.Len(t, srvs, 2)
	require.True(t, expired)
}
