package grpcplugins

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
	"github.com/pkg/errors"
)

type Img struct {
	Repository string
	Tag        string
	ImageID    string
	Created    string
	Size       string
}

func ComputeRunResultDockerDetail(name string, img Img) sdk.V2WorkflowRunResultDetail {
	return sdk.V2WorkflowRunResultDetail{
		Data: sdk.V2WorkflowRunResultDockerDetail{
			Name:         name,
			ID:           img.ImageID,
			HumanSize:    img.Size,
			HumanCreated: img.Created,
		},
	}
}

type dockerManifestConfig struct {
	Digest string `json:"digest"`
}
type dockerManifest struct {
	Config dockerManifestConfig `json:"config"`
}

func getDockerManifest(ctx context.Context, c *actionplugin.Common, rtConfig ArtifactoryConfig, manifestDownloadURI string) (*dockerManifest, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", manifestDownloadURI, nil)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to create request to retrieve file docker manifest")
	}

	req.Header.Set("Authorization", "Bearer "+rtConfig.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to get docker manifest file")
	}

	if resp.StatusCode > 200 {
		return nil, sdk.Errorf("unable to download file %s (HTTP %d)", manifestDownloadURI, resp.StatusCode)
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var manifest dockerManifest
	if err := json.Unmarshal(bts, &manifest); err != nil {
		return nil, sdk.WrapError(err, "unable to read docker manifest")
	}

	return &manifest, nil
}

// FinalizeRunResultDockerDetail is computing the runResult Object for a docker image (imageDestinationName) push from local reference (imageStruct)
// As result, the parameter result is updated
// This function is used by addRunResult and dockerPush actions
func FinalizeRunResultDockerDetail(ctx context.Context, c *actionplugin.Common, rtConfig ArtifactoryConfig, result *sdk.V2WorkflowRunResult, imageDestinationName string, imageStruct *Img) (err error) {
	jobCtx, err := GetJobContext(ctx, c)
	if err != nil {
		return err
	}

	// Reset run result details because dockerPush action is not doing it properly on creation
	result.Detail = ComputeRunResultDockerDetail(imageDestinationName, *imageStruct)

	splittedImageDestinationName := strings.SplitN(imageDestinationName, ":", 2)
	if len(splittedImageDestinationName) != 2 {
		return errors.Errorf("invalid imageDestinationName: %s", imageDestinationName)
	}

	// Computing the destination path (repository, maturity, etc...) from the image
	integration := jobCtx.Integrations.ArtifactManager
	maturity := integration.Get(sdk.ArtifactoryConfigPromotionLowMaturity)
	dockerRepo := integration.Get(sdk.ArtifactoryConfigRepositoryPrefix) + "-docker"
	rtFolderPath := imageStruct.Repository + "/" + imageStruct.Tag
	// Each docker tags are different folder
	rtFolderPathInfo, err := GetArtifactoryFolderInfo(ctx, c, rtConfig, dockerRepo+"-"+maturity, rtFolderPath)
	if err != nil {
		return sdk.WrapError(err, "unable to get folder %s/%s info", dockerRepo+"-"+maturity, rtFolderPath)
	}
	// It the tag folder, we have to found the docker image manifest
	var manifestFound bool
	var manifestDownloadURI string
	var manifestFileInfo *ArtifactoryFileInfo
	for _, child := range rtFolderPathInfo.Children {
		if strings.HasSuffix(child.URI, "manifest.json") { // Can be manifest.json of list.manifest.json for multi-arch docker image
			manifestFileInfo, err = GetArtifactoryFileInfo(ctx, c, rtConfig, dockerRepo+"-"+maturity, rtFolderPath+child.URI)
			if err != nil {
				return sdk.WrapError(err, "unable to get manifest %s/%s info", dockerRepo+"-"+maturity, rtFolderPath+child.URI)
			}
			manifestFound = true
			manifestDownloadURI = manifestFileInfo.DownloadURI // We have the download URI for the manifest, we download it later
			// Extract details to put in the details of the run result
			ExtractFileInfoIntoRunResult(result, *manifestFileInfo, imageDestinationName, "docker", dockerRepo+"-"+maturity, dockerRepo, maturity)
			result.ArtifactManagerMetadata.Set("id", imageStruct.ImageID)
			break
		}
	}
	// Get the manifest file to get the ImageID and the date
	manifest, err := getDockerManifest(ctx, c, rtConfig, manifestDownloadURI)
	if err != nil {
		return sdk.WrapError(err, "unable to download manifest from %s", manifestDownloadURI)
	}
	imageStruct.ImageID = strings.TrimPrefix(manifest.Config.Digest, "sha256:")[0:12]
	imageStruct.Created = manifestFileInfo.Created.Format(time.RFC3339)
	result.Detail.Data = sdk.V2WorkflowRunResultDockerDetail{
		Name:         imageDestinationName,
		ID:           imageStruct.ImageID,
		HumanSize:    imageStruct.Size,
		HumanCreated: imageStruct.Created,
	}

	result.ArtifactManagerMetadata.Set("dir", rtFolderPathInfo.Path)
	if !manifestFound {
		return errors.New("unable to get uploaded image manifest")
	}
	details, err := sdk.GetConcreteDetail[*sdk.V2WorkflowRunResultDockerDetail](result)
	if err != nil {
		return err
	}
	// Now we are sure that the stored name is the one from artifactory
	details.Name = imageDestinationName
	result.Detail.Data = details
	result.Status = sdk.V2WorkflowRunResultStatusCompleted
	return nil
}

func ComputeRunResultDebianDetail(name string, size int64, md5, sha1, sha256 string, components, distributions, architectures []string) sdk.V2WorkflowRunResultDetail {
	return sdk.V2WorkflowRunResultDetail{
		Data: sdk.V2WorkflowRunResultDebianDetail{
			Name:          name,
			Size:          size,
			MD5:           md5,
			SHA1:          sha1,
			SHA256:        sha256,
			Components:    components,
			Distributions: distributions,
			Architectures: architectures,
		},
	}
}

func ComputeRunResultTestsDetail(c *actionplugin.Common, filePath string, fileContent []byte, size int64, md5, sha1, sha256 string) (*sdk.V2WorkflowRunResultDetail, int, error) {
	var ftests sdk.JUnitTestsSuites
	if err := xml.Unmarshal(fileContent, &ftests); err != nil {
		// Check if file contains testsuite only (and no testsuites)
		var s sdk.JUnitTestSuite
		if err := xml.Unmarshal([]byte(fileContent), &s); err != nil {
			Error(c, fmt.Sprintf("Unable to unmarshal junit file %q: %v.", filePath, err))
			return nil, 0, errors.New("unable to read file " + filePath)
		}

		if s.Name != "" {
			ftests.TestSuites = append(ftests.TestSuites, s)
		}
	}

	reportLogs := computeTestsReasons(ftests)
	for _, l := range reportLogs {
		Log(c, l)
	}
	ftests = ftests.EnsureData()
	stats := ftests.ComputeStats()

	_, fileName := filepath.Split(filePath)
	perm := os.FileMode(0755)

	// Create run result at status "pending"
	return &sdk.V2WorkflowRunResultDetail{
		Data: sdk.V2WorkflowRunResultTestDetail{
			Name:        fileName,
			Size:        size,
			Mode:        perm,
			MD5:         md5,
			SHA1:        sha1,
			SHA256:      sha256,
			TestsSuites: ftests,
			TestStats:   stats,
		}}, stats.TotalKO, nil

}

func computeTestsReasons(s sdk.JUnitTestsSuites) []string {
	reasons := []string{fmt.Sprintf("JUnit parser: %d testsuite(s)", len(s.TestSuites))}
	for _, ts := range s.TestSuites {
		reasons = append(reasons, fmt.Sprintf("JUnit parser: testsuite %s has %d testcase(s)", ts.Name, len(ts.TestCases)))
		for _, tc := range ts.TestCases {
			if len(tc.Failures) > 0 {
				reasons = append(reasons, fmt.Sprintf("JUnit parser: testcase %s has %d failure(s)", tc.Name, len(tc.Failures)))
			}
			if len(tc.Errors) > 0 {
				reasons = append(reasons, fmt.Sprintf("JUnit parser: testcase %s has %d error(s)", tc.Name, len(tc.Errors)))
			}
			if len(tc.Skipped) > 0 {
				reasons = append(reasons, fmt.Sprintf("JUnit parser: testcase %s has %d test(s) skipped", tc.Name, len(tc.Skipped)))
			}
		}
		if ts.Failures > 0 {
			reasons = append(reasons, fmt.Sprintf("JUnit parser: testsuite %s has %d failure(s)", ts.Name, ts.Failures))
		}
		if ts.Errors > 0 {
			reasons = append(reasons, fmt.Sprintf("JUnit parser: testsuite %s has %d error(s)", ts.Name, ts.Errors))
		}
		if ts.Failures+ts.Errors > 0 {
			reasons = append(reasons, fmt.Sprintf("JUnit parser: testsuite %s has %d test(s) failed", ts.Name, ts.Failures+ts.Errors))
		}
		if ts.Skipped > 0 {
			reasons = append(reasons, fmt.Sprintf("JUnit parser: testsuite %s has %d test(s) skipped", ts.Name, ts.Skipped))
		}
	}
	return reasons
}

func ComputeRunResultHelmDetail(chartName, appVersion, chartVersion string) sdk.V2WorkflowRunResultDetail {
	return sdk.V2WorkflowRunResultDetail{
		Data: sdk.V2WorkflowRunResultHelmDetail{
			Name:         chartName,
			AppVersion:   appVersion,
			ChartVersion: chartVersion,
		},
	}
}

func ComputeRunResultPythonDetail(packageName string, version string, extension string) sdk.V2WorkflowRunResultDetail {
	return sdk.V2WorkflowRunResultDetail{
		Data: sdk.V2WorkflowRunResultPythonDetail{
			Name:      packageName,
			Version:   version,
			Extension: extension,
		},
	}
}
