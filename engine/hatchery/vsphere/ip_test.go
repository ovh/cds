package vsphere

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/vim25/mo"
	"go.uber.org/mock/gomock"
)

// findAvailableIP must reserve atomically: concurrent callers (parallel
// provisioning clones) must each get a distinct IP, never the same one.
func TestHatcheryVSphere_findAvailableIP_concurrentDistinct(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{vSphereClient: c}

	ips := []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"}
	h.availableNetworks = []availableNetwork{{
		config:      NetworkConfig{Gateway: "10.0.0.254", SubnetMask: "255.255.255.0"},
		ipAddresses: ips,
	}}
	h.availableIPAddresses = ips

	// No VM uses any IP yet.
	c.EXPECT().ListVirtualMachines(gomock.Any()).Return([]mo.VirtualMachine{}, nil).AnyTimes()

	const goroutines = 8
	var mu sync.Mutex
	handed := map[string]int{}
	var errCount int64
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := h.findAvailableIP(context.Background())
			if err != nil {
				atomic.AddInt64(&errCount, 1)
				return
			}
			mu.Lock()
			handed[res.ip]++
			mu.Unlock()
		}()
	}
	wg.Wait()

	// Exactly the 3 IPs are handed out, each exactly once; the rest fail.
	assert.Len(t, handed, 3, "all 3 IPs should have been allocated")
	for ip, n := range handed {
		assert.Equalf(t, 1, n, "IP %s was handed to more than one caller", ip)
	}
	assert.Equal(t, int64(goroutines-len(ips)), errCount, "callers beyond the IP budget must get an error, not a duplicate")
}
