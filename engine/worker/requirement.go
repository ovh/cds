package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/plugin"
)

func checkRequirement(r sdk.Requirement) (bool, error) {
	switch r.Type {
	case sdk.BinaryRequirement:
		return checkBinaryRequirement(r)
	case sdk.HostnameRequirement:
		return checkHostnameRequirement(r)
	case sdk.ModelRequirement:
		return checkModelRequirement(r)
	case sdk.NetworkAccessRequirement:
		return checkNetworkAccessRequirement(r)
	case sdk.PluginRequirement:
		return checkPluginRequirement(r)
	case sdk.ServiceRequirement:
		return checkServiceRequirement(r)
	default:
		log.Printf("checkRequirement> Unknown type of requirement: %s\n", r.Type)
		return false, fmt.Errorf("unknown type of requirement %s", r.Type)
	}
}

func checkPluginRequirement(r sdk.Requirement) (bool, error) {
	if err := sdk.DownloadPlugin(r.Name, os.TempDir()); err != nil {
		return false, err
	}
	pluginBinary := path.Join(os.TempDir(), r.Name)
	if err := os.Chmod(pluginBinary, 0700); err != nil {
		return false, err
	}
	pluginClient := plugin.NewClient(r.Name, pluginBinary, "", "", false)
	defer pluginClient.Kill()

	_plugin, err := pluginClient.Instance()
	if err != nil {
		log.Printf("[WARNING] Error Checkin %s requirement : %s", r.Name, err)
		return false, err
	}
	log.Printf("[NOTICE] Plugin %s successfully started", _plugin.Name())

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

	_, err := exec.LookPath(r.Value)
	if err != nil {
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
	_, err := net.LookupIP(r.Name)
	if err != nil {
		return false, err
	}

	return true, nil
}
