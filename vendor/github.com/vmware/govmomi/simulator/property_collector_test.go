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

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/simulator/esx"
	"github.com/vmware/govmomi/simulator/vpx"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

func TestRetrieveProperties(t *testing.T) {
	configs := []struct {
		folder  mo.Folder
		content types.ServiceContent
		dc      *types.ManagedObjectReference
	}{
		{esx.RootFolder, esx.ServiceContent, &esx.Datacenter.Self},
		{vpx.RootFolder, vpx.ServiceContent, nil},
	}

	for _, config := range configs {
		s := New(NewServiceInstance(config.content, config.folder))

		ts := s.NewServer()
		defer ts.Close()

		ctx := context.Background()

		client, err := govmomi.NewClient(ctx, ts.URL, true)
		if err != nil {
			t.Fatal(err)
		}

		if config.dc == nil {
			dc, cerr := object.NewRootFolder(client.Client).CreateDatacenter(ctx, "dc1")
			if cerr != nil {
				t.Fatal(cerr)
			}
			ref := dc.Reference()
			config.dc = &ref
		}

		// Retrieve a specific property
		f := mo.Folder{}
		err = client.RetrieveOne(ctx, config.content.RootFolder, []string{"name"}, &f)
		if err != nil {
			t.Fatal(err)
		}

		if f.Name != config.folder.Name {
			t.Fail()
		}

		// Retrieve all properties
		f = mo.Folder{}
		err = client.RetrieveOne(ctx, config.content.RootFolder, nil, &f)
		if err != nil {
			t.Fatal(err)
		}

		if f.Name != config.folder.Name {
			t.Fatalf("'%s' vs '%s'", f.Name, config.folder.Name)
		}

		// Retrieve an ArrayOf property
		f = mo.Folder{}
		err = client.RetrieveOne(ctx, config.content.RootFolder, []string{"childEntity"}, &f)
		if err != nil {
			t.Fatal(err)
		}

		if len(f.ChildEntity) != 1 {
			t.Fail()
		}

		es, err := mo.Ancestors(ctx, client.Client, config.content.PropertyCollector, config.content.RootFolder)
		if err != nil {
			t.Fatal(err)
		}

		if len(es) != 1 {
			t.Fail()
		}

		finder := find.NewFinder(client.Client, false)
		dc, err := finder.DatacenterOrDefault(ctx, "")
		if err != nil {
			t.Fatal(err)
		}

		if dc.Reference() != *config.dc {
			t.Fail()
		}

		finder.SetDatacenter(dc)

		es, err = mo.Ancestors(ctx, client.Client, config.content.PropertyCollector, dc.Reference())
		if err != nil {
			t.Fatal(err)
		}

		expect := map[string]types.ManagedObjectReference{
			"Folder":     config.folder.Reference(),
			"Datacenter": dc.Reference(),
		}

		if len(es) != len(expect) {
			t.Fail()
		}

		for _, e := range es {
			ref := e.Reference()
			if r, ok := expect[ref.Type]; ok {
				if r != ref {
					t.Errorf("%#v vs %#v", r, ref)
				}
			} else {
				t.Errorf("unexpected object %#v", e.Reference())
			}
		}

		// finder tests
		ls, err := finder.ManagedObjectListChildren(ctx, ".")
		if err != nil {
			t.Error(err)
		}

		folders, err := dc.Folders(ctx)
		if err != nil {
			t.Fatal(err)
		}

		// Validated name properties are recursively retrieved for the datacenter and its folder children
		ipaths := []string{
			folders.VmFolder.InventoryPath,
			folders.HostFolder.InventoryPath,
			folders.DatastoreFolder.InventoryPath,
			folders.NetworkFolder.InventoryPath,
		}

		var lpaths []string
		for _, p := range ls {
			lpaths = append(lpaths, p.Path)
		}

		if !reflect.DeepEqual(ipaths, lpaths) {
			t.Errorf("%#v != %#v\n", ipaths, lpaths)
		}

		// We have no VMs, expect NotFoundError
		_, err = finder.VirtualMachineList(ctx, "*")
		if err == nil {
			t.Error("expected error")
		} else {
			if _, ok := err.(*find.NotFoundError); !ok {
				t.Error(err)
			}
		}

		// Retrieve a missing property
		mdc := mo.Datacenter{}
		err = client.RetrieveOne(ctx, dc.Reference(), []string{"enoent"}, &mdc)
		if err == nil {
			t.Error("expected error")
		} else {
			switch fault := soap.ToVimFault(err).(type) {
			case *types.InvalidProperty:
				// ok
			default:
				t.Errorf("unexpected fault: %#v", fault)
			}
		}

		// Retrieve a nested property
		Map.Get(dc.Reference()).(*mo.Datacenter).Configuration.DefaultHardwareVersionKey = "foo"
		mdc = mo.Datacenter{}
		err = client.RetrieveOne(ctx, dc.Reference(), []string{"configuration.defaultHardwareVersionKey"}, &mdc)
		if err != nil {
			t.Fatal(err)
		}
		if mdc.Configuration.DefaultHardwareVersionKey != "foo" {
			t.Fail()
		}

		// Retrieve a missing nested property
		mdc = mo.Datacenter{}
		err = client.RetrieveOne(ctx, dc.Reference(), []string{"configuration.enoent"}, &mdc)
		if err == nil {
			t.Error("expected error")
		} else {
			switch fault := soap.ToVimFault(err).(type) {
			case *types.InvalidProperty:
				// ok
			default:
				t.Errorf("unexpected fault: %#v", fault)
			}
		}

		// Retrieve an empty property
		err = client.RetrieveOne(ctx, dc.Reference(), []string{""}, &mdc)
		if err != nil {
			t.Error(err)
		}

		// Expect ManagedObjectNotFoundError
		Map.Remove(dc.Reference())
		err = client.RetrieveOne(ctx, dc.Reference(), []string{"name"}, &mdc)
		if err == nil {
			t.Fatal("expected error")
		}
	}
}

func TestWaitForUpdates(t *testing.T) {
	folder := esx.RootFolder
	s := New(NewServiceInstance(esx.ServiceContent, folder))

	ts := s.NewServer()
	defer ts.Close()

	ctx := context.Background()

	c, err := govmomi.NewClient(ctx, ts.URL, true)
	if err != nil {
		t.Fatal(err)
	}

	cb := func(once bool) func([]types.PropertyChange) bool {
		return func(pc []types.PropertyChange) bool {
			if len(pc) != 1 {
				t.Fail()
			}

			c := pc[0]
			if c.Op != types.PropertyChangeOpAssign {
				t.Fail()
			}
			if c.Name != "name" {
				t.Fail()
			}
			if c.Val.(string) != folder.Name {
				t.Fail()
			}

			return once
		}
	}

	pc := property.DefaultCollector(c.Client)
	props := []string{"name"}

	err = property.Wait(ctx, pc, folder.Reference(), props, cb(true))
	if err != nil {
		t.Error(err)
	}

	// incremental updates not yet suppported
	err = property.Wait(ctx, pc, folder.Reference(), props, cb(false))
	if err == nil {
		t.Error("expected error")
	}

	// test object not found
	Map.Remove(folder.Reference())

	err = property.Wait(ctx, pc, folder.Reference(), props, cb(true))
	if err == nil {
		t.Error("expected error")
	}

	// test CancelWaitForUpdates
	p, err := pc.Create(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// test the deprecated WaitForUpdates methods
	_, err = methods.WaitForUpdates(ctx, c.Client, &types.WaitForUpdates{This: p.Reference()})
	if err != nil {
		t.Fatal(err)
	}

	_, err = methods.CancelWaitForUpdates(ctx, c.Client, &types.CancelWaitForUpdates{This: p.Reference()})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCollectInterfaceType(t *testing.T) {
	// test that we properly collect an interface type (types.BaseVirtualDevice in this case)
	var config types.VirtualMachineConfigInfo
	config.Hardware.Device = append(config.Hardware.Device, new(types.VirtualFloppy))

	_, err := fieldValue(reflect.ValueOf(&config), "hardware.device")
	if err != nil {
		t.Fatal(err)
	}
}

func TestExtractEmbeddedField(t *testing.T) {
	type YourResourcePool struct {
		mo.ResourcePool
	}

	type MyResourcePool struct {
		YourResourcePool
	}

	x := new(MyResourcePool)

	Map.Put(x)

	obj, ok := getObject(x.Reference())
	if !ok {
		t.Error("expected obj")
	}

	if obj.Type() != reflect.ValueOf(new(mo.ResourcePool)).Elem().Type() {
		t.Errorf("unexpected type=%s", obj.Type().Name())
	}

	// satisfies the mo.Reference interface, but does not embed a type from the "mo" package
	type NoMo struct {
		types.ManagedObjectReference

		Self types.ManagedObjectReference
	}

	n := new(NoMo)
	n.ManagedObjectReference = types.ManagedObjectReference{Type: "NoMo", Value: "no-mo"}
	Map.Put(n)

	_, ok = getObject(n.Reference())
	if ok {
		t.Error("expected not ok")
	}
}

func TestPropertyCollectorFold(t *testing.T) {
	ctx := context.Background()

	m := VPX()

	defer m.Remove()

	err := m.Create()
	if err != nil {
		t.Fatal(err)
	}

	s := m.Service.NewServer()
	defer s.Close()

	client, err := govmomi.NewClient(ctx, s.URL, true)
	if err != nil {
		t.Fatal(err)
	}

	cluster := Map.Any("ClusterComputeResource")
	compute := Map.Any("ComputeResource")

	// Test that we fold duplicate properties (rbvmomi depends on this)
	var content []types.ObjectContent
	err = client.Retrieve(ctx, []types.ManagedObjectReference{cluster.Reference()}, []string{"name", "name"}, &content)
	if err != nil {
		t.Fatal(err)
	}
	if len(content) != 1 {
		t.Fatalf("len(content)=%d", len(content))
	}
	if len(content[0].PropSet) != 1 {
		t.Fatalf("len(PropSet)=%d", len(content[0].PropSet))
	}

	// Test that we fold embedded type properties (rbvmomi depends on this)
	req := types.RetrieveProperties{
		SpecSet: []types.PropertyFilterSpec{
			{
				PropSet: []types.PropertySpec{
					{
						DynamicData: types.DynamicData{},
						Type:        "ComputeResource",
						PathSet:     []string{"name"},
					},
					{
						DynamicData: types.DynamicData{},
						Type:        "ClusterComputeResource",
						PathSet:     []string{"name"},
					},
				},
				ObjectSet: []types.ObjectSpec{
					{
						Obj: compute.Reference(),
					},
					{
						Obj: cluster.Reference(),
					},
				},
			},
		},
	}

	pc := client.PropertyCollector()

	res, err := pc.RetrieveProperties(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	content = res.Returnval

	if len(content) != 2 {
		t.Fatalf("len(content)=%d", len(content))
	}

	for _, oc := range content {
		if len(oc.PropSet) != 1 {
			t.Errorf("%s len(PropSet)=%d", oc.Obj, len(oc.PropSet))
		}
	}
}

func TestPropertyCollectorInvalidSpecName(t *testing.T) {
	obj := Map.Put(new(Folder))
	obj.(*Folder).putChild(new(Folder))

	pc := &PropertyCollector{}

	req := types.RetrievePropertiesEx{
		SpecSet: []types.PropertyFilterSpec{
			{
				PropSet: []types.PropertySpec{
					{
						Type:    obj.Reference().Type,
						PathSet: []string{"name"},
					},
				},
				ObjectSet: []types.ObjectSpec{
					{
						Obj: obj.Reference(),
						SelectSet: []types.BaseSelectionSpec{
							&types.TraversalSpec{
								Type: "Folder",
								Path: "childEntity",
								SelectSet: []types.BaseSelectionSpec{
									&types.SelectionSpec{
										Name: "enoent",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := pc.collect(&req)
	if err == nil {
		t.Fatal("expected error")
	}

	if _, ok := err.(*types.InvalidArgument); !ok {
		t.Errorf("unexpected fault: %#v", err)
	}
}

func TestPropertyCollectorRecursiveSelectSet(t *testing.T) {
	ctx := context.Background()

	m := VPX()

	defer m.Remove()

	err := m.Create()
	if err != nil {
		t.Fatal(err)
	}

	s := m.Service.NewServer()
	defer s.Close()

	client, err := govmomi.NewClient(ctx, s.URL, true)
	if err != nil {
		t.Fatal(err)
	}

	// Capture of PowerCLI's Get-VM spec
	req := types.RetrieveProperties{
		SpecSet: []types.PropertyFilterSpec{
			{
				PropSet: []types.PropertySpec{
					{
						Type: "VirtualMachine",
						PathSet: []string{
							"runtime.host",
							"parent",
							"resourcePool",
							"resourcePool",
							"datastore",
							"config.swapPlacement",
							"config.version",
							"config.instanceUuid",
							"config.guestId",
							"config.annotation",
							"summary.storage.committed",
							"summary.storage.uncommitted",
							"summary.storage.committed",
							"config.template",
						},
					},
				},
				ObjectSet: []types.ObjectSpec{
					{
						Obj:  client.ServiceContent.RootFolder,
						Skip: types.NewBool(false),
						SelectSet: []types.BaseSelectionSpec{
							&types.TraversalSpec{
								SelectionSpec: types.SelectionSpec{
									Name: "traverseFolders",
								},
								Type: "Folder",
								Path: "childEntity",
								Skip: types.NewBool(true),
								SelectSet: []types.BaseSelectionSpec{
									&types.TraversalSpec{
										Type: "HostSystem",
										Path: "vm",
										Skip: types.NewBool(false),
									},
									&types.TraversalSpec{
										Type: "ComputeResource",
										Path: "host",
										Skip: types.NewBool(true),
										SelectSet: []types.BaseSelectionSpec{
											&types.TraversalSpec{
												Type: "HostSystem",
												Path: "vm",
												Skip: types.NewBool(false),
											},
										},
									},
									&types.TraversalSpec{
										Type: "Datacenter",
										Path: "hostFolder",
										Skip: types.NewBool(true),
										SelectSet: []types.BaseSelectionSpec{
											&types.SelectionSpec{
												Name: "traverseFolders",
											},
										},
									},
									&types.SelectionSpec{
										Name: "traverseFolders",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	pc := client.PropertyCollector()

	res, err := pc.RetrieveProperties(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	content := res.Returnval

	count := m.Count()

	if len(content) != count.Machine {
		t.Fatalf("len(content)=%d", len(content))
	}
}

func TestPropertyCollectorSelectionSpec(t *testing.T) {
	ctx := context.Background()

	m := VPX()

	defer m.Remove()

	err := m.Create()
	if err != nil {
		t.Fatal(err)
	}

	s := m.Service.NewServer()
	defer s.Close()

	client, err := govmomi.NewClient(ctx, s.URL, true)
	if err != nil {
		t.Fatal(err)
	}

	// Capture of PowerCLI's Start-VM spec
	// Differs from the norm in that:
	// 1) Named SelectionSpec before TraversalSpec is defined
	// 2) Skip=false for all mo types, but PropSet only has Type VirtualMachine
	req := types.RetrieveProperties{
		SpecSet: []types.PropertyFilterSpec{
			{
				PropSet: []types.PropertySpec{
					{
						Type:    "VirtualMachine",
						All:     types.NewBool(false),
						PathSet: []string{"name", "parent", "runtime.host", "config.template"},
					},
				},
				ObjectSet: []types.ObjectSpec{
					{
						Obj:  client.ServiceContent.RootFolder,
						Skip: types.NewBool(false),
						SelectSet: []types.BaseSelectionSpec{
							&types.TraversalSpec{
								SelectionSpec: types.SelectionSpec{
									Name: "folderTraversalSpec",
								},
								Type: "Folder",
								Path: "childEntity",
								Skip: types.NewBool(false),
								SelectSet: []types.BaseSelectionSpec{
									&types.SelectionSpec{
										Name: "computeResourceRpTraversalSpec",
									},
									&types.SelectionSpec{
										Name: "computeResourceHostTraversalSpec",
									},
									&types.SelectionSpec{
										Name: "folderTraversalSpec",
									},
									&types.SelectionSpec{
										Name: "datacenterHostTraversalSpec",
									},
									&types.SelectionSpec{
										Name: "hostVmTraversalSpec",
									},
									&types.SelectionSpec{
										Name: "resourcePoolVmTraversalSpec",
									},
									&types.SelectionSpec{
										Name: "hostRpTraversalSpec",
									},
									&types.SelectionSpec{
										Name: "datacenterVmTraversalSpec",
									},
									&types.SelectionSpec{
										Name: "datastoreVmTraversalSpec",
									},
									&types.SelectionSpec{
										Name: "datacenterDatastoreTraversalSpec",
									},
									&types.SelectionSpec{
										Name: "vappTraversalSpec",
									},
									&types.SelectionSpec{
										Name: "datacenterNetworkTraversalSpec",
									},
								},
							},
							&types.TraversalSpec{
								SelectionSpec: types.SelectionSpec{
									Name: "computeResourceHostTraversalSpec",
								},
								Type: "ComputeResource",
								Path: "host",
								Skip: types.NewBool(false),
								SelectSet: []types.BaseSelectionSpec{
									&types.SelectionSpec{
										Name: "hostVmTraversalSpec",
									},
								},
							},
							&types.TraversalSpec{
								SelectionSpec: types.SelectionSpec{
									Name: "computeResourceRpTraversalSpec",
								},
								Type: "ComputeResource",
								Path: "resourcePool",
								Skip: types.NewBool(false),
								SelectSet: []types.BaseSelectionSpec{
									&types.SelectionSpec{
										Name: "resourcePoolTraversalSpec",
									},
									&types.SelectionSpec{
										Name: "vappTraversalSpec",
									},
								},
							},
							&types.TraversalSpec{
								SelectionSpec: types.SelectionSpec{
									Name: "datacenterHostTraversalSpec",
								},
								Type: "Datacenter",
								Path: "hostFolder",
								Skip: types.NewBool(false),
								SelectSet: []types.BaseSelectionSpec{
									&types.SelectionSpec{
										Name: "folderTraversalSpec",
									},
								},
							},
							&types.TraversalSpec{
								SelectionSpec: types.SelectionSpec{
									Name: "resourcePoolTraversalSpec",
								},
								Type: "ResourcePool",
								Path: "resourcePool",
								Skip: types.NewBool(false),
								SelectSet: []types.BaseSelectionSpec{
									&types.SelectionSpec{
										Name: "resourcePoolTraversalSpec",
									},
									&types.SelectionSpec{
										Name: "resourcePoolVmTraversalSpec",
									},
								},
							},
							&types.TraversalSpec{
								SelectionSpec: types.SelectionSpec{
									Name: "hostVmTraversalSpec",
								},
								Type: "HostSystem",
								Path: "vm",
								Skip: types.NewBool(false),
								SelectSet: []types.BaseSelectionSpec{
									&types.SelectionSpec{
										Name: "folderTraversalSpec",
									},
								},
							},
							&types.TraversalSpec{
								SelectionSpec: types.SelectionSpec{
									Name: "datacenterVmTraversalSpec",
								},
								Type: "Datacenter",
								Path: "vmFolder",
								Skip: types.NewBool(false),
								SelectSet: []types.BaseSelectionSpec{
									&types.SelectionSpec{
										Name: "folderTraversalSpec",
									},
								},
							},
							&types.TraversalSpec{
								SelectionSpec: types.SelectionSpec{
									Name: "resourcePoolVmTraversalSpec",
								},
								Type: "ResourcePool",
								Path: "vm",
								Skip: types.NewBool(false),
							},
							&types.TraversalSpec{
								SelectionSpec: types.SelectionSpec{
									Name: "datastoreVmTraversalSpec",
								},
								Type: "Datastore",
								Path: "vm",
								Skip: types.NewBool(false),
							},
							&types.TraversalSpec{
								SelectionSpec: types.SelectionSpec{
									Name: "datacenterDatastoreTraversalSpec",
								},
								Type: "Datacenter",
								Path: "datastoreFolder",
								Skip: types.NewBool(false),
								SelectSet: []types.BaseSelectionSpec{
									&types.SelectionSpec{
										Name: "folderTraversalSpec",
									},
								},
							},
							&types.TraversalSpec{
								SelectionSpec: types.SelectionSpec{
									Name: "vappTraversalSpec",
								},
								Type: "VirtualApp",
								Path: "resourcePool",
								Skip: types.NewBool(false),
								SelectSet: []types.BaseSelectionSpec{
									&types.SelectionSpec{
										Name: "vappTraversalSpec",
									},
									&types.SelectionSpec{
										Name: "resourcePoolTraversalSpec",
									},
									&types.SelectionSpec{
										Name: "resourcePoolVmTraversalSpec",
									},
								},
							},
							&types.TraversalSpec{
								SelectionSpec: types.SelectionSpec{
									Name: "vappVmTraversalSpec",
								},
								Type: "VirtualApp",
								Path: "vm",
								Skip: types.NewBool(false),
							},
							&types.TraversalSpec{
								SelectionSpec: types.SelectionSpec{
									Name: "distributedSwitchHostTraversalSpec",
								},
								Type: "DistributedVirtualSwitch",
								Path: "summary.hostMember",
								Skip: types.NewBool(false),
							},
							&types.TraversalSpec{
								SelectionSpec: types.SelectionSpec{
									Name: "distributedSwitchVmTraversalSpec",
								},
								Type: "DistributedVirtualSwitch",
								Path: "summary.vm",
								Skip: types.NewBool(false),
							},
							&types.TraversalSpec{
								SelectionSpec: types.SelectionSpec{
									Name: "datacenterNetworkTraversalSpec",
								},
								Type: "Datacenter",
								Path: "networkFolder",
								Skip: types.NewBool(false),
								SelectSet: []types.BaseSelectionSpec{
									&types.SelectionSpec{
										Name: "folderTraversalSpec",
									},
								},
							},
							&types.TraversalSpec{
								SelectionSpec: types.SelectionSpec{
									Name: "hostRpTraversalSpec",
								},
								Type: "HostSystem",
								Path: "parent",
								Skip: types.NewBool(false),
								SelectSet: []types.BaseSelectionSpec{
									&types.SelectionSpec{
										Name: "computeResourceRpTraversalSpec",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	pc := client.PropertyCollector()

	res, err := pc.RetrieveProperties(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	content := res.Returnval
	count := m.Count()

	if len(content) != count.Machine {
		t.Fatalf("len(content)=%d", len(content))
	}
}
