package vsphere

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/ovh/cds/sdk/log"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
)

//This a embeded cache for servers list
var lservers = struct {
	mu   sync.RWMutex
	list []mo.VirtualMachine
}{
	mu:   sync.RWMutex{},
	list: []mo.VirtualMachine{},
}

func (h *HatcheryVSphere) getServers() []mo.VirtualMachine {
	var vms []mo.VirtualMachine
	ctx := context.Background()

	t := time.Now()
	defer log.Debug("getServers() : %fs", time.Since(t).Seconds())

	lservers.mu.RLock()
	nbServers := len(lservers.list)
	lservers.mu.RUnlock()

	if nbServers == 0 {
		m := view.NewManager(h.client.Client)

		v, err := m.CreateContainerView(ctx, h.client.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
		if err != nil {
			log.Error("Unable to create container view for vsphere api")
			os.Exit(12)
		}
		defer v.Destroy(ctx)

		// Retrieve summary property for all machines
		// Reference: http://pubs.vmware.com/vsphere-60/topic/com.vmware.wssdk.apiref.doc/vim.VirtualMachine.html
		if err := v.Retrieve(ctx, []string{"VirtualMachine"}, []string{"summary", "config"}, &vms); err != nil {
			log.Warning("Unable to retrieve virtual machines from vsphere %s", err)
			return lservers.list
		}

		lservers.mu.Lock()
		lservers.list = vms
		lservers.mu.Unlock()
		//Remove data from the cache after 2 seconds
		go func() {
			time.Sleep(2 * time.Second)
			lservers.mu.Lock()
			lservers.list = []mo.VirtualMachine{}
			lservers.mu.Unlock()
		}()
	}

	return lservers.list
}

func (h *HatcheryVSphere) deleteServer(s mo.VirtualMachine) error {
	ctx := context.TODO()
	vms, errVml := h.finder.VirtualMachineList(ctx, s.Name)
	if errVml != nil {
		return errVml
	}

	for _, vm := range vms {
		task, errOff := vm.PowerOff(ctx)
		if errOff != nil {
			return errOff
		}
		task.Wait(ctx)

		var errD error
		task, errD = vm.Destroy(ctx)
		if errD != nil {
			return errD
		}

		return task.Wait(ctx)
	}

	return nil
}
