package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/mem"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/plugin"
)

var requirementCheckFuncs = map[string]func(r sdk.Requirement) (bool, error){
	sdk.BinaryRequirement:        checkBinaryRequirement,
	sdk.HostnameRequirement:      checkHostnameRequirement,
	sdk.ModelRequirement:         checkModelRequirement,
	sdk.NetworkAccessRequirement: checkNetworkAccessRequirement,
	sdk.PluginRequirement:        checkPluginRequirement,
	sdk.ServiceRequirement:       checkServiceRequirement,
	sdk.MemoryRequirement:        checkMemoryRequirement,
}

func checkRequirement(r sdk.Requirement) (bool, error) {
	check := requirementCheckFuncs[r.Type]
	if check == nil {
		log.Warning("checkRequirement> Unknown type of requirement: %s\n", r.Type)
		log.Warning("checkRequirement> Support requirements are : %v", requirementCheckFuncs)
		return false, fmt.Errorf("unknown type of requirement %s", r.Type)
	}
	return check(r)
}

func checkPluginRequirement(r sdk.Requirement) (bool, error) {
	pluginBinary := path.Join(os.TempDir(), r.Name)

	if _, err := os.Stat(pluginBinary); os.IsNotExist(err) {
		//If the file doesn't exist. Download it.
		if err := sdk.DownloadPlugin(r.Name, os.TempDir()); err != nil {
			return false, err
		}
		if err := os.Chmod(pluginBinary, 0700); err != nil {
			return false, err
		}
	}

	pluginClient := plugin.NewClient(r.Name, pluginBinary, "", "", false)
	defer pluginClient.Kill()

	_plugin, err := pluginClient.Instance()
	if err != nil {
		log.Warning("[WARNING] Error Checkin %s requirement : %s", r.Name, err)
		return false, err
	}
	log.Warning("[NOTICE] Plugin %s successfully started", _plugin.Name())

	return true, nil
}

func checkHostnameRequirement(r sdk.Requirement) (bool, error) {
	h, err := os.Hostname()
	if err != nil {
		return false, err
	}

	if h == r.Value {
		return true, nil
	}

	return false, nil
}

func checkBinaryRequirement(r sdk.Requirement) (bool, error) {
	if _, err := exec.LookPath(r.Value); err != nil {
		// Return nil because the error contains 'Executable file not found', that's what we wanted
		return false, nil
	}

	return true, nil
}

func checkModelRequirement(r sdk.Requirement) (bool, error) {
	wm, err := sdk.GetWorkerModel(r.Value)
	if err != nil {
		return false, nil
	}

	if wm.ID == model {
		return true, nil
	}

	return false, nil
}

func checkNetworkAccessRequirement(r sdk.Requirement) (bool, error) {
	conn, err := net.DialTimeout("tcp", r.Value, 10*time.Second)
	if err != nil {
		return false, nil
	}
	conn.Close()

	return true, nil
}

func checkServiceRequirement(r sdk.Requirement) (bool, error) {
	if _, err := net.LookupIP(r.Name); err != nil {
		log.Warning("Error checking requirement : %s\n", err)
		return false, nil
	}

	return true, nil
}

func checkMemoryRequirement(r sdk.Requirement) (bool, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return false, err
	}
	totalMemory := v.Total

	neededMemory, err := strconv.ParseInt(r.Value, 10, 64)
	if err != nil {
		return false, err
	}
	//Assuming memory is in megabytes
	//If we have more than 90% of neededMemory, lets do it
	return int64(totalMemory) >= (neededMemory*1024*1024)*90/100, nil
}
