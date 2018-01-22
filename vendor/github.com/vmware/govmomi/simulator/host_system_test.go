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
	"testing"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/simulator/esx"
)

func TestDefaultESX(t *testing.T) {
	s := New(NewServiceInstance(esx.ServiceContent, esx.RootFolder))

	ts := s.NewServer()
	defer ts.Close()

	ctx := context.Background()

	client, err := govmomi.NewClient(ctx, ts.URL, true)
	if err != nil {
		t.Fatal(err)
	}

	finder := find.NewFinder(client.Client, false)

	dc, err := finder.DatacenterOrDefault(ctx, "")
	if err != nil {
		t.Fatal(err)
	}

	finder.SetDatacenter(dc)

	host, err := finder.HostSystemOrDefault(ctx, "*")
	if err != nil {
		t.Fatal(err)
	}

	if host.Name() != esx.HostSystem.Summary.Config.Name {
		t.Fail()
	}

	pool, err := finder.ResourcePoolOrDefault(ctx, "*")
	if err != nil {
		t.Fatal(err)
	}

	if pool.Name() != "Resources" {
		t.Fail()
	}
}
