/*
Copyright (c) 2017 VMware, Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package simulator

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/simulator/esx"
	"github.com/vmware/govmomi/task"
	"github.com/vmware/govmomi/vim25/types"
)

func TestCreateVm(t *testing.T) {
	ctx := context.Background()

	for _, model := range []*Model{ESX(), VPX()} {
		defer model.Remove()
		err := model.Create()
		if err != nil {
			t.Fatal(err)
		}

		s := model.Service.NewServer()
		defer s.Close()

		c, err := govmomi.NewClient(ctx, s.URL, true)
		if err != nil {
			t.Fatal(err)
		}

		p := property.DefaultCollector(c.Client)

		finder := find.NewFinder(c.Client, false)

		dc, err := finder.DefaultDatacenter(ctx)
		if err != nil {
			t.Fatal(err)
		}

		finder.SetDatacenter(dc)

		folders, err := dc.Folders(ctx)
		if err != nil {
			t.Fatal(err)
		}

		ds, err := finder.DefaultDatastore(ctx)
		if err != nil {
			t.Fatal(err)
		}

		hosts, err := finder.HostSystemList(ctx, "*/*")
		if err != nil {
			t.Fatal(err)
		}

		nhosts := len(hosts)
		host := hosts[rand.Intn(nhosts)]
		pool, err := host.ResourcePool(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if nhosts == 1 {
			// test the default path against the ESX model
			host = nil
		}

		vmFolder := folders.VmFolder

		var vmx string

		spec := types.VirtualMachineConfigSpec{
			// Note: real ESX allows the VM to be created without a GuestId,
			// but will power on will fail.
			GuestId: string(types.VirtualMachineGuestOsIdentifierOtherGuest),
		}

		steps := []func(){
			func() {
				spec.Name = "test"
				vmx = fmt.Sprintf("%s/%s.vmx", spec.Name, spec.Name)
			},
			func() {
				spec.Files = &types.VirtualMachineFileInfo{
					VmPathName: fmt.Sprintf("[%s] %s", ds.Name(), vmx),
				}
			},
		}

		// expecting CreateVM to fail until all steps are taken
		for _, step := range steps {
			task, cerr := vmFolder.CreateVM(ctx, spec, pool, host)
			if cerr != nil {
				t.Fatal(err)
			}

			cerr = task.Wait(ctx)
			if cerr == nil {
				t.Error("expected error")
			}

			step()
		}

		task, err := vmFolder.CreateVM(ctx, spec, pool, host)
		if err != nil {
			t.Fatal(err)
		}

		info, err := task.WaitForResult(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}

		// Test that datastore files were created
		_, err = ds.Stat(ctx, vmx)
		if err != nil {
			t.Fatal(err)
		}

		vm := object.NewVirtualMachine(c.Client, info.Result.(types.ManagedObjectReference))

		name, err := vm.ObjectName(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if name != spec.Name {
			t.Errorf("name=%s", name)
		}

		_, err = vm.Device(ctx)
		if err != nil {
			t.Fatal(err)
		}

		recreate := func(context.Context) (*object.Task, error) {
			return vmFolder.CreateVM(ctx, spec, pool, nil)
		}

		ops := []struct {
			method func(context.Context) (*object.Task, error)
			state  types.VirtualMachinePowerState
			fail   bool
		}{
			// Powered off by default
			{nil, types.VirtualMachinePowerStatePoweredOff, false},
			// Create with same .vmx path should fail
			{recreate, "", true},
			// Off -> On  == ok
			{vm.PowerOn, types.VirtualMachinePowerStatePoweredOn, false},
			// On  -> On  == fail
			{vm.PowerOn, types.VirtualMachinePowerStatePoweredOn, true},
			// On  -> Off == ok
			{vm.PowerOff, types.VirtualMachinePowerStatePoweredOff, false},
			// Off -> Off == fail
			{vm.PowerOff, types.VirtualMachinePowerStatePoweredOff, true},
			// Off -> On  == ok
			{vm.PowerOn, types.VirtualMachinePowerStatePoweredOn, false},
			// Destroy == fail (power is On)
			{vm.Destroy, types.VirtualMachinePowerStatePoweredOn, true},
			// On  -> Off == ok
			{vm.PowerOff, types.VirtualMachinePowerStatePoweredOff, false},
			// Destroy == ok (power is Off)
			{vm.Destroy, "", false},
		}

		for i, op := range ops {
			if op.method != nil {
				task, err = op.method(ctx)
				if err != nil {
					t.Fatal(err)
				}

				err = task.Wait(ctx)
				if op.fail {
					if err == nil {
						t.Errorf("%d: expected error", i)
					}
				} else {
					if err != nil {
						t.Errorf("%d: %s", i, err)
					}
				}
			}

			if len(op.state) != 0 {
				state, err := vm.PowerState(ctx)
				if err != nil {
					t.Fatal(err)
				}

				if state != op.state {
					t.Errorf("state=%s", state)
				}

				err = property.Wait(ctx, p, vm.Reference(), []string{object.PropRuntimePowerState}, func(pc []types.PropertyChange) bool {
					for _, c := range pc {
						switch v := c.Val.(type) {
						case types.VirtualMachinePowerState:
							if v != op.state {
								t.Errorf("state=%s", v)
							}
						default:
							t.Errorf("unexpected type %T", v)
						}

					}
					return false
				})
			}
		}

		// Test that datastore files were removed
		_, err = ds.Stat(ctx, vmx)
		if err == nil {
			t.Error("expected error")
		}
	}
}

func TestReconfigVm(t *testing.T) {
	ctx := context.Background()

	m := ESX()
	defer m.Remove()
	err := m.Create()
	if err != nil {
		t.Fatal(err)
	}

	s := m.Service.NewServer()
	defer s.Close()

	c, err := govmomi.NewClient(ctx, s.URL, true)
	if err != nil {
		t.Fatal(err)
	}

	finder := find.NewFinder(c.Client, false)
	finder.SetDatacenter(object.NewDatacenter(c.Client, esx.Datacenter.Reference()))

	vms, err := finder.VirtualMachineList(ctx, "*")
	if err != nil {
		t.Fatal(err)
	}

	vm := vms[0]
	device, err := vm.Device(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// verify default device list
	_, err = device.FindIDEController("")
	if err != nil {
		t.Fatal(err)
	}

	// default list of devices + 1 NIC + 1 SCSI controller + 1 CDROM + 1 disk created by the Model
	mdevices := len(esx.VirtualDevice) + 4

	if len(device) != mdevices {
		t.Errorf("expected %d devices, got %d", mdevices, len(device))
	}

	d := device.FindByKey(esx.EthernetCard.Key)

	err = vm.AddDevice(ctx, d)
	if _, ok := err.(task.Error).Fault().(*types.InvalidDeviceSpec); !ok {
		t.Fatalf("err=%v", err)
	}

	err = vm.RemoveDevice(ctx, false, d)
	if err != nil {
		t.Fatal(err)
	}

	device, err = vm.Device(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(device) != mdevices-1 {
		t.Error("device list mismatch")
	}

	// cover the path where the simulator assigns a UnitNumber
	d.GetVirtualDevice().UnitNumber = nil
	// cover the path where the simulator assigns a Key
	d.GetVirtualDevice().Key = -1

	err = vm.AddDevice(ctx, d)
	if err != nil {
		t.Fatal(err)
	}

	device, err = vm.Device(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(device) != mdevices {
		t.Error("device list mismatch")
	}
}

func TestCreateVmWithDevices(t *testing.T) {
	ctx := context.Background()

	m := ESX()
	defer m.Remove()

	err := m.Create()
	if err != nil {
		t.Fatal(err)
	}

	s := m.Service.NewServer()
	defer s.Close()

	c := m.Service.client

	folder := object.NewFolder(c, esx.Datacenter.VmFolder)
	pool := object.NewResourcePool(c, esx.ResourcePool.Self)

	// different set of devices from Model.Create's
	var devices object.VirtualDeviceList
	ide, _ := devices.CreateIDEController()
	cdrom, _ := devices.CreateCdrom(ide.(*types.VirtualIDEController))

	devices = append(devices, ide, cdrom)
	create, _ := devices.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)

	spec := types.VirtualMachineConfigSpec{
		Name:         "foo",
		GuestId:      string(types.VirtualMachineGuestOsIdentifierOtherGuest),
		DeviceChange: create,
		Files: &types.VirtualMachineFileInfo{
			VmPathName: "[LocalDS_0] foo/foo.vmx",
		},
	}

	ctask, _ := folder.CreateVM(ctx, spec, pool, nil)
	info, err := ctask.WaitForResult(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	vm := Map.Get(info.Result.(types.ManagedObjectReference)).(*VirtualMachine)

	expect := len(esx.VirtualDevice) + len(devices)
	ndevice := len(vm.Config.Hardware.Device)

	if expect != ndevice {
		t.Errorf("expected %d, got %d", expect, ndevice)
	}
}

func TestShutdownGuest(t *testing.T) {
	// use the default vm for testing
	ctx := context.Background()

	for _, model := range []*Model{ESX(), VPX()} {
		defer model.Remove()
		err := model.Create()
		if err != nil {
			t.Fatal(err)
		}

		s := model.Service.NewServer()
		defer s.Close()

		c, err := govmomi.NewClient(ctx, s.URL, true)
		if err != nil {
			t.Fatal(err)
		}

		finder := find.NewFinder(c.Client, false)

		dc, err := finder.DefaultDatacenter(ctx)
		if err != nil {
			t.Fatal(err)
		}

		finder.SetDatacenter(dc)

		vms, err := finder.VirtualMachineList(ctx, "*")
		// use the default first vm for test
		if len(vms) > 0 {
			vmm := vms[0]
			// powon first
			task, err := vmm.PowerOn(ctx)
			if err != nil {
				t.Fatal(err)
			}

			err = task.Wait(ctx)
			if err != nil {
				t.Fatal(err)
			}

			// shutdown the vm
			err = vmm.ShutdownGuest(ctx)
			if err != nil {
				t.Fatal(err)
			}
			// state should be poweroff
			state, err := vmm.PowerState(ctx)
			if err != nil {
				t.Fatal(err)
			}

			if state != types.VirtualMachinePowerStatePoweredOff {
				t.Errorf("state=%s", state)
			}

			// shutdown a poweroff vm should fail
			err = vmm.ShutdownGuest(ctx)
			if err == nil {
				t.Error("expected error: InvalidPowerState")
			}
		}

	}
}
