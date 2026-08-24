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
)

// getModelConfig returns the ModelConfig for the given model, or nil if not configured.
func (h *HatcheryVSphere) getModelConfig(model sdk.WorkerStarterWorkerModel) *ModelConfig {
	return h.getModelConfigByImage(model.GetVSphereImage())
}

// getModelConfigByImage returns the ModelConfig for a VMware template name, or
// nil if not configured. The provisioning path only knows that name, never a
// sdk.WorkerStarterWorkerModel.
func (h *HatcheryVSphere) getModelConfigByImage(vmwareImage string) *ModelConfig {
	for i := range h.Config.Models {
		if h.Config.Models[i].ModelVMWare == vmwareImage {
			return &h.Config.Models[i]
		}
	}
	return nil
}

// isGuestInfoImage reports whether a VMware template is driven through guestinfo
// rather than guest customization and guest operations. See ModelConfig.GuestInfo.
func (h *HatcheryVSphere) isGuestInfoImage(vmwareImage string) bool {
	cfg := h.getModelConfigByImage(vmwareImage)
	return cfg != nil && cfg.GuestInfo
}

// reconfigureVM changes the CPU and RAM of a powered-off VM to match the requested flavor
func (h *HatcheryVSphere) reconfigureVM(ctx context.Context, vm *object.VirtualMachine, flavor *VSphereFlavorConfig) error {
	if flavor == nil {
		return nil
	}

	// Ensure VM is powered off
	powerState, err := vm.PowerState(ctx)
	if err != nil {
		return sdk.WrapError(err, "reconfigureVM> cannot get power state")
	}
	if powerState != types.VirtualMachinePowerStatePoweredOff {
		return fmt.Errorf("reconfigureVM> VM must be powered off to reconfigure (current state: %s)", powerState)
	}

	// Prepare reconfigure spec
	spec := types.VirtualMachineConfigSpec{}
	if flavor.CPUs > 0 {
		spec.NumCPUs = int32(flavor.CPUs)
	}
	if flavor.MemoryMB > 0 {
		spec.MemoryMB = int64(flavor.MemoryMB)
	}

	// Resize the first disk if DiskSizeGB is configured
	if flavor.DiskSizeGB > 0 {
		devices, err := h.vSphereClient.LoadVirtualMachineDevices(ctx, vm)
		if err != nil {
			return sdk.WrapError(err, "reconfigureVM> cannot load VM devices")
		}
		for _, device := range devices {
			if disk, ok := device.(*types.VirtualDisk); ok {
				newCapacityKB := int64(flavor.DiskSizeGB) * 1024 * 1024
				if newCapacityKB > disk.CapacityInKB {
					disk.CapacityInKB = newCapacityKB
					spec.DeviceChange = append(spec.DeviceChange, &types.VirtualDeviceConfigSpec{
						Operation: types.VirtualDeviceConfigSpecOperationEdit,
						Device:    disk,
					})
					log.Info(ctx, "reconfigureVM> resizing disk to %d GB", flavor.DiskSizeGB)
				}
				break
			}
		}
	}

	log.Info(ctx, "reconfigureVM> reconfiguring VM to %d vCPUs, %d MB RAM, disk %d GB", flavor.CPUs, flavor.MemoryMB, flavor.DiskSizeGB)

	if err := h.vSphereClient.ReconfigureVirtualMachine(ctx, vm, spec); err != nil {
		return sdk.WrapError(err, "reconfigureVM> cannot reconfigure VM")
	}

	log.Info(ctx, "reconfigureVM> VM successfully reconfigured")
	return nil
}

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
	result := make([]mo.VirtualMachine, 0, len(vms))
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
	result := make([]mo.VirtualMachine, 0, len(vms))
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
		annot := getVirtualMachineCDSAnnotation(ctx, srv)
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

	annot := getVirtualMachineCDSAnnotation(ctx, *vmRef)
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

		annot := getVirtualMachineCDSAnnotation(ctx, m)
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

	annot := getVirtualMachineCDSAnnotation(ctx, s)
	if annot == nil {
		return nil
	}

	isPoweredOn := s.Summary.Runtime.PowerState != types.VirtualMachinePowerStatePoweredOff

	if isPoweredOn {
		if err := h.vSphereClient.ShutdownVirtualMachine(ctx, vm); err != nil {
			log.Warn(ctx, "deleteServer> can't shutdown %q because err:%v", s.Name, err)
			// do not return here, the err could be :
			// err: The attempted operation cannot be performed in the current state (Powered off).
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

// prepareCloneSpec create a basic configuration in order to create a vm. When ip
// is non-nil the VM is given that static IP via guest customization and the IP is
// recorded in the annotation (the compatibility anchor); the IP itself is chosen
// by the caller (provisioningV2) so parallel clones never collide.
func (h *HatcheryVSphere) prepareCloneSpec(ctx context.Context, vm *object.VirtualMachine, annot *annotation, ip *ipResult) (*types.VirtualMachineCloneSpec, error) {
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
	networkConfigSpecs := []types.BaseVirtualDeviceConfigSpec{
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
		DeviceChange: networkConfigSpecs,
		DiskMoveType: string(types.VirtualMachineRelocateDiskMoveOptionsMoveChildMostDiskBacking),
		Pool:         types.NewReference(resPool.Reference()),
	}

	datastore, err := h.vSphereClient.LoadDatastore(ctx, h.Config.VSphereDatastoreString)
	if err != nil {
		return nil, err
	}
	datastoreref := datastore.Reference()

	// guestinfo models (see ModelConfig.GuestInfo) get their network through
	// guestinfo keys read by the guest at every boot: GOSC does not support them
	// and would reject or silently no-op the customization.
	guestInfo := h.isGuestInfoImage(annot.VMwareModelPath)

	customSpec := &types.CustomizationSpec{
		Identity: &types.CustomizationLinuxPrep{
			HostName: new(types.CustomizationVirtualMachineName),
		},
	}

	if ip != nil && !guestInfo {
		log.Debug(ctx, "assigning %s as IP (gw=%s, mask=%s)", ip.ip, ip.gateway, ip.subnetMask)

		customSpec.NicSettingMap = []types.CustomizationAdapterMapping{
			{
				Adapter: types.CustomizationIPSettings{
					Ip:         &types.CustomizationFixedIp{IpAddress: ip.ip},
					SubnetMask: ip.subnetMask,
				},
			},
		}
		if ip.gateway != "" {
			customSpec.NicSettingMap[0].Adapter.Gateway = []string{ip.gateway}
		}
		if h.Config.DNS != "" {
			customSpec.GlobalIPSettings = types.CustomizationGlobalIPSettings{DnsServerList: []string{h.Config.DNS}}
		}

		// Store the IP in the annotation too (compat anchor: old provisions and a
		// rolled-back binary read the IP from here).
		annot.IPAddress = ip.ip

		log.Debug(ctx, "IP: %s; Gateway: %v; DNS: %v", ip.ip, customSpec.NicSettingMap[0].Adapter.Gateway, customSpec.GlobalIPSettings.DnsServerList)
	}

	var extraConfig []types.BaseOptionValue
	if ip != nil && guestInfo {
		log.Debug(ctx, "assigning %s as IP through guestinfo (gw=%s, mask=%s)", ip.ip, ip.gateway, ip.subnetMask)

		extraConfig = append(extraConfig,
			&types.OptionValue{Key: "guestinfo.cds.net.ip", Value: ip.ip},
			&types.OptionValue{Key: "guestinfo.cds.net.mask", Value: ip.subnetMask},
			&types.OptionValue{Key: "guestinfo.cds.net.gateway", Value: ip.gateway},
			&types.OptionValue{Key: "guestinfo.cds.net.dns", Value: h.Config.DNS},
			&types.OptionValue{Key: "guestinfo.cds.net.hostname", Value: annot.WorkerName},
		)
		extraConfig = append(extraConfig, h.guestInfoAccessOptions()...)

		// Same compat anchor as the customization path: getUsedIPs reads the IP
		// from here to avoid handing it out twice.
		annot.IPAddress = ip.ip
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
			ExtraConfig: extraConfig,
		},
	}

	// A guestinfo model must not carry a customization spec at all: the GOSC
	// engine has no support for its guest OS. Everything it needs is in
	// ExtraConfig above.
	if !guestInfo {
		cloneSpec.Customization = customSpec
	}

	// Set the destination datastore
	cloneSpec.Location.Datastore = &datastoreref
	return cloneSpec, nil
}

// guestInfoAccessOptions returns the optional debug-access guestinfo keys, empty
// when nothing is configured — the guest image bakes no credentials and no
// allowlist, so an unset key means the worker is unreachable, which is the
// intended default. The two keys are separate layers: allowed_cidrs feeds the
// guest's packet filter (inbound SSH is dropped from anywhere else), while
// authorized_keys is what sshd authenticates once a packet gets through.
func (h *HatcheryVSphere) guestInfoAccessOptions() []types.BaseOptionValue {
	var opts []types.BaseOptionValue
	if len(h.Config.SSHAllowedCIDRs) > 0 {
		opts = append(opts, &types.OptionValue{
			Key:   "guestinfo.cds.access.allowed_cidrs",
			Value: strings.Join(h.Config.SSHAllowedCIDRs, " "),
		})
	}
	if len(h.Config.InjectSSHPublicKeys) > 0 {
		opts = append(opts, &types.OptionValue{
			Key:   "guestinfo.cds.access.authorized_keys",
			Value: strings.Join(h.Config.InjectSSHPublicKeys, "\n"),
		})
	}
	return opts
}

// launchClientOp launch a script on the virtual machine given in parameters
func (h *HatcheryVSphere) launchClientOp(ctx context.Context, vm *object.VirtualMachine, model sdk.WorkerStarterWorkerModel, script string, env []string) error {
	procman, err := h.vSphereClient.ProcessManager(ctx, vm)
	if err != nil {
		return err
	}

	var auth types.NamePasswordAuthentication

	// Look up credentials from Models config first
	if modelCfg := h.getModelConfig(model); modelCfg != nil {
		auth.Username = modelCfg.Username
		auth.Password = modelCfg.Password
	}

	// Fallback to deprecated GuestCredentials if not found in Models
	if auth.Username == "" || auth.Password == "" {
		for i := range h.Config.GuestCredentials {
			if h.Config.GuestCredentials[i].ModelVMWare == model.GetVSphereImage() {
				auth.Username = h.Config.GuestCredentials[i].Username
				auth.Password = h.Config.GuestCredentials[i].Password
				break
			}
		}
	}

	if auth.Username == "" || auth.Password == "" {
		return sdk.WithStack(fmt.Errorf("username and/or password not well configured for GetVSphereImage:%q", model.GetVSphereImage()))
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
