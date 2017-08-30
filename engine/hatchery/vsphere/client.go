package vsphere

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/vmware/govmomi/guest"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/ovh/cds/sdk/log"
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

	m := view.NewManager(h.vclient.Client)

	v, err := m.CreateContainerView(ctx, h.vclient.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		log.Error("Unable to create container view for vsphere api")
		return lservers.list
	}
	defer v.Destroy(ctx)

	// Retrieve summary property for all machines
	// Reference: http://pubs.vmware.com/vsphere-60/topic/com.vmware.wssdk.apiref.doc/vim.VirtualMachine.html
	if err := v.Retrieve(ctx, []string{"VirtualMachine"}, []string{"name", "summary", "config"}, &vms); err != nil {
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

	return lservers.list
}

//This a embeded cache for images list
var lmodels = struct {
	mu   sync.RWMutex
	list []mo.VirtualMachine
}{
	mu:   sync.RWMutex{},
	list: []mo.VirtualMachine{},
}

func (h *HatcheryVSphere) getModels() []mo.VirtualMachine {
	srvs := h.getServers()
	models := make([]mo.VirtualMachine, len(srvs))

	if len(srvs) == 0 {
		log.Warning("getModels> no servers found")
		return lmodels.list
	}

	for _, srv := range srvs {
		var annot annotation

		if srv.Config == nil || srv.Config.Annotation == "" {
			log.Warning("getModels> config or annotation are empty for server %s", srv.Name)
			continue
		}
		if err := json.Unmarshal([]byte(srv.Config.Annotation), &annot); err == nil {
			if annot.Model {
				models = append(models, srv)
			}
		}
	}

	lmodels.mu.Lock()
	lmodels.list = models
	lmodels.mu.Unlock()
	//Remove data from the cache after 2 seconds
	go func() {
		time.Sleep(2 * time.Second)
		lmodels.mu.Lock()
		lmodels.list = []mo.VirtualMachine{}
		lmodels.mu.Unlock()
	}()

	return models
}

func (h *HatcheryVSphere) getModelByName(name string) (mo.VirtualMachine, error) {
	models := h.getModels()

	if len(models) == 0 {
		return mo.VirtualMachine{}, fmt.Errorf("no models list found")
	}

	for _, m := range models {
		var annot annotation
		if m.Config == nil || m.Config.Annotation == "" || m.Name != name {
			continue
		}
		if err := json.Unmarshal([]byte(m.Config.Annotation), &annot); err == nil && annot.Model {
			return m, nil
		}
	}

	return mo.VirtualMachine{}, fmt.Errorf("model not found")
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

func (h *HatcheryVSphere) createVMConfig(vm *object.VirtualMachine, annot annotation) (*types.VirtualMachineCloneSpec, *object.Folder, error) {
	ctx := context.Background()

	folder, errF := h.finder.FolderOrDefault(ctx, "")
	if errF != nil {
		log.Warning("createVMConfig> cannot find folder")
		return nil, folder, errF
	}

	devices, errD := vm.Device(ctx)
	if errD != nil {
		log.Warning("createVMConfig> Cannot find device")
		return nil, folder, errD
	}

	var card *types.VirtualEthernetCard
	for _, device := range devices {
		if c, ok := device.(types.BaseVirtualEthernetCard); ok {
			card = c.GetVirtualEthernetCard()
			break
		}
	}

	if card == nil {
		log.Warning("createVMConfig> no network device found.")
		return nil, folder, fmt.Errorf("no network device found.")
	}

	backing, errB := h.network.EthernetCardBackingInfo(ctx)
	if errB != nil {
		log.Warning("createVMConfig> cannot have ethernet backing info")
		return nil, folder, errB
	}

	device, errE := object.EthernetCardTypes().CreateEthernetCard("e1000", backing)
	if errE != nil {
		log.Warning("createVMConfig> cannot create ethernet card")
		return nil, folder, errE
	}
	//set backing info
	card.Backing = device.(types.BaseVirtualEthernetCard).GetVirtualEthernetCard().Backing

	// prepare virtual device config spec for network card
	configSpecs := []types.BaseVirtualDeviceConfigSpec{
		&types.VirtualDeviceConfigSpec{
			Operation: types.VirtualDeviceConfigSpecOperationEdit,
			Device:    card,
		},
	}

	relocateSpec := types.VirtualMachineRelocateSpec{
		DeviceChange: configSpecs,
		DiskMoveType: string(types.VirtualMachineRelocateDiskMoveOptionsMoveChildMostDiskBacking),
	}

	datastore, errD := h.finder.DatastoreOrDefault(ctx, h.datastoreString)
	if errD != nil {
		log.Warning("createVMConfig> cannot find datastore")
		return nil, folder, errD
	}
	datastoreref := datastore.Reference()

	annotStr, errM := json.Marshal(annot)
	if errM != nil {
		log.Warning("createVMConfig> cannot marshall annotation")
		return nil, folder, errM
	}

	afterPO := true
	cloneSpec := &types.VirtualMachineCloneSpec{
		Location: relocateSpec,
		PowerOn:  true,
		Template: false,
		Config: &types.VirtualMachineConfigSpec{
			RepConfig: &types.ReplicationConfigSpec{
				QuiesceGuestEnabled: false,
			},
			Annotation: string(annotStr),
			Tools: &types.ToolsConfigInfo{
				AfterPowerOn: &afterPO,
			},
		},
	}

	// Set the destination datastore
	cloneSpec.Location.Datastore = &datastoreref

	return cloneSpec, folder, nil
}

func (h *HatcheryVSphere) launchClientOp(vm *object.VirtualMachine, script string, env []string) (int64, error) {
	ctx := context.Background()

	running, errT := vm.IsToolsRunning(ctx)
	if errT != nil {
		log.Warning("launchClientOp> cannot fetch if tools are running %s", errT)
		return -1, errT
	}
	if !running {
		log.Warning("launchClientOp> VmTools is not running")
	}

	opman := guest.NewOperationsManager(h.vclient.Client, vm.Reference())

	auth := types.NamePasswordAuthentication{
		Username: "root",
		Password: "",
	}

	procman, errPr := opman.ProcessManager(ctx)
	if errPr != nil {
		log.Warning("launchClientOp> cannot create processManager %s", errPr)
		return -1, errPr
	}

	guestspec := types.GuestProgramSpec{
		ProgramPath:      "/bin/echo",
		Arguments:        "-n ;" + script,
		WorkingDirectory: "/root",
		EnvVariables:     env,
	}

	ctxTo, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()

	return procman.StartProgram(ctxTo, &auth, &guestspec)
}
