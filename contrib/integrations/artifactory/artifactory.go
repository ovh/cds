package art

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/auth"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/jfrog/jfrog-client-go/config"
	"github.com/jfrog/jfrog-client-go/distribution"
	authdistrib "github.com/jfrog/jfrog-client-go/distribution/auth"

	"github.com/ovh/cds/sdk"
)

type FileInfoResponse struct {
	Checksums         *FileInfoChecksum `json:"checksums,omitempty"`
	Created           time.Time         `json:"created"`
	CreatedBy         string            `json:"createdBy"`
	DownloadURI       string            `json:"downloadUri"`
	LastModified      time.Time         `json:"lastModified"`
	LastUpdated       time.Time         `json:"lastUpdated"`
	MimeType          string            `json:"mimeType"`
	ModifiedBy        string            `json:"modifiedBy"`
	OriginalChecksums *FileInfoChecksum `json:"originalChecksums,omitempty"`
	Path              string            `json:"path"`
	RemoteURL         string            `json:"remoteUrl"`
	Repo              string            `json:"repo"`
	Size              string            `json:"size"`
	URI               string            `json:"uri"`
	Children          []FileChildren    `json:"children,omitempty"`
}

type FileInfoChecksum struct {
	Md5    string `json:"md5"`
	Sha1   string `json:"sha1"`
	Sha256 string `json:"sha256"`
}

type FileChildren struct {
	Uri    string `json:"uri"`
	Folder bool   `json:"folder"`
}

func CreateDistributionClient(url, token string) (*distribution.DistributionServicesManager, error) {
	dtb := authdistrib.NewDistributionDetails()
	dtb.SetUrl(strings.Replace(url, "/artifactory/", "/distribution/", -1))
	dtb.SetAccessToken(token)
	serviceConfig, err := config.NewConfigBuilder().
		SetServiceDetails(dtb).
		SetThreads(1).
		SetDryRun(false).
		Build()
	if err != nil {
		return nil, fmt.Errorf("unable to create service config: %v", err)
	}
	return distribution.New(serviceConfig)
}

func CreateArtifactoryClient(url, token string) (artifactory.ArtifactoryServicesManager, error) {
	rtDetails := auth.NewArtifactoryDetails()
	rtDetails.SetUrl(url)
	rtDetails.SetAccessToken(token)
	serviceConfig, err := config.NewConfigBuilder().
		SetServiceDetails(rtDetails).
		SetThreads(1).
		SetDryRun(false).
		Build()
	if err != nil {
		return nil, fmt.Errorf("unable to create service config: %v", err)
	}
	return artifactory.New(serviceConfig)
}

func GetFileInfo(artiClient artifactory.ArtifactoryServicesManager, repoName string, filePath string) (FileInfoResponse, error) {
	var resp FileInfoResponse
	fi := sdk.FileInfo{}
	repoDetails := services.RepositoryDetails{}
	if err := artiClient.GetRepository(repoName, &repoDetails); err != nil {
		return resp, sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to get repository %s: %v", repoName, err)
	}
	fi.Type = repoDetails.PackageType

	fileInfoURL := fmt.Sprintf("%sapi/storage/%s/%s", artiClient.GetConfig().GetServiceDetails().GetUrl(), repoName, filePath)
	httpDetails := artiClient.GetConfig().GetServiceDetails().CreateHttpClientDetails()
	re, body, _, err := artiClient.Client().SendGet(fileInfoURL, true, &httpDetails)
	if err != nil {
		return resp, sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to call artifactory: %v", err)
	}

	if re.StatusCode >= 400 {
		return resp, sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to call artifactory [HTTP: %d] %s", re.StatusCode, string(body))
	}

	if err := sdk.JSONUnmarshal(body, &resp); err != nil {
		return resp, sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to read artifactory response %s: %v", string(body), err)
	}
	return resp, nil
}

func SetProperties(artiClient artifactory.ArtifactoryServicesManager, repoName string, filePath string, props map[string]string) error {
	fileInfoURL := fmt.Sprintf("%sapi/storage/%s/%s?properties=", artiClient.GetConfig().GetServiceDetails().GetUrl(), repoName, filePath)

	for k, v := range props {
		fileInfoURL += fmt.Sprintf("%s=%s%s", k, url.QueryEscape(v), url.QueryEscape("|"))

	}
	httpDetails := artiClient.GetConfig().GetServiceDetails().CreateHttpClientDetails()
	resp, body, err := artiClient.Client().SendPut(fileInfoURL, nil, &httpDetails)
	if err != nil {
		return sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to call artifactory: %v", err)
	}

	if resp.StatusCode >= 400 {
		return sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to call artifactory [HTTP: %d] %s: %s", resp.StatusCode, fileInfoURL, string(body))
	}
	return nil
}
