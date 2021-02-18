package vsphere

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rockbears/log"
	"github.com/vmware/govmomi/guest"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

const reqTimeout = 20 * time.Second

//This a embedded cache for servers list
var lservers = struct {
	mu   sync.RWMutex
	list []mo.VirtualMachine
}{
	mu:   sync.RWMutex{},
	list: []mo.VirtualMachine{},
}

// get all servers on our host
func (h *HatcheryVSphere) getServers(ctx context.Context) []mo.VirtualMachine {
	var vms []mo.VirtualMachine
	ctxC, cancelC := context.WithTimeout(ctx, reqTimeout)
	defer cancelC()

	m := view.NewManager(h.vclient.Client)

	v, err := m.CreateContainerView(ctxC, h.vclient.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		log.Warn(ctx, "Unable to create container view for vsphere api: %v", err)
		return lservers.list
	}
	defer v.Destroy(ctx)

	ctxR, cancelR := context.WithTimeout(ctx, reqTimeout)
	defer cancelR()
	// Retrieve summary property for all machines
	// Reference: http://pubs.vmware.com/vsphere-60/topic/com.vmware.wssdk.apiref.doc/vim.VirtualMachine.html
	if err := v.Retrieve(ctxR, []string{"VirtualMachine"}, []string{"name", "summary", "guest", "config"}, &vms); err != nil {
		log.Warn(ctx, "Unable to retrieve virtual machines from vsphere: %v", err)
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

//This a embedded cache for images list
var lmodels = struct {
	mu   sync.RWMutex
	list []mo.VirtualMachine
}{
	mu:   sync.RWMutex{},
	list: []mo.VirtualMachine{},
}

// get all servers tagged with model on our host
func (h *HatcheryVSphere) getModels(ctx context.Context) []mo.VirtualMachine {
	srvs := h.getServers(ctx)
	models := make([]mo.VirtualMachine, len(srvs))

	if len(srvs) == 0 {
		log.Warn(ctx, "getModels> no servers found")
		return lmodels.list
	}

	for _, srv := range srvs {
		var annot annotation

		if srv.Config == nil || srv.Config.Annotation == "" {
			log.Warn(ctx, "getModels> config or annotation are empty for server %s", srv.Name)
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
func (h *HatcheryVSphere) getModelByName(ctx context.Context, name string) (mo.VirtualMachine, error) {
	models := h.getModels(ctx)

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
func (h *HatcheryVSphere) deleteServer(ctx context.Context, s mo.VirtualMachine) error {
	vms, errVml := h.finder.VirtualMachineList(ctx, s.Name)
	if errVml != nil {
		return errVml
	}

	for _, vm := range vms {
		// If its a worker "register", check registration before deleting it
		var annot = annotation{}
		if err := json.Unmarshal([]byte(s.Config.Annotation), &annot); err != nil {
			log.Error(ctx, "deleteServer> unable to get server annotation")
		} else {
			if strings.HasPrefix(s.Name, "register-") {
				if err := hatchery.CheckWorkerModelRegister(h, annot.WorkerModelPath); err != nil {
					var spawnErr = sdk.SpawnErrorForm{
						Error: err.Error(),
					}
					tuple := strings.SplitN(annot.WorkerModelPath, "/", 2)
					if err := h.CDSClient().WorkerModelSpawnError(tuple[0], tuple[1], spawnErr); err != nil {
						log.Error(ctx, "CheckWorkerModelRegister> error on call client.WorkerModelSpawnError on worker model %s for register: %v", annot.WorkerModelPath, err)
					}
				}
			}
		}

		log.Info(ctx, "shuting down server %v", s.Name)

		ctxC, cancelC := context.WithTimeout(ctx, reqTimeout)
		defer cancelC()
		task, errOff := vm.PowerOff(ctxC)
		if errOff != nil {
			return sdk.WithStack(errOff)
		}
		task.Wait(ctx)

		var errD error
		log.Info(ctx, "destroying server %v", s.Name)
		task, errD = vm.Destroy(ctx)
		if errD != nil {
			return errD
		}

		return sdk.WithStack(task.Wait(ctx))
	}

	return nil
}

// createVMConfig create a basic configuration in order to create a vm
func (h *HatcheryVSphere) createVMConfig(ctx context.Context, vm *object.VirtualMachine, annot annotation, workerName string) (*types.VirtualMachineCloneSpec, *object.Folder, error) {
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
		log.Warn(ctx, "createVMConfig> no network device found")
		return nil, folder, sdk.WithStack(fmt.Errorf("no network device found"))
	}

	ctxC, cancelC = context.WithTimeout(ctx, reqTimeout)
	defer cancelC()
	backing, errB := h.network.EthernetCardBackingInfo(ctxC)
	if errB != nil {
		return nil, folder, sdk.WrapError(errB, "createVMConfig> cannot have ethernet backing info")
	}

	device, errE := object.EthernetCardTypes().CreateEthernetCard(h.Config.VSphereCardName, backing)
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
	resPool, err := h.finder.DefaultResourcePool(ctx)
	if err != nil {
		return nil, folder, sdk.WrapError(err, "unable to get default resource pool")
	}

	relocateSpec := types.VirtualMachineRelocateSpec{
		DeviceChange: configSpecs,
		DiskMoveType: string(types.VirtualMachineRelocateDiskMoveOptionsMoveChildMostDiskBacking),
		Pool:         types.NewReference(resPool.Reference()),
	}

	ctxC, cancelC = context.WithTimeout(ctx, reqTimeout)
	defer cancelC()
	datastore, errD := h.finder.DatastoreOrDefault(ctxC, h.Config.VSphereDatastoreString)
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

	customSpec := &types.CustomizationSpec{
		Identity: &types.CustomizationLinuxPrep{
			HostName: new(types.CustomizationVirtualMachineName),
		},
	}
	// Ip len(ipsInfos.ips) > 0, specify one of those
	if len(ipsInfos.ips) > 0 {
		var err error
		ip, err := h.findAvailableIP(ctx, workerName)
		if err != nil {
			return nil, folder, sdk.WithStack(err)
		}
		log.Debug(ctx, "Found %s as available IP", ip)
		customSpec.NicSettingMap = []types.CustomizationAdapterMapping{
			{
				Adapter: types.CustomizationIPSettings{
					Ip:         &types.CustomizationFixedIp{IpAddress: ip},
					SubnetMask: h.Config.SubnetMask,
				},
			},
		}
		if h.Config.Gateway != "" {
			customSpec.NicSettingMap[0].Adapter.Gateway = []string{h.Config.Gateway}
		}
		if h.Config.DNS != "" {
			customSpec.GlobalIPSettings = types.CustomizationGlobalIPSettings{DnsServerList: []string{h.Config.DNS}}
		}
		log.Debug(ctx, "%s / %+v / %+v", ip, customSpec.NicSettingMap[0].Adapter.Gateway, customSpec.NicSettingMap[0].Adapter.DnsServerList)
	}
	cloneSpec.Customization = customSpec

	// FIXME Windows Identity
	/*
		customSpec.Identity = &types.CustomizationSysprep{
			UserData: types.CustomizationUserData{
				ComputerName: new(types.CustomizationVirtualMachineName),
			},
		}
	*/

	// Set the destination datastore
	cloneSpec.Location.Datastore = &datastoreref

	return cloneSpec, folder, nil
}

// launchClientOp launch a script on the virtual machine given in parameters
func (h *HatcheryVSphere) launchClientOp(ctx context.Context, vm *object.VirtualMachine, model sdk.ModelVirtualMachine, script string, env []string) (int64, error) {
	ctxA, cancel := context.WithTimeout(ctx, reqTimeout)
	defer cancel()
	running, err := vm.IsToolsRunning(ctxA)
	if err != nil {
		return -1, sdk.WrapError(err, "launchClientOp> cannot fetch if tools are running")
	}
	if !running {
		log.Warn(ctx, "launchClientOp> VmTools is not running")
	}

	opman := guest.NewOperationsManager(h.vclient.Client, vm.Reference())

	procman, errPr := opman.ProcessManager(ctx)
	if errPr != nil {
		return -1, sdk.WrapError(errPr, "launchClientOp> cannot create processManager")
	}

	auth := types.NamePasswordAuthentication{
		Username: model.User,
		Password: model.Password,
	}

	guestspec := types.GuestProgramSpec{
		ProgramPath:  "/bin/echo",
		Arguments:    "-n ;" + script,
		EnvVariables: env,
	}

	req := types.StartProgramInGuest{
		This: procman.Reference(),
		Vm:   vm.Reference(),
		Auth: &auth,
		Spec: &guestspec,
	}

	log.Debug(ctx, "starting program %+v in guest...", guestspec)
	ctxB, cancel := context.WithTimeout(ctx, reqTimeout)
	defer cancel()

	res, err := methods.StartProgramInGuest(ctxB, procman.Client(), &req)
	if res != nil {
		log.Debug(ctx, "program result: %+v", res)
	}
	if err != nil {
		return 0, sdk.WrapError(err, "unable to start program in guest")
	}

	return res.Returnval, nil
}
