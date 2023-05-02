package edge

import (
	"fmt"

	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/distribution"
	"github.com/jfrog/jfrog-client-go/distribution/services"
	"github.com/ovh/cds/sdk"
)

type EdgeNode struct {
	Name     string `json:"name"`
	SiteName string `json:"site_name"`
	City     struct {
		Name        string `json:"name"`
		CountryCode string `json:"country_code"`
	} `json:"city"`
	LicenseType   string `json:"license_type"`
	LicenseStatus string `json:"license_status"`
}

func ListEdgeNodes(distriClient distribution.DistributionServicesManager) ([]EdgeNode, error) {
	listEdgeNodePath := fmt.Sprintf("api/ui/distribution/edge_nodes?action=x")
	fakeService := services.NewCreateReleaseBundleService(distriClient.Client())
	fakeService.DistDetails = distriClient.Config().GetServiceDetails()
	clientDetail := fakeService.DistDetails.CreateHttpClientDetails()

	listEdgeURL := fmt.Sprintf("%s%s", fakeService.DistDetails.GetUrl(), listEdgeNodePath)
	utils.SetContentType("application/json", &clientDetail.Headers)

	resp, body, _, err := distriClient.Client().SendGet(listEdgeURL, true, &clientDetail)
	if err != nil {
		return nil, fmt.Errorf("unable to list edge node from distribution: %v", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http error %d: %s", resp.StatusCode, string(body))
	}

	var edges []EdgeNode
	if err := sdk.JSONUnmarshal(body, &edges); err != nil {
		return nil, fmt.Errorf("unable to unmarshal response %s: %v", string(body), err)
	}
	return edges, nil
}
