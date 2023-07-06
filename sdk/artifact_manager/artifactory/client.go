package artifactory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	buildinfo "github.com/jfrog/build-info-go/entities"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/pkg/errors"

	"github.com/ovh/cds/sdk"
)

type Client struct {
	Asm artifactory.ArtifactoryServicesManager
}

func (c *Client) GetFolderInfo(repoName string, folderPath string) (*utils.FolderInfo, error) {
	return c.Asm.FolderInfo(repoName + "/" + folderPath)
}

func (c *Client) GetRepository(repoName string) (*services.RepositoryDetails, error) {
	var repoDetails services.RepositoryDetails
	if err := c.Asm.GetRepository(repoName, &repoDetails); err != nil {
		return nil, err
	}
	return &repoDetails, nil
}

func (c *Client) GetFileInfo(repoName string, filePath string) (sdk.FileInfo, error) {
	fi := sdk.FileInfo{}

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

	return fi, nil
}

func (c *Client) SetProperties(repoName string, filePath string, props *utils.Properties) error {
	if props == nil {
		return nil
	}
	var properties []string
	for key, values := range props.ToMap() {
		properties = append(properties, url.QueryEscape(key)+"="+url.QueryEscape(strings.Join(values, ",")))
	}
	// https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-SetItemProperties
	propertiesString := strings.Join(properties, ";")
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}
	fileInfoURL := fmt.Sprintf("%sapi/storage/%s%s?properties=%s&recursive=1", c.Asm.GetConfig().GetServiceDetails().GetUrl(), repoName, filePath, propertiesString)
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

func (c *Client) GetProperties(repoName string, filePath string) (map[string][]string, error) {
	type PropertiesResponse struct {
		Properties map[string][]string `json:"properties"`
	}

	var rp = services.NewCreateReplicationService(c.Asm.Client())
	rp.ArtDetails = c.Asm.GetConfig().GetServiceDetails()
	var httpDetails = rp.ArtDetails.CreateHttpClientDetails()
	propUrl := fmt.Sprintf("%sapi/storage/%s%s?properties", c.Asm.GetConfig().GetServiceDetails().GetUrl(), repoName, filePath)

	resp, body, _, err := c.Asm.Client().SendGet(propUrl, true, &httpDetails)
	if err != nil {
		return nil, errors.Errorf("unable to get properties on %s%s: %v", repoName, filePath, err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, errors.WithStack(errorutils.CheckError(errors.New(fmt.Sprintf("GET Properties Artifactory on repository %s response: [%d] %v\n", key, resp.StatusCode, string(body)))))
	}

	var props PropertiesResponse
	if err := json.Unmarshal(body, &props); err != nil {
		return nil, errors.Errorf("unable get properties, unable to read response: %s: %v", string(body), err)
	}
	return props.Properties, nil
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

func (c *Client) CheckArtifactExists(repoName string, artiName string) (bool, error) {
	httpDetails := c.Asm.GetConfig().GetServiceDetails().CreateHttpClientDetails()
	fileInfoURL := fmt.Sprintf("%sapi/storage/%s/%s", c.Asm.GetConfig().GetServiceDetails().GetUrl(), repoName, artiName)
	re, body, _, err := c.Asm.Client().SendGet(fileInfoURL, true, &httpDetails)
	if err != nil {
		return false, fmt.Errorf("unable to get file info %s/%s: %v", repoName, artiName, err)
	}
	if re.StatusCode == 404 {
		return false, nil
	}
	if re.StatusCode >= 400 {
		return false, fmt.Errorf("unable to call artifactory [HTTP: %d] %s %s", re.StatusCode, fileInfoURL, string(body))
	}
	return true, nil
}

func (c *Client) PromoteDocker(params services.DockerPromoteParams) error {
	return c.Asm.PromoteDocker(params)
}

func (c *Client) Move(params services.MoveCopyParams) (successCount, failedCount int, err error) {
	return c.Asm.Move(params)
}

func (c *Client) Copy(params services.MoveCopyParams) (successCount, failedCount int, err error) {
	return c.Asm.Copy(params)
}

type PropertiesResponse struct {
	Properties map[string][]string
}

func (c *Client) GetRepositoryMaturity(repoName string) (string, error) {
	httpDetails := c.Asm.GetConfig().GetServiceDetails().CreateHttpClientDetails()
	uri := fmt.Sprintf(c.Asm.GetConfig().GetServiceDetails().GetUrl()+"api/storage/%s?properties", repoName)
	re, body, _, err := c.Asm.Client().SendGet(uri, true, &httpDetails)
	if err != nil {
		return "", errors.Errorf("unable to get properties %s: %v", repoName, err)
	}
	if re.StatusCode == 404 {
		return "", errors.Errorf("repository %s properties not found", repoName)
	}
	if re.StatusCode >= 400 {
		return "", errors.Errorf("unable to call artifactory [HTTP: %d] %s %s", re.StatusCode, uri, string(body))
	}
	var props PropertiesResponse
	if err := json.Unmarshal(body, &props); err != nil {
		return "", errors.WithStack(err)
	}
	fmt.Printf("Repository %q has properties: %+v\n", repoName, props.Properties)
	for k, p := range props.Properties {
		if k == "ovh.maturity" {
			return p[0], nil
		}
	}
	return "", nil
}

func (c *Client) Search(_ context.Context, query string) (sdk.ArtifactResults, error) {
	var result sdk.ArtifactResults
	var offset int
	var nbPage int
	for {
		nbPage = nbPage + 1
		query := fmt.Sprintf(query+".offset(%d).limit(100)", offset)
		body, err := c.Asm.Aql(query)
		if err != nil {
			return nil, err
		}

		btes, err := io.ReadAll(body)
		body.Close()
		if err != nil {
			return nil, errors.WithStack(err)
		}

		var page sdk.ArtifactResultsSearchPage
		if err := json.Unmarshal(btes, &page); err != nil {
			return nil, errors.WithStack(err)
		}

		if page.Range.EndPos == 0 {
			break
		}

		result = append(result, page.Results...)
		offset = page.Range.StartPos + page.Range.EndPos
	}

	return unique(result), nil
}

func unique(s sdk.ArtifactResults) sdk.ArtifactResults {
	inResult := make(map[string]struct{})
	var result sdk.ArtifactResults
	for _, i := range s {
		if _, ok := inResult[i.String()]; !ok {
			inResult[i.String()] = struct{}{}
			result = append(result, i)
		}
	}
	return result
}
