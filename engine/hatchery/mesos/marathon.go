package mesos

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	"github.com/ovh/cds/sdk/hatchery"
)

// Application is the definition for an application in marathon
type Application struct {
	ID                    string            `json:"id,omitempty"`
	Cmd                   string            `json:"cmd,omitempty"`
	Args                  []string          `json:"args"`
	Constraints           [][]string        `json:"constraints"`
	CPUs                  float64           `json:"cpus,omitempty"`
	Disk                  float64           `json:"disk,omitempty"`
	Env                   map[string]string `json:"env"`
	Executor              string            `json:"executor,omitempty"`
	Instances             int               `json:"instances,omitempty"`
	Mem                   float64           `json:"mem,omitempty"`
	Ports                 []int             `json:"ports"`
	RequirePorts          bool              `json:"requirePorts,omitempty"`
	BackoffSeconds        float64           `json:"backoffSeconds,omitempty"`
	BackoffFactor         float64           `json:"backoffFactor,omitempty"`
	MaxLaunchDelaySeconds float64           `json:"maxLaunchDelaySeconds,omitempty"`
	Dependencies          []string          `json:"dependencies"`
	TasksRunning          int               `json:"tasksRunning,omitempty"`
	TasksStaged           int               `json:"tasksStaged,omitempty"`
	TasksHealthy          int               `json:"tasksHealthy,omitempty"`
	TasksUnhealthy        int               `json:"tasksUnhealthy,omitempty"`
	User                  string            `json:"user,omitempty"`
	Uris                  []string          `json:"uris"`
	Version               string            `json:"version,omitempty"`
	Labels                map[string]string `json:"labels,omitempty"`
	AcceptedResourceRoles []string          `json:"acceptedResourceRoles,omitempty"`
}

// Applications is a collection of applications
type Applications struct {
	Apps []Application `json:"apps"`
}

func deleteApp(url, user, password string, appID string) error {
	uri := url + path.Join("/v2/apps", appID)
	req, err := http.NewRequest("DELETE", uri, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(user, password)

	resp, err := hatchery.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s", resp.Status)
	}

	return nil
}

func getApps(url, user, password string, env string) ([]Application, error) {
	req, err := http.NewRequest("GET", url+"/v2/apps?embed=apps.counts&id="+env, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(user, password)

	resp, err := hatchery.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apps Applications
	if err = json.Unmarshal(body, &apps); err != nil {
		return nil, err
	}

	return apps.Apps, nil
}

func countOf(model string, apps []Application) int {
	var count int

	for i := range apps {
		if strings.Contains(apps[i].ID, "/"+strings.ToLower(model)+"-") {
			count++
		}
	}

	return count
}

func getDeployments(url, user, password string) ([]Application, error) {
	req, err := http.NewRequest("GET", url+"/v2/deployments", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(user, password)

	resp, err := hatchery.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apps []Application
	if err = json.Unmarshal(body, &apps); err != nil {
		return nil, err
	}

	return apps, nil
}
