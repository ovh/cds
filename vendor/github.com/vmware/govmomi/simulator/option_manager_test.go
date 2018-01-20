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

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/simulator/esx"
)

func TestOptionManagerESX(t *testing.T) {
	ctx := context.Background()

	model := ESX()
	model.Datastore = 0
	model.Machine = 0

	err := model.Create()
	if err != nil {
		t.Fatal(err)
	}

	c := model.Service.client

	m := object.NewOptionManager(c, *c.ServiceContent.Setting)
	_, err = m.Query(ctx, "config.vpxd.")
	if err == nil {
		t.Error("expected error")
	}

	host := object.NewHostSystem(c, esx.HostSystem.Reference())
	m, err = host.ConfigManager().OptionManager(ctx)
	if err != nil {
		t.Fatal(err)
	}

	res, err := m.Query(ctx, "Config.HostAgent.")
	if err != nil {
		t.Error(err)
	}

	if len(res) == 0 {
		t.Error("no results")
	}
}

func TestOptionManagerVPX(t *testing.T) {
	ctx := context.Background()

	model := VPX()
	model.Datastore = 0
	model.Machine = 0

	err := model.Create()
	if err != nil {
		t.Fatal(err)
	}

	c := model.Service.client

	m := object.NewOptionManager(c, *c.ServiceContent.Setting)
	_, err = m.Query(ctx, "enoent")
	if err == nil {
		t.Error("expected error")
	}

	res, err := m.Query(ctx, "config.vpxd.")
	if err != nil {
		t.Error(err)
	}

	if len(res) == 0 {
		t.Error("no results")
	}
}
