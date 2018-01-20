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
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/simulator/esx"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

func TestResourcePool(t *testing.T) {
	ctx := context.Background()

	m := &Model{
		ServiceContent: esx.ServiceContent,
		RootFolder:     esx.RootFolder,
	}

	err := m.Create()
	if err != nil {
		t.Fatal(err)
	}

	c := m.Service.client

	finder := find.NewFinder(c, false)
	finder.SetDatacenter(object.NewDatacenter(c, esx.Datacenter.Reference()))

	spec := NewResourceConfigSpec()

	parent := object.NewResourcePool(c, esx.ResourcePool.Self)

	// can't destroy a root pool
	task, err := parent.Destroy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = task.Wait(ctx); err == nil {
		t.Fatal("expected error destroying a root pool")
	}

	// create a child pool
	childName := uuid.New().String()

	child, err := parent.Create(ctx, childName, spec)
	if err != nil {
		t.Fatal(err)
	}

	if child.Reference() == esx.ResourcePool.Self {
		t.Error("expected new pool Self reference")
	}

	_, err = parent.Create(ctx, childName, spec)
	if err == nil {
		t.Error("expected error")
	}

	// create a grandchild pool
	grandChildName := uuid.New().String()
	_, err = child.Create(ctx, grandChildName, spec)
	if err != nil {
		t.Fatal(err)
	}

	// create sibling (of the grand child) pool
	siblingName := uuid.New().String()
	_, err = child.Create(ctx, siblingName, spec)
	if err != nil {
		t.Fatal(err)
	}

	// finder should return the 2 grand children
	pools, err := finder.ResourcePoolList(ctx, "*/Resources/"+childName+"/*")
	if err != nil {
		t.Fatal(err)
	}
	if len(pools) != 2 {
		t.Fatalf("len(pools) == %d", len(pools))
	}

	// destroy the child
	task, err = child.Destroy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = task.Wait(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// finder should error not found after destroying the child
	_, err = finder.ResourcePoolList(ctx, "*/Resources/"+childName+"/*")
	if err == nil {
		t.Fatal("expected not found error")
	}

	// since the child was destroyed, grand child pools should now be children of the root pool
	pools, err = finder.ResourcePoolList(ctx, "*/Resources/*")
	if err != nil {
		t.Fatal(err)
	}

	if len(pools) != 2 {
		t.Fatalf("len(pools) == %d", len(pools))
	}
}

func TestCreateVAppESX(t *testing.T) {
	ctx := context.Background()

	m := ESX()
	m.Datastore = 0
	m.Machine = 0

	err := m.Create()
	if err != nil {
		t.Fatal(err)
	}

	c := m.Service.client

	parent := object.NewResourcePool(c, esx.ResourcePool.Self)

	rspec := NewResourceConfigSpec()
	vspec := NewVAppConfigSpec()

	_, err = parent.CreateVApp(ctx, "myapp", rspec, vspec, nil)
	if err == nil {
		t.Fatal("expected error")
	}

	fault := soap.ToSoapFault(err).Detail.Fault

	if reflect.TypeOf(fault) != reflect.TypeOf(&types.MethodDisabled{}) {
		t.Errorf("fault=%#v", fault)
	}
}

func TestCreateVAppVPX(t *testing.T) {
	ctx := context.Background()

	m := VPX()

	err := m.Create()
	if err != nil {
		t.Fatal(err)
	}

	defer m.Remove()

	c := m.Service.client

	parent := object.NewResourcePool(c, Map.Any("ResourcePool").Reference())

	rspec := NewResourceConfigSpec()
	vspec := NewVAppConfigSpec()

	vapp, err := parent.CreateVApp(ctx, "myapp", rspec, vspec, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = parent.CreateVApp(ctx, "myapp", rspec, vspec, nil)
	if err == nil {
		t.Error("expected error")
	}

	spec := types.VirtualMachineConfigSpec{
		GuestId: string(types.VirtualMachineGuestOsIdentifierOtherGuest),
		Files: &types.VirtualMachineFileInfo{
			VmPathName: "[LocalDS_0]",
		},
	}

	for _, fail := range []bool{true, false} {
		task, cerr := vapp.CreateChildVM(ctx, spec, nil)
		if cerr != nil {
			t.Fatal(err)
		}

		cerr = task.Wait(ctx)
		if fail {
			if cerr == nil {
				t.Error("expected error")
			}
		} else {
			if cerr != nil {
				t.Error(err)
			}
		}

		spec.Name = "test"
	}

	si := object.NewSearchIndex(c)
	vm, err := si.FindChild(ctx, vapp, spec.Name)
	if err != nil {
		t.Fatal(err)
	}

	if vm == nil {
		t.Errorf("FindChild(%s)==nil", spec.Name)
	}

	task, err := vapp.Destroy(ctx)
	if err != nil {
		t.Fatal(err)
	}

	err = task.Wait(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
