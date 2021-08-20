package artifactory

import (
	"fmt"
	"strconv"
	"time"

	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/ovh/cds/sdk"
)

type FileInfoResponse struct {
	Checksums         *FileInfoChecksum `json:"checksums"`
	Created           time.Time         `json:"created"`
	CreatedBy         string            `json:"createdBy"`
	DownloadURI       string            `json:"downloadUri"`
	LastModified      time.Time         `json:"lastModified"`
	LastUpdated       time.Time         `json:"lastUpdated"`
	MimeType          string            `json:"mimeType"`
	ModifiedBy        string            `json:"modifiedBy"`
	OriginalChecksums *FileInfoChecksum `json:"originalChecksums"`
	Path              string            `json:"path"`
	RemoteURL         string            `json:"remoteUrl"`
	Repo              string            `json:"repo"`
	Size              string            `json:"size"`
	URI               string            `json:"uri"`
}

type FileInfoChecksum struct {
	Md5    string `json:"md5"`
	Sha1   string `json:"sha1"`
	Sha256 string `json:"sha256"`
}

type Client struct {
	Asm artifactory.ArtifactoryServicesManager
}

func (c *Client) GetFileInfo(repoName string, filePath string) (sdk.FileInfo, error) {
	fi := sdk.FileInfo{}
	repoDetails := services.RepositoryDetails{}
	if err := c.Asm.GetRepository(repoName, &repoDetails); err != nil {
		return fi, sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to get repository %s: %v", repoName, err)
	}
	fi.Type = repoDetails.PackageType

	fileInfoURL := fmt.Sprintf("%sapi/storage/%s/%s", c.Asm.GetConfig().GetServiceDetails().GetUrl(), repoName, filePath)
	httpDetails := c.Asm.GetConfig().GetServiceDetails().CreateHttpClientDetails()
	re, body, _, err := c.Asm.Client().SendGet(fileInfoURL, true, &httpDetails)
	if err != nil {
		return fi, sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to call artifactory: %v", err)
	}

	if re.StatusCode >= 400 {
		return fi, sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to call artifactory [HTTP: %d] %s %s", re.StatusCode, fileInfoURL, string(body))
	}

	var resp FileInfoResponse
	if err := sdk.JSONUnmarshal(body, &resp); err != nil {
		return fi, sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to read artifactory response %s: %v", string(body), err)
	}

	if resp.Size != "" {
		s, err := strconv.ParseInt(resp.Size, 10, 64)
		if err != nil {
			return fi, sdk.NewErrorFrom(sdk.ErrInvalidData, "size return by artifactory is not an integer %s: %v", resp.Size, err)
		}
		fi.Size = s
	}
	if resp.Checksums != nil {
		fi.Md5 = resp.Checksums.Md5
	}
	return fi, nil
}
