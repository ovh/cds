package vsphere

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vmware/govmomi/guest"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

const reqTimeout = 7 * time.Second

//This a embeded cache for servers list
var lservers = struct {
	mu   sync.RWMutex
	list []mo.VirtualMachine
}{
	mu:   sync.RWMutex{},
	list: []mo.VirtualMachine{},
}

// get all servers on our host
func (h *HatcheryVSphere) getServers() []mo.VirtualMachine {
	var vms []mo.VirtualMachine
	ctx := context.Background()
	ctxC, cancelC := context.WithTimeout(ctx, reqTimeout)
	defer cancelC()

	t := time.Now()
	defer log.Debug("getServers() : %fs", time.Since(t).Seconds())

	m := view.NewManager(h.vclient.Client)

	v, errC := m.CreateContainerView(ctxC, h.vclient.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if errC != nil {
		log.Warning("Unable to create container view for vsphere api %s", errC)
		return lservers.list
	}
	defer v.Destroy(ctx)

	ctxR, cancelR := context.WithTimeout(context.Background(), reqTimeout)
	defer cancelR()
	// Retrieve summary property for all machines
	// Reference: http://pubs.vmware.com/vsphere-60/topic/com.vmware.wssdk.apiref.doc/vim.VirtualMachine.html
	if err := v.Retrieve(ctxR, []string{"VirtualMachine"}, []string{"name", "summary", "config"}, &vms); err != nil {
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

// get all servers tagged with model on our host
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

// Get a model by name
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

// Shutdown and delete a specific server
func (h *HatcheryVSphere) deleteServer(s mo.VirtualMachine) error {
	ctx := context.Background()
	vms, errVml := h.finder.VirtualMachineList(ctx, s.Name)
	if errVml != nil {
		return errVml
	}

	for _, vm := range vms {
		// If its a worker "register", check registration before deleting it
		var annot = annotation{}
		if err := json.Unmarshal([]byte(s.Config.Annotation), &annot); err != nil {
			log.Error("deleteServer> unable to get server annotation")
		} else {
			if strings.Contains(s.Name, "register-") {
				hatchery.CheckWorkerModelRegister(h, annot.WorkerModelID)
			}
		}

		ctxC, cancelC := context.WithTimeout(ctx, reqTimeout)
		defer cancelC()
		task, errOff := vm.PowerOff(ctxC)
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

// createVMConfig create a basic configuration in order to create a vm
func (h *HatcheryVSphere) createVMConfig(vm *object.VirtualMachine, annot annotation) (*types.VirtualMachineCloneSpec, *object.Folder, error) {
	ctx := context.Background()
	ctxC, cancelC := context.WithTimeout(ctx, reqTimeout)
	defer cancelC()

	folder, errF := h.finder.FolderOrDefault(ctxC, "")
	if errF != nil {
		return nil, folder, sdk.WrapError(errF, "createVMConfig> cannot find folder")
	}

	ctxC, cancelC = context.WithTimeout(ctx, reqTimeout)
	defer cancelC()
	devices, errD := vm.Device(ctxC)
	if errD != nil {
		return nil, folder, sdk.WrapError(errD, "createVMConfig> Cannot find device")
	}

	var card *types.VirtualEthernetCard
	for _, device := range devices {
		if c, ok := device.(types.BaseVirtualEthernetCard); ok {
			card = c.GetVirtualEthernetCard()
			break
		}
	}

	if card == nil {
		log.Warning("createVMConfig> no network device found")
		return nil, folder, fmt.Errorf("no network device found")
	}

	ctxC, cancelC = context.WithTimeout(ctx, reqTimeout)
	defer cancelC()
	backing, errB := h.network.EthernetCardBackingInfo(ctxC)
	if errB != nil {
		return nil, folder, sdk.WrapError(errB, "createVMConfig> cannot have ethernet backing info")
	}

	device, errE := object.EthernetCardTypes().CreateEthernetCard(h.cardName, backing)
	if errE != nil {
		return nil, folder, sdk.WrapError(errE, "createVMConfig> cannot create ethernet card")
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

	ctxC, cancelC = context.WithTimeout(ctx, reqTimeout)
	defer cancelC()
	datastore, errD := h.finder.DatastoreOrDefault(ctxC, h.datastoreString)
	if errD != nil {
		return nil, folder, sdk.WrapError(errD, "createVMConfig> cannot find datastore")
	}
	datastoreref := datastore.Reference()

	annotStr, errM := json.Marshal(annot)
	if errM != nil {
		return nil, folder, sdk.WrapError(errM, "createVMConfig> cannot marshall annotation")
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

// launchClientOp launch a script on the virtual machine given in paramters
func (h *HatcheryVSphere) launchClientOp(vm *object.VirtualMachine, script string, env []string) (int64, error) {
	ctx := context.Background()
	ctxC, cancelC := context.WithTimeout(ctx, reqTimeout)
	defer cancelC()

	running, errT := vm.IsToolsRunning(ctxC)
	if errT != nil {
		return -1, sdk.WrapError(errT, "launchClientOp> cannot fetch if tools are running")
	}
	if !running {
		log.Warning("launchClientOp> VmTools is not running")
	}

	opman := guest.NewOperationsManager(h.vclient.Client, vm.Reference())

	auth := types.NamePasswordAuthentication{
		Username: "root",
		Password: "",
	}

	ctxC, cancelC = context.WithTimeout(ctx, reqTimeout)
	defer cancelC()
	procman, errPr := opman.ProcessManager(ctxC)
	if errPr != nil {
		return -1, sdk.WrapError(errPr, "launchClientOp> cannot create processManager")
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
