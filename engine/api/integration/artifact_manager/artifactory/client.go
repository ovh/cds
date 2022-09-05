package artifactory

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/buildinfo"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"

	"github.com/ovh/cds/sdk"
)

type Client struct {
	Asm artifactory.ArtifactoryServicesManager
}

func (c *Client) GetFileInfo(repoName string, filePath string) (sdk.FileInfo, error) {
	fi := sdk.FileInfo{}
	repoDetails := services.RepositoryDetails{}
	if err := c.Asm.GetRepository(repoName, &repoDetails); err != nil {
		return fi, sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to get repository %s: %v", repoName, err)
	}

	// To get FileInfo for a docker image, we have to check the manifest file
	if repoDetails.PackageType == "docker" && !strings.HasSuffix(filePath, "manifest.json") {
		filePath = path.Join(filePath, "manifest.json")
	}

	fileInfoURL := fmt.Sprintf("%sapi/storage/%s/%s", c.Asm.GetConfig().GetServiceDetails().GetUrl(), repoName, filePath)
	httpDetails := c.Asm.GetConfig().GetServiceDetails().CreateHttpClientDetails()
	re, body, _, err := c.Asm.Client().SendGet(fileInfoURL, true, &httpDetails)
	if err != nil {
		return fi, sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to call artifactory: %v", err)
	}

	if re.StatusCode >= 400 {
		return fi, sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to call artifactory [HTTP: %d] %s %s", re.StatusCode, fileInfoURL, string(body))
	}

	if err := sdk.JSONUnmarshal(body, &fi); err != nil {
		return fi, sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to read artifactory response %s: %v", string(body), err)
	}

	if fi.SizeString != "" {
		s, err := strconv.ParseInt(fi.SizeString, 10, 64)
		if err != nil {
			return fi, sdk.NewErrorFrom(sdk.ErrInvalidData, "size return by artifactory is not an integer %s: %v", fi.SizeString, err)
		}
		fi.Size = s
	}
	fi.Type = repoDetails.PackageType

	return fi, nil
}

func (c *Client) SetProperties(repoName string, filePath string, values ...sdk.KeyValues) error {
	var properties string
	for i, kv := range values {
		if i > 0 {
			properties += ";" // https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-SetItemProperties
		}
		properties += url.QueryEscape(kv.Key) + "=" + url.QueryEscape(strings.Join(kv.Values, ","))
	}
	fileInfoURL := fmt.Sprintf("%sapi/storage/%s/%s?properties=%s&recursive=1", c.Asm.GetConfig().GetServiceDetails().GetUrl(), repoName, filePath, properties)
	httpDetails := c.Asm.GetConfig().GetServiceDetails().CreateHttpClientDetails()
	re, _, err := c.Asm.Client().SendPut(fileInfoURL, nil, &httpDetails)
	if err != nil {
		return sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to call artifactory: %v", err)
	}

	if re.StatusCode >= 400 {
		return sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to call artifactory [HTTP: %d] %s", re.StatusCode, fileInfoURL)
	}

	return nil
}

type DeleteBuildRequest struct {
	Project         string   `json:"project"`
	BuildName       string   `json:"buildName"`
	BuildNumbers    []string `json:"buildNumbers"`
	DeleteArtifacts bool     `json:"deleteArtifacts"`
	DeleteAll       bool     `json:"deleteAll"`
}

func (c *Client) DeleteBuild(project string, buildName string, buildVersion string) error {
	httpDetails := c.Asm.GetConfig().GetServiceDetails().CreateHttpClientDetails()
	utils.SetContentType("application/json", &httpDetails.Headers)
	request := DeleteBuildRequest{
		Project:      project,
		BuildName:    buildName,
		BuildNumbers: []string{buildVersion},
	}
	bts, _ := json.Marshal(request)
	deleteBuildURL := fmt.Sprintf("%sapi/build/delete", c.Asm.GetConfig().GetServiceDetails().GetUrl())
	re, body, err := c.Asm.Client().SendPost(deleteBuildURL, bts, &httpDetails)
	if err != nil {
		return err
	}
	if re.StatusCode == http.StatusNotFound || re.StatusCode < 400 {
		return nil
	}
	return fmt.Errorf("unable to delete build: %s", string(body))
}

func (c *Client) PublishBuildInfo(project string, request *buildinfo.BuildInfo) error {
	var nbAttempts int
	for {
		nbAttempts++
		_, err := c.Asm.PublishBuildInfo(request, project)
		if err == nil {
			break
		} else if nbAttempts >= 3 {
			return err
		}
	}
	return nil
}

func (c *Client) XrayScanBuild(params services.XrayScanParams) ([]byte, error) {
	return c.Asm.XrayScanBuild(params)
}

func (c *Client) GetURL() string {
	return c.Asm.GetConfig().GetServiceDetails().GetUrl()
}
