package edge

import (
	"fmt"
	"strings"

	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/distribution"
	authdistrib "github.com/jfrog/jfrog-client-go/distribution/auth"
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

func ListEdgeNodes(distriClient *distribution.DistributionServicesManager, url, token string) ([]EdgeNode, error) {
	listEdgeNodePath := fmt.Sprintf("api/ui/distribution/edge_nodes?action=x")
	dtb := authdistrib.NewDistributionDetails()
	dtb.SetUrl(strings.Replace(url, "/artifactory/", "/distribution/", -1))
	dtb.SetAccessToken(token)

	fakeService := services.NewCreateReleaseBundleService(distriClient.Client())
	fakeService.DistDetails = dtb
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

func RemoveNonEdge(edges []EdgeNode) []EdgeNode {
	edgeFiltered := make([]EdgeNode, 0, len(edges))
	for _, e := range edges {
		if e.LicenseType != "EDGE" {
			continue
		}
		edgeFiltered = append(edgeFiltered, e)
	}
	return edgeFiltered
}
