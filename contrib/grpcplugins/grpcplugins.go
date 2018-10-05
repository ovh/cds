package grpcplugins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ovh/cds/sdk"
)

// SendVulnerabilityReport call worker to send vulnerabiliry report to API
func SendVulnerabilityReport(workerHTTPPort int32, report sdk.VulnerabilityWorkerReport) error {
	if workerHTTPPort == 0 {
		return nil
	}

	data, errD := json.Marshal(report)
	if errD != nil {
		return fmt.Errorf("unable to marshal report: %v", errD)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/vulnerability", workerHTTPPort), bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("send report to worker /vulnerability: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot send report to worker /vulnerability: %v", err)
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("cannot send report to worker /vulnerability: HTTP %d", resp.StatusCode)
	}

	return nil
}

// GetExternalServices call worker to get external service configuration
func GetExternalServices(workerHTTPPort int32, serviceType string) (sdk.ExternalService, error) {
	if workerHTTPPort == 0 {
		return sdk.ExternalService{}, nil
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/services/%s", workerHTTPPort, serviceType), nil)
	if err != nil {
		return sdk.ExternalService{}, fmt.Errorf("get service from worker /services: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return sdk.ExternalService{}, fmt.Errorf("cannot get service from worker /services: %v", err)
	}

	if resp.StatusCode >= 300 {
		return sdk.ExternalService{}, fmt.Errorf("cannot get services from worker /services: HTTP %d", resp.StatusCode)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return sdk.ExternalService{}, fmt.Errorf("cannot read body /services: %v", err)
	}

	var serv sdk.ExternalService
	if err := json.Unmarshal(b, &serv); err != nil {
		return serv, fmt.Errorf("cannot unmarshal body /services: %v", err)
	}
	return serv, nil
}
