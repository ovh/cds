package vsphere

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rockbears/log"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

// get all servers on our host
func (h *HatcheryVSphere) getRawVMs(ctx context.Context) []mo.VirtualMachine {
	vms, err := h.vSphereClient.ListVirtualMachines(ctx)
	if err != nil {
		ctx = sdk.ContextWithStacktrace(ctx, err)
		log.Error(ctx, "unable to list virtual machines: %v", err)
		return nil
	}
	return vms
}

func (h *HatcheryVSphere) getVirtualMachines(ctx context.Context) []mo.VirtualMachine {
	vms := h.getRawVMs(ctx)
	var result = make([]mo.VirtualMachine, 0, len(vms))
	for i := range vms {
		isNotTemplate := !vms[i].Summary.Config.Template
		if isNotTemplate {
			result = append(result, vms[i])
		}
	}
	return result
}

func (h *HatcheryVSphere) getRawTemplates(ctx context.Context) []mo.VirtualMachine {
	vms := h.getRawVMs(ctx)
	var result = make([]mo.VirtualMachine, 0, len(vms))
	for i := range vms {
		isTemplate := vms[i].Summary.Config.Template
		if isTemplate {
			result = append(result, vms[i])
		}

	}
	return result
}

// get all servers tagged with model on our host
func (h *HatcheryVSphere) getVirtualMachineTemplates(ctx context.Context) []mo.VirtualMachine {
	srvs := h.getRawTemplates(ctx)
	models := make([]mo.VirtualMachine, 0, len(srvs))

	if len(srvs) == 0 {
		log.Warn(ctx, "getModels> no servers found")
		return nil
	}

	for _, srv := range srvs {
		if srv.Config == nil || srv.Config.Annotation == "" {
			log.Warn(ctx, "getModels> config or annotation are empty for server %s", srv.Name)
			continue
		}
		var annot = getVirtualMachineCDSAnnotation(ctx, srv)
		if annot != nil {
			if annot.Model {
				models = append(models, srv)
			}
			continue
		}
	}

	return models
}

func (h *HatcheryVSphere) getVirtualMachineByName(ctx context.Context, name string) (*mo.VirtualMachine, error) {
	// Reload the vm ref to get the annotation
	allVMRef, err := h.vSphereClient.ListVirtualMachines(ctx)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to get virtual machines")
	}

	var vmRef *mo.VirtualMachine
	for i := range allVMRef {
		if allVMRef[i].Name == name {
			vmRef = &allVMRef[i]
			break
		}
	}

	if vmRef == nil {
		err := sdk.WithStack(fmt.Errorf("virtual machine ref %q not found", name))
		return nil, sdk.WrapError(err, "unable to get virtual machine")
	}

	var annot = getVirtualMachineCDSAnnotation(ctx, *vmRef)
	if annot == nil {
		err := sdk.WithStack(fmt.Errorf("virtual machine ref %q not found", name))
		return nil, sdk.WrapError(err, "unable to get virtual machine")
	}

	return vmRef, nil
}

// Get a model by name
func (h *HatcheryVSphere) getVirtualMachineTemplateByName(ctx context.Context, name string) (mo.VirtualMachine, error) {
	models := h.getVirtualMachineTemplates(ctx)

	if len(models) == 0 {
		return mo.VirtualMachine{}, fmt.Errorf("no templates found")
	}

	for _, m := range models {
		if m.Name != name {
			log.Debug(ctx, "%q (%+v) doesn't match  with %q", m.Name, m.Config, name)
			continue
		}

		var annot = getVirtualMachineCDSAnnotation(ctx, m)
		if annot == nil {
			continue
		}

		if annot.Model {
			log.Debug(ctx, "found vm template %v", m.Name)
			return m, nil
		}
	}

	return mo.VirtualMachine{}, fmt.Errorf("template %q not found", name)
}

// Shutdown and delete a specific server
func (h *HatcheryVSphere) deleteServer(ctx context.Context, s mo.VirtualMachine) error {
	vm, err := h.vSphereClient.LoadVirtualMachine(ctx, s.Name)
	if err != nil {
		return err
	}

	var annot = getVirtualMachineCDSAnnotation(ctx, s)
	if annot == nil {
		return nil
	}

	if strings.HasPrefix(s.Name, "register-") {
		if err := hatchery.CheckWorkerModelRegister(ctx, h, annot.WorkerModelPath); err != nil {
			var spawnErr = sdk.SpawnErrorForm{
				Error: err.Error(),
			}
			log.Error(ctx, "failed check worker model register: %v", err)
			tuple := strings.SplitN(annot.WorkerModelPath, "/", 2)
			if err := h.CDSClient().WorkerModelSpawnError(tuple[0], tuple[1], spawnErr); err != nil {
				log.Error(ctx, "CheckWorkerModelRegister> error on call client.WorkerModelSpawnError on worker model %s for register: %v", annot.WorkerModelPath, err)
			}
		}
	}

	var isPoweredOn = s.Summary.Runtime.PowerState != types.VirtualMachinePowerStatePoweredOff

	if isPoweredOn {
		if err := h.vSphereClient.ShutdownVirtualMachine(ctx, vm); err != nil {
			return err
		}
	}

	h.cacheToDelete.mu.Lock()
	h.cacheToDelete.list = sdk.DeleteFromArray(h.cacheToDelete.list, s.Name)
	h.cacheToDelete.mu.Unlock()

	if err := h.vSphereClient.DestroyVirtualMachine(ctx, vm); err != nil {
		return err
	}

	return nil
}

// prepareCloneSpec create a basic configuration in order to create a vm
func (h *HatcheryVSphere) prepareCloneSpec(ctx context.Context, vm *object.VirtualMachine, annot *annotation) (*types.VirtualMachineCloneSpec, error) {
	devices, err := h.vSphereClient.LoadVirtualMachineDevices(ctx, vm)
	if err != nil {
		return nil, err
	}

	var card *types.VirtualEthernetCard
	for _, device := range devices {
		if c, ok := device.(types.BaseVirtualEthernetCard); ok {
			card = c.GetVirtualEthernetCard()
			break
		}
	}

	if card == nil {
		return nil, sdk.WithStack(fmt.Errorf("no network device found"))
	}

	network, err := h.vSphereClient.LoadNetwork(ctx, h.Config.VSphereNetworkString)
	if err != nil {
		return nil, err
	}

	if err := h.vSphereClient.SetupEthernetCard(ctx, card, h.Config.VSphereCardName, network); err != nil {
		return nil, err
	}

	// prepare virtual device config spec for network card
	configSpecs := []types.BaseVirtualDeviceConfigSpec{
		&types.VirtualDeviceConfigSpec{
			Operation: types.VirtualDeviceConfigSpecOperationEdit,
			Device:    card,
		},
	}

	resPool, err := h.vSphereClient.LoadResourcePool(ctx)
	if err != nil {
		return nil, err
	}

	relocateSpec := types.VirtualMachineRelocateSpec{
		DeviceChange: configSpecs,
		DiskMoveType: string(types.VirtualMachineRelocateDiskMoveOptionsMoveChildMostDiskBacking),
		Pool:         types.NewReference(resPool.Reference()),
	}

	datastore, err := h.vSphereClient.LoadDatastore(ctx, h.Config.VSphereDatastoreString)
	if err != nil {
		return nil, err
	}
	datastoreref := datastore.Reference()

	customSpec := &types.CustomizationSpec{
		Identity: &types.CustomizationLinuxPrep{
			HostName: new(types.CustomizationVirtualMachineName),
		},
	}

	if len(h.availableIPAddresses) > 0 {
		var err error
		ip, err := h.findAvailableIP(ctx)
		if err != nil {
			return nil, err
		}
		log.Debug(ctx, "Found %s as available IP", ip)
		// Once we found an IP Address, we have to reserve this IP in local memory
		// because the IP address won't be used directly on the server
		if err := h.reserveIPAddress(ctx, ip); err != nil {
			return nil, err
		}

		customSpec.NicSettingMap = []types.CustomizationAdapterMapping{{
			Adapter: types.CustomizationIPSettings{
				Ip:         &types.CustomizationFixedIp{IpAddress: ip},
				SubnetMask: h.Config.SubnetMask,
			}},
		}
		if h.Config.Gateway != "" {
			customSpec.NicSettingMap[0].Adapter.Gateway = []string{h.Config.Gateway}
		}
		if h.Config.DNS != "" {
			customSpec.GlobalIPSettings = types.CustomizationGlobalIPSettings{DnsServerList: []string{h.Config.DNS}}
		}

		annot.IPAddress = ip

		log.Debug(ctx, "IP: %s; Gateway: %v; DNS: %v", ip, customSpec.NicSettingMap[0].Adapter.Gateway, customSpec.GlobalIPSettings.DnsServerList)
	}

	annotStr, err := json.Marshal(annot)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to marshal annotation")
	}

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
				AfterPowerOn: &sdk.True,
			},
		},
	}

	cloneSpec.Customization = customSpec

	// Set the destination datastore
	cloneSpec.Location.Datastore = &datastoreref
	return cloneSpec, nil
}

// launchClientOp launch a script on the virtual machine given in parameters
func (h *HatcheryVSphere) launchClientOp(ctx context.Context, vm *object.VirtualMachine, model sdk.WorkerStarterWorkerModel, script string, env []string) error {
	procman, err := h.vSphereClient.ProcessManager(ctx, vm)
	if err != nil {
		return err
	}

	var auth types.NamePasswordAuthentication

	if model.ModelV2 != nil {
		// For model v2, we search the credentials from the VM Model name
		for i := range h.Config.GuestCredentials {
			if h.Config.GuestCredentials[i].ModelVMWare == model.GetVSphereImage() {
				auth.Username = h.Config.GuestCredentials[i].Username
				auth.Password = h.Config.GuestCredentials[i].Password
				break
			}
		}
	} else {
		// For model v1, we search the credentials from the CDS Worker model name
		for i := range h.Config.GuestCredentials {
			if h.Config.GuestCredentials[i].ModelPath == model.GetFullPath() {
				auth.Username = h.Config.GuestCredentials[i].Username
				auth.Password = h.Config.GuestCredentials[i].Password
				break
			}
		}
	}

	if auth.Username == "" || auth.Password == "" {
		if model.ModelV2 != nil {
			return sdk.WithStack(fmt.Errorf("username and/or password not well configured for GetVSphereImage:%q", model.GetVSphereImage()))
		} else {
			return sdk.WithStack(fmt.Errorf("username and/or password not well configured for modelFullPath:%q", model.GetFullPath()))
		}
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

	log.Debug(ctx, "starting program in guest. username:%v ProgramPath:%v Arguments:%v", auth.Username, guestspec.ProgramPath, guestspec.Arguments)

	_, err = h.vSphereClient.StartProgramInGuest(ctx, procman, &req)
	return err
}
