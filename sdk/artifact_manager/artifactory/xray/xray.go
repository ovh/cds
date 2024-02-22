package xray

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ovh/cds/sdk/artifact_manager/artifactory/http"
)

type Client interface {
	GetReleaseBundleSBOM(ctx context.Context, name, version string) (CycloneDXReport, error)
	GetReleaseBundleSBOMRaw(ctx context.Context, name, version string) (json.RawMessage, error)
}

func NewClient(artifactory_url string, token string) (Client, error) {
	c := &client{}

	c.config.Host = artifactory_url
	c.config.Token = token
	c.httpClient = http.NewClient(artifactory_url, token)
	return c, nil
}

type client struct {
	httpClient http.HTTPClient
	config     struct {
		Host  string
		Token string
	}
}

func (c *client) GetReleaseBundleSBOM(ctx context.Context, name, version string) (CycloneDXReport, error) {
	componentDetails := ComponentDetailsRequest{
		ComponentName:   name + ":" + version,
		PackageType:     "ReleaseBundle",
		Cyclonedx:       true,
		CyclonedxFormat: "json",
	}
	var res CycloneDXReport
	code, err := c.httpClient.PostJSON(ctx, "/api/v1/component/exportDetails", componentDetails, &res)
	if err != nil {
		return res, err
	}
	if err := CheckError(code); err != nil {
		return res, err
	}
	return res, nil
}

func (c *client) GetReleaseBundleSBOMRaw(ctx context.Context, name, version string) (json.RawMessage, error) {
	componentDetails := ComponentDetailsRequest{
		ComponentName:   name + ":" + version,
		PackageType:     "ReleaseBundle",
		Cyclonedx:       true,
		CyclonedxFormat: "json",
	}
	var res json.RawMessage
	code, err := c.httpClient.PostJSON(ctx, "/api/v1/component/exportDetails", componentDetails, &res)
	if err != nil {
		return res, err
	}
	if err := CheckError(code); err != nil {
		return res, err
	}
	return res, nil
}

type CycloneDXReport struct {
	BomFormat    string `json:"bomFormat"`
	SpecVersion  string `json:"specVersion"`
	SerialNumber string `json:"serialNumber"`
	Version      int    `json:"version"`
	Metadata     struct {
		Timestamp time.Time `json:"timestamp"`
		Tools     []struct {
			Vendor  string `json:"vendor"`
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"tools"`
		Component struct {
			Type    string `json:"type"`
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"component"`
	} `json:"metadata"`
	Components      []CycloneDXReportComponent       `json:"components"`
	Vulnerabilities []CycloneDXReportVulnerabilities `json:"vulnerabilities"`
}

type CycloneDXReportComponent struct {
	BomRef  string `json:"bom-ref"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Hashes  []struct {
		Alg     string `json:"alg"`
		Content string `json:"content"`
	} `json:"hashes,omitempty"`
	Licenses []struct {
		License struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		} `json:"license"`
	} `json:"licenses"`
	Purl string `json:"purl"`
}

type CycloneDXReportVulnerabilities struct {
	BomRef   string `json:"bom-ref"`
	ID       string `json:"id"`
	Analysis struct {
		State  string `json:"state"`
		Detail string `json:"detail"`
	} `json:"analysis,omitempty"`
}
