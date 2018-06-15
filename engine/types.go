package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fatih/structs"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/hatchery/kubernetes"
	"github.com/ovh/cds/engine/hatchery/local"
	"github.com/ovh/cds/engine/hatchery/marathon"
	"github.com/ovh/cds/engine/hatchery/openstack"
	"github.com/ovh/cds/engine/hatchery/swarm"
	"github.com/ovh/cds/engine/hatchery/vsphere"
	"github.com/ovh/cds/engine/hooks"
	"github.com/ovh/cds/engine/migrateservice"
	"github.com/ovh/cds/engine/repositories"
	"github.com/ovh/cds/engine/vcs"
)

// Configuration contains CDS Configuration and toml description
type Configuration struct {
	Log struct {
		Level   string `toml:"level" default:"warning" comment:"Log Level: debug, info, warning, notice, critical"`
		Graylog struct {
			Host       string `toml:"host" comment:"Example: thot.ovh.com"`
			Port       int    `toml:"port" comment:"Example: 12202"`
			Protocol   string `toml:"protocol" default:"tcp" comment:"tcp or udp"`
			ExtraKey   string `toml:"extraKey" comment:"Example: X-OVH-TOKEN. You can use many keys: aaa,bbb"`
			ExtraValue string `toml:"extraValue" comment:"value for extraKey field. For many keys: valueaaa,valuebbb"`
		} `toml:"graylog"`
	} `toml:"log" comment:"#####################\n CDS Logs Settings \n####################"`
	Debug struct {
		Enable         bool   `toml:"enable" default:"false" comment:"allow debugging with gops"`
		RemoteDebugURL string `toml:"remoteDebugURL" comment:"start a gops agent on specified URL. Ex: localhost:9999"`
	} `toml:"debug" comment:"#####################\n Debug with gops \n####################"`
	API      api.Configuration `toml:"api" comment:"#####################\n API Configuration \n####################"`
	Hatchery struct {
		Local      local.HatcheryConfiguration      `toml:"local" comment:"Hatchery Local."`
		Kubernetes kubernetes.HatcheryConfiguration `toml:"kubernetes" comment:"Hatchery Kubernetes."`
		Marathon   marathon.HatcheryConfiguration   `toml:"marathon" comment:"Hatchery Marathon."`
		Openstack  openstack.HatcheryConfiguration  `toml:"openstack" comment:"Hatchery OpenStack. Doc: https://ovh.github.io/cds/advanced/advanced.hatcheries.openstack/"`
		Swarm      swarm.HatcheryConfiguration      `toml:"swarm" comment:"Hatchery Swarm. Doc: https://ovh.github.io/cds/advanced/advanced.hatcheries.swarm/"`
		VSphere    vsphere.HatcheryConfiguration    `toml:"vsphere" comment:"Hatchery VShpere. Doc: https://ovh.github.io/cds/advanced/advanced.hatcheries.vsphere/"`
	} `toml:"hatchery"`
	Hooks           hooks.Configuration          `toml:"hooks" comment:"######################\n CDS Hooks Settings \n######################"`
	VCS             vcs.Configuration            `toml:"vcs" comment:"######################\n CDS VCS Settings \n######################"`
	Repositories    repositories.Configuration   `toml:"repositories" comment:"######################\n CDS Repositories Settings \n######################"`
	DatabaseMigrate migrateservice.Configuration `toml:"databaseMigrate" comment:"######################\n CDS DB Migrate Service Settings \n######################"`
}

// AsEnvVariables returns the object attributes as env variables. It used for configuration structs
func AsEnvVariables(o interface{}, prefix string, skipCommented bool) map[string]string {
	r := map[string]string{}
	prefix = strings.ToUpper(prefix)
	delim := "_"
	if prefix == "" {
		delim = ""
	}
	fields := structs.Fields(o)
	for _, f := range fields {
		if skipCommented {
			if commented, _ := strconv.ParseBool(f.Tag("commented")); commented {
				continue
			}
		}
		if structs.IsStruct(f.Value()) {
			rf := AsEnvVariables(f.Value(), prefix+delim+f.Name(), skipCommented)
			for k, v := range rf {
				r[k] = v
			}
		} else {
			r[prefix+"_"+strings.ToUpper(f.Name())] = fmt.Sprintf("%v", f.Value())
		}
	}
	return r
}
