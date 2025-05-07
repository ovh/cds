package vsphere

import (
	"context"
	"fmt"
	"time"

	"github.com/rockbears/log"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/event"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/guest"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/ovh/cds/sdk"
)

var properties = []string{"name", "summary", "guest", "config"}

type VSphereClient interface {
	ListVirtualMachines(ctx context.Context) ([]mo.VirtualMachine, error)
	LoadVirtualMachine(ctx context.Context, name string) (*object.VirtualMachine, error)
	LoadVirtualMachineDevices(ctx context.Context, vm *object.VirtualMachine) (object.VirtualDeviceList, error)
	StartVirtualMachine(ctx context.Context, vm *object.VirtualMachine) error
	ShutdownVirtualMachine(ctx context.Context, vm *object.VirtualMachine) error
	DestroyVirtualMachine(ctx context.Context, vm *object.VirtualMachine) error
	CloneVirtualMachine(ctx context.Context, vm *object.VirtualMachine, folder *object.Folder, name string, config *types.VirtualMachineCloneSpec) (*types.ManagedObjectReference, error)
	GetVirtualMachinePowerState(ctx context.Context, vm *object.VirtualMachine) (types.VirtualMachinePowerState, error)
	NewVirtualMachine(ctx context.Context, cloneSpec *types.VirtualMachineCloneSpec, ref *types.ManagedObjectReference, vmName string) (*object.VirtualMachine, error)
	RenameVirtualMachine(ctx context.Context, vm *object.VirtualMachine, newName string) error
	MarkVirtualMachineAsTemplate(ctx context.Context, vm *object.VirtualMachine) error
	WaitForVirtualMachineShutdown(ctx context.Context, vm *object.VirtualMachine) error
	WaitForVirtualMachineIP(ctx context.Context, vm *object.VirtualMachine, IPAddress *string, vmName string) error
	LoadFolder(ctx context.Context) (*object.Folder, error)
	SetupEthernetCard(ctx context.Context, card *types.VirtualEthernetCard, ethernetCardName string, network object.NetworkReference) error
	LoadNetwork(ctx context.Context, name string) (object.NetworkReference, error)
	LoadResourcePool(ctx context.Context) (*object.ResourcePool, error)
	LoadDatastore(ctx context.Context, name string) (*object.Datastore, error)
	ProcessManager(ctx context.Context, vm *object.VirtualMachine) (*guest.ProcessManager, error)
	StartProgramInGuest(ctx context.Context, procman *guest.ProcessManager, req *types.StartProgramInGuest) (*types.StartProgramInGuestResponse, error)
	LoadVirtualMachineEvents(ctx context.Context, vm *object.VirtualMachine, eventTypes ...string) ([]types.BaseEvent, error)
}

func NewVSphereClient(vclient *govmomi.Client, datacenter string) VSphereClient {
	return &vSphereClient{
		vclient:        vclient,
		requestTimeout: 15 * time.Second,
		datacenter:     datacenter,
	}
}

type vSphereClient struct {
	datacenter     string
	vclient        *govmomi.Client
	requestTimeout time.Duration
}

func (c *vSphereClient) finder(ctx context.Context) (*find.Finder, error) {
	finder := find.NewFinder(c.vclient.Client, false)

	datacenter, err := finder.DatacenterOrDefault(ctx, c.datacenter)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to find datacenter %q", err)
	}
	finder.SetDatacenter(datacenter)

	return finder, nil
}

func (c *vSphereClient) ListVirtualMachines(ctx context.Context) ([]mo.VirtualMachine, error) {
	var vms []mo.VirtualMachine
	var m = view.NewManager(c.vclient.Client)

	ctxC, cancelC := context.WithTimeout(ctx, c.requestTimeout)
	defer cancelC()

	v, err := m.CreateContainerView(ctxC, c.vclient.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to create container view for vsphere api")
	}
	defer v.Destroy(ctx) // nolint

	ctxR, cancelR := context.WithTimeout(ctx, c.requestTimeout)
	defer cancelR()
	// Retrieve summary property for all machines
	if err := v.Retrieve(ctxR, []string{"VirtualMachine"}, properties, &vms); err != nil {
		return nil, sdk.WrapError(err, "unable to retrieve virtual machines from vsphere")
	}

	return vms, nil
}

func (c *vSphereClient) LoadVirtualMachine(ctx context.Context, name string) (*object.VirtualMachine, error) {
	finder, err := c.finder(ctx)
	if err != nil {
		return nil, err
	}

	vm, err := finder.VirtualMachine(ctx, name)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to find virtual machine %q", name)
	}

	return vm, nil
}

// eventTypes could be:
/*
EventEx
TaskEvent
VmPoweredOffEvent
VmPoweredOnEvent
VmReconfiguredEvent
VmStartingEvent
...
*/
func (c *vSphereClient) LoadVirtualMachineEvents(ctx context.Context, vm *object.VirtualMachine, eventTypes ...string) ([]types.BaseEvent, error) {
	m := event.NewManager(c.vclient.Client)
	objs := []types.ManagedObjectReference{vm.Reference()}

	var res []types.BaseEvent
	m.Events(ctx, objs, 50, false, false, func(ref types.ManagedObjectReference, events []types.BaseEvent) error {
		event.Sort(events)
		res = events
		return nil
	}, eventTypes...)

	return res, nil
}

func (c *vSphereClient) LoadVirtualMachineDevices(ctx context.Context, vm *object.VirtualMachine) (object.VirtualDeviceList, error) {
	ctxC, cancelC := context.WithTimeout(ctx, c.requestTimeout)
	defer cancelC()

	devices, err := vm.Device(ctxC)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to get vm devices")
	}

	return devices, nil
}

func (c *vSphereClient) StartVirtualMachine(ctx context.Context, vm *object.VirtualMachine) error {
	log.Info(ctx, "starting server %v", vm.Name())

	ctxC, cancelC := context.WithTimeout(ctx, c.requestTimeout)
	defer cancelC()

	task, err := vm.PowerOn(ctxC)
	if err != nil {
		return sdk.WithStack(err)
	}
	return sdk.WithStack(task.Wait(ctx))
}

func (c *vSphereClient) ShutdownVirtualMachine(ctx context.Context, vm *object.VirtualMachine) error {
	log.Info(ctx, "shutdown server %v", vm.Name())

	ctxC, cancelC := context.WithTimeout(ctx, c.requestTimeout)
	defer cancelC()

	task, err := vm.PowerOff(ctxC)
	if err != nil {
		return sdk.WithStack(err)
	}
	return sdk.WithStack(task.Wait(ctx))
}

func (c *vSphereClient) DestroyVirtualMachine(ctx context.Context, vm *object.VirtualMachine) error {
	log.Info(ctx, "destroying server %v", vm.Name())

	ctxD, cancelD := context.WithTimeout(ctx, c.requestTimeout)
	defer cancelD()

	task, err := vm.Destroy(ctxD)
	if err != nil {
		return sdk.WithStack(err)
	}

	return sdk.WithStack(task.Wait(ctx))
}

func (c *vSphereClient) LoadFolder(ctx context.Context) (*object.Folder, error) {
	finder, err := c.finder(ctx)
	if err != nil {
		return nil, err
	}

	ctxC, cancelC := context.WithTimeout(ctx, c.requestTimeout)
	defer cancelC()

	folder, err := finder.FolderOrDefault(ctxC, "")
	if err != nil {
		return nil, sdk.WrapError(err, "cannot find folder")
	}

	return folder, nil
}

func (c *vSphereClient) LoadNetwork(ctx context.Context, name string) (object.NetworkReference, error) {
	finder, err := c.finder(ctx)
	if err != nil {
		return nil, err
	}

	ctxC, cancelC := context.WithTimeout(ctx, c.requestTimeout)
	defer cancelC()

	network, err := finder.NetworkOrDefault(ctxC, name)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to find network %s", err)
	}

	return network, nil
}

func (c *vSphereClient) SetupEthernetCard(ctx context.Context, card *types.VirtualEthernetCard, ethernetCardName string, network object.NetworkReference) error {
	ctxC, cancelC := context.WithTimeout(ctx, c.requestTimeout)
	defer cancelC()

	backing, err := network.EthernetCardBackingInfo(ctxC)
	if err != nil {
		return sdk.WrapError(err, "cannot have ethernet backing info")
	}

	device, err := object.EthernetCardTypes().CreateEthernetCard(ethernetCardName, backing)
	if err != nil {
		return sdk.WrapError(err, "cannot create ethernet card")
	}

	//set backing info
	card.Backing = device.(types.BaseVirtualEthernetCard).GetVirtualEthernetCard().Backing

	return nil
}

func (c *vSphereClient) LoadResourcePool(ctx context.Context) (*object.ResourcePool, error) {
	finder, err := c.finder(ctx)
	if err != nil {
		return nil, err
	}

	pool, err := finder.DefaultResourcePool(ctx)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to get default resource pool")
	}
	return pool, nil
}

func (c *vSphereClient) LoadDatastore(ctx context.Context, name string) (*object.Datastore, error) {
	finder, err := c.finder(ctx)
	if err != nil {
		return nil, err
	}

	ctxC, cancelC := context.WithTimeout(ctx, c.requestTimeout)
	defer cancelC()

	datastore, err := finder.DatastoreOrDefault(ctxC, name)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot find datastore %q", name)
	}

	return datastore, nil
}

func (c *vSphereClient) CloneVirtualMachine(ctx context.Context, vm *object.VirtualMachine, folder *object.Folder, name string, config *types.VirtualMachineCloneSpec) (*types.ManagedObjectReference, error) {
	task, err := vm.Clone(ctx, folder, name, *config)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot clone VM name %v", name)
	}

	info, err := task.WaitForResult(ctx, nil)
	if err != nil || info.State == types.TaskInfoStateError {
		return nil, sdk.WrapError(err, "state in error: %+v", info)
	}

	res := info.Result.(types.ManagedObjectReference)

	log.Debug(ctx, "VM cloned: %+v", res)
	return &res, nil
}

func (c *vSphereClient) ProcessManager(ctx context.Context, vm *object.VirtualMachine) (*guest.ProcessManager, error) {
	ctxA, cancel := context.WithTimeout(ctx, c.requestTimeout)
	defer cancel()

	running, err := vm.IsToolsRunning(ctxA)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to fetch if VmTools are running on %q", vm.String())
	}
	if !running {
		log.Warn(ctx, "VmTools is not running")
	}

	opman := guest.NewOperationsManager(c.vclient.Client, vm.Reference())

	procman, err := opman.ProcessManager(ctx)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot create processManager")
	}

	return procman, nil
}

func (c *vSphereClient) StartProgramInGuest(ctx context.Context, procman *guest.ProcessManager, req *types.StartProgramInGuest) (*types.StartProgramInGuestResponse, error) {
	ctxB, cancel := context.WithTimeout(ctx, c.requestTimeout)
	defer cancel()

	res, err := methods.StartProgramInGuest(ctxB, procman.Client(), req)
	return res, sdk.WrapError(err, "unable to start program in guest")
}

func (c *vSphereClient) NewVirtualMachine(ctx context.Context, cloneSpec *types.VirtualMachineCloneSpec, ref *types.ManagedObjectReference, vmName string) (*object.VirtualMachine, error) {
	vm := object.NewVirtualMachine(c.vclient.Client, *ref)
	// vm.Name() is empty here

	log.Debug(ctx, "new virtual machine %q is nearly ready...", vmName)

	ctxReady, cancelReady := context.WithTimeout(ctx, 3*time.Minute)
	defer cancelReady()

	var isGuestReady bool
	for !isGuestReady {
		if ctxReady.Err() != nil {
			return nil, sdk.WithStack(fmt.Errorf("vm %q guest operation is not ready: %v", vmName, ctxReady.Err()))
		}

		var o mo.VirtualMachine
		if err := vm.Properties(ctx, *ref, properties, &o); err != nil {
			return nil, sdk.WrapError(err, "unable to get vm %q properties", vmName)
		}

		var operationReady = o.Guest.GuestOperationsReady
		if operationReady != nil && *operationReady {
			isGuestReady = true
		}
	}

	var expectedIP *string
	customFixedIP, ok := cloneSpec.Customization.NicSettingMap[0].Adapter.Ip.(*types.CustomizationFixedIp)
	if ok {
		expectedIP = &customFixedIP.IpAddress
	}

	if err := c.WaitForVirtualMachineIP(ctx, vm, expectedIP, vmName); err != nil {
		return vm, err
	}

	return vm, nil
}

func (c *vSphereClient) WaitVirtualMachineForShutdown(ctx context.Context, vm *object.VirtualMachine) error {
	ctxTo, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	log.Debug(ctx, "waiting virtual machine %q to be powered off...", vm.String())
	if err := vm.WaitForPowerState(ctxTo, types.VirtualMachinePowerStatePoweredOff); err != nil {
		return sdk.WrapError(err, "cannot wait for power state result")
	}

	return sdk.WithStack(ctxTo.Err())
}

func (c *vSphereClient) RenameVirtualMachine(ctx context.Context, vm *object.VirtualMachine, newName string) error {
	ctxTo, cancel := context.WithTimeout(ctx, c.requestTimeout)
	defer cancel()
	log.Debug(ctx, "renaming virtual machine %q to %q...", vm.Name(), newName)

	task, errR := vm.Rename(ctxTo, newName)
	if errR != nil {
		return sdk.WrapError(errR, "unable to rename model %s", newName)
	}

	ctxTo, cancel = context.WithTimeout(ctx, time.Minute)
	defer cancel()
	if _, err := task.WaitForResult(ctxTo, nil); err != nil {
		return sdk.WrapError(err, "error on waiting result for vm renaming %q to %q", vm.String(), newName)
	}

	vm2, err := c.LoadVirtualMachine(ctx, newName)
	if err != nil {
		return sdk.WrapError(err, "unable to reload VM %q", newName)
	}

	*vm = *vm2

	return nil
}

func (c *vSphereClient) MarkVirtualMachineAsTemplate(ctx context.Context, vm *object.VirtualMachine) error {
	ctxTo, cancel := context.WithTimeout(ctx, c.requestTimeout)
	defer cancel()

	if err := vm.MarkAsTemplate(ctxTo); err != nil {
		return sdk.WrapError(err, "unable to mark vm as template")
	}

	return sdk.WithStack(ctxTo.Err())
}

func (c *vSphereClient) WaitForVirtualMachineShutdown(ctx context.Context, vm *object.VirtualMachine) error {
	ctxTo, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	log.Debug(ctx, "waiting virtual machine %q to be powered off...", vm.Name())
	if err := vm.WaitForPowerState(ctxTo, types.VirtualMachinePowerStatePoweredOff); err != nil {
		return sdk.WrapError(err, "error while waiting for power state off")
	}

	return sdk.WithStack(ctxTo.Err())
}

func (c *vSphereClient) WaitForVirtualMachineIP(ctx context.Context, vm *object.VirtualMachine, IPAddress *string, vmName string) error {
	ctxIP, cancelIP := context.WithTimeout(ctx, 3*time.Minute)
	defer cancelIP()

	var ip string

	if IPAddress != nil && *IPAddress != "" {
		log.Debug(ctx, "waiting virtual machine %q got expected IP address: %v)", vmName, *IPAddress)
	}

	for ctxIP.Err() == nil {
		var err error
		ip, err = vm.WaitForIP(ctxIP, true)
		if err != nil {
			return sdk.WrapError(err, "cannot get an ip")
		}

		if IPAddress != nil && *IPAddress != "" {
			if ip == *IPAddress {
				break
			}
			continue
		}
		break
	}

	if ctxIP.Err() != nil {
		return sdk.WithStack(ctxIP.Err())
	}

	log.Info(ctx, "virtual machine %q (%q) has IP %q", vmName, vm.String(), ip)

	return nil
}

func (c *vSphereClient) GetVirtualMachinePowerState(ctx context.Context, vm *object.VirtualMachine) (types.VirtualMachinePowerState, error) {
	ctxTo, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	if ps, err := vm.PowerState(ctxTo); err != nil {
		return ps, sdk.WrapError(err, "error while getting vm powerstate")
	}

	return types.VirtualMachinePowerState(""), sdk.WithStack(ctxTo.Err())
}
