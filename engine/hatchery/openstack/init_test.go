package openstack

import (
	"testing"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHatcheryOpenstack_initFlavors(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	h := &HatcheryOpenstack{}
	h.cache = NewCache(1, 1)

	allFlavors := []flavors.Flavor{
		{Name: "b2-7", VCPUs: 2},
		{Name: "b2-15", VCPUs: 4},
		{Name: "b2-30", VCPUs: 8},
		{Name: "b2-60", VCPUs: 16},
		{Name: "b2-120", VCPUs: 32},
	}

	filteredFlavors := h.filterAllowedFlavors(allFlavors)
	require.Len(t, filteredFlavors, 5, "no filter as allowed flavor list is empty in config")

	h.Config.AllowedFlavors = []string{"b2-15", "b2-60"}

	filteredFlavors = h.filterAllowedFlavors(allFlavors)
	require.Len(t, filteredFlavors, 2)
	assert.Equal(t, "b2-15", filteredFlavors[0].Name)
	assert.Equal(t, "b2-60", filteredFlavors[1].Name)

	h.Config.AllowedFlavors = []string{"s1-4", "b2-15"}

	filteredFlavors = h.filterAllowedFlavors(allFlavors)
	require.Len(t, filteredFlavors, 1)
	assert.Equal(t, "b2-15", filteredFlavors[0].Name)
}
