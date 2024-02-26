package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	cm "github.com/chartmuseum/helm-push/pkg/chartmuseum"
	cmhelm "github.com/chartmuseum/helm-push/pkg/helm"
	"github.com/jfrog/jfrog-client-go/utils/io/fileutils"
	"github.com/ovh/cds/contrib/grpcplugins"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
	"github.com/ovh/cds/sdk/helm"
	"github.com/pkg/errors"

	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	HelmInstallURL = "https://get.helm.sh/helm-v3.14.0-linux-amd64.tar.gz"
)

type helmPushPlugin struct {
	actionplugin.Common
}

func main() {
	actPlugin := helmPushPlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
}

// Manifest implements actionplugin.ActionPluginServer.
func (*helmPushPlugin) Manifest(context.Context, *emptypb.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "helm-push",
		Author:      "François SAMIN <francois.samin@corp.ovh.com>",
		Description: "Push helm chart on a helm registry",
		Version:     sdk.VERSION,
	}, nil
}

// Run implements actionplugin.ActionPluginServer.
func (p *helmPushPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	res := &actionplugin.ActionResult{
		Status: sdk.StatusSuccess,
	}

	chartFolder := q.GetOptions()["chartFolder"]
	chartVersion := q.GetOptions()["chartVersion"]
	appVersion := q.GetOptions()["appVersion"]
	updateDependenciesS := q.GetOptions()["updateDependencies"]
	registryURL := q.GetOptions()["registryURL"]
	registryUsername := q.GetOptions()["registryUsername"]
	registryPassword := q.GetOptions()["registryPassword"]
	registryAccessToken := q.GetOptions()["registryAccessToken"]
	registryAuthHeader := q.GetOptions()["registryAuthHeader"]

	var updateDependencies = false
	if strings.TrimSpace(updateDependenciesS) != "" {
		var err error
		updateDependencies, err = strconv.ParseBool(updateDependenciesS)
		if err != nil {
			res.Status = sdk.StatusFail
			res.Details = "invalid value for parameter <updateDependencies>"
			return res, err
		}
	}

	registryOpts := chartMuseumOptions{
		registryURL:         registryURL,
		registryUsername:    registryUsername,
		registryPassword:    registryPassword,
		registryAccessToken: registryAccessToken,
		registryAuthHeader:  registryAuthHeader,
	}
	result, d, err := p.perform(ctx, chartFolder, chartVersion, appVersion, updateDependencies, registryOpts)
	if err != nil {
		res.Status = sdk.StatusFail
		res.Details = err.Error()
		return res, err
	}

	grpcplugins.Logf("Helm chart %s pushed in %.3fs", result.Name(), d.Seconds())

	return res, nil
}

type chartMuseumOptions struct {
	registryURL         string
	registryUsername    string
	registryPassword    string
	registryAccessToken string
	registryAuthHeader  string
}

func (p *helmPushPlugin) perform(
	ctx context.Context, chartFolder string, chartVersion string, appVersion string, updateDependencies bool,
	registryOpts chartMuseumOptions,
) (*sdk.V2WorkflowRunResult, time.Duration, error) {
	var t0 = time.Now()

	// Prepare teh chart package
	chart, err := helm.GetChartByName(chartFolder)
	if err != nil {
		return nil, time.Since(t0), errors.Errorf("unable to get chart: %v", err)
	}

	if chartVersion != "" {
		chart.SetVersion(chartVersion)
	}

	if appVersion != "" {
		chart.SetAppVersion(appVersion)
	}

	tmp, err := os.MkdirTemp("", "plugin-helm-push")
	if err != nil {
		return nil, time.Since(t0), err
	}
	defer os.RemoveAll(tmp)

	chartPackagePath, err := helm.CreateChartPackage(chart, tmp)
	if err != nil {
		return nil, time.Since(t0), errors.Errorf("unable to create chart package: %v", err)
	}

	if updateDependencies {
		if err := helm.UpdateDependencies(chart); err != nil {
			return nil, time.Since(t0), errors.Errorf("unable to update chart dependencies: %v", err)
		}
	}

	// Create run result at status "pending"
	var runResultRequest = workerruntime.V2RunResultRequest{
		RunResult: &sdk.V2WorkflowRunResult{
			IssuedAt: time.Now(),
			Type:     sdk.V2WorkflowRunResultTypeHelm,
			Status:   sdk.V2WorkflowRunResultStatusPending,
			Detail: sdk.V2WorkflowRunResultDetail{
				Data: sdk.V2WorkflowRunResultHelmDetail{
					Name:         chart.Name(),
					AppVersion:   chart.AppVersion(),
					ChartVersion: chart.Metadata.Version,
				},
			},
		},
	}

	response, err := grpcplugins.CreateRunResult(ctx, &p.Common, &runResultRequest)
	if err != nil {
		return nil, time.Since(t0), err
	}

	result := response.RunResult

	switch {
	case result.ArtifactManagerIntegrationName != nil:
		integration, err := grpcplugins.GetIntegrationByName(ctx, &p.Common, *result.ArtifactManagerIntegrationName)
		if err != nil {
			return nil, time.Since(t0), err
		}

		rtConfig := grpcplugins.ArtifactoryConfig{
			URL:   integration.Config[sdk.ArtifactoryConfigURL].Value,
			Token: integration.Config[sdk.ArtifactoryConfigToken].Value,
		}

		if !strings.HasSuffix(rtConfig.URL, "/") {
			rtConfig.URL = rtConfig.URL + "/"
		}

		if err := p.pushArtifactory(ctx, result, chart, chartPackagePath, integration, rtConfig); err != nil {
			return nil, time.Since(t0), err
		}
	default:
		if err := p.pushChartMuseum(ctx, result, chart, chartPackagePath, registryOpts); err != nil {
			return nil, time.Since(t0), err
		}
	}

	// Update run result
	result.Status = sdk.V2WorkflowRunResultStatusCompleted

	updatedRunresult, err := grpcplugins.UpdateRunResult(ctx, &p.Common, &workerruntime.V2RunResultRequest{RunResult: result})
	return updatedRunresult.RunResult, time.Since(t0), err
}

func (p *helmPushPlugin) pushChartMuseum(ctx context.Context, result *sdk.V2WorkflowRunResult, chart *helm.Chart, chartPackagePath string, registryOpts chartMuseumOptions) error {
	client, err := cm.NewClient(
		cm.URL(registryOpts.registryURL),
		cm.Username(registryOpts.registryUsername),
		cm.Password(registryOpts.registryPassword),
		cm.AccessToken(registryOpts.registryAccessToken),
		cm.AuthHeader(registryOpts.registryAuthHeader),
		cm.Timeout(60),
	)
	if err != nil {
		return err
	}

	repo, err := cmhelm.TempRepoFromURL(registryOpts.registryURL)
	if err != nil {
		return err
	}

	index, err := cmhelm.GetIndexByRepo(repo, getIndexDownloader(client))
	if err != nil {
		return err
	}
	client.Option(cm.ContextPath(index.ServerInfo.ContextPath))

	grpcplugins.Logf("Pushing %s to %s...", filepath.Base(chartPackagePath), repo.Config.URL)
	resp, err := client.UploadChartPackage(chartPackagePath, true)
	if err != nil {
		return err
	}
	if resp.StatusCode != 201 && resp.StatusCode != 202 {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return getChartmuseumError(b, resp.StatusCode)
	}
	grpcplugins.Logf("Done.")

	result.ArtifactManagerMetadata = &sdk.V2WorkflowRunResultArtifactManagerMetadata{}
	result.ArtifactManagerMetadata.Set("repository", repo.Config.URL) // This is the virtual repository
	result.ArtifactManagerMetadata.Set("name", chart.Metadata.Name)
	result.ArtifactManagerMetadata.Set("chartVersion", chart.Metadata.Version)
	result.ArtifactManagerMetadata.Set("appVersion", chart.Metadata.AppVersion)

	return nil
}

func (p *helmPushPlugin) pushArtifactory(ctx context.Context, result *sdk.V2WorkflowRunResult, chart *helm.Chart, chartPackagePath string, integration *sdk.ProjectIntegration, rtConfig grpcplugins.ArtifactoryConfig) error {
	repository := integration.Config[sdk.ArtifactoryConfigRepositoryPrefix].Value + "-helm"

	resp, err := p.UploadChartPackageToArtifactory(ctx, repository, chart.Metadata.Name, chartPackagePath, rtConfig)
	if err != nil {
		return errors.Errorf("unable to upload chart package: %v", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		btes, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		grpcplugins.Error(string(btes))
		grpcplugins.Error(fmt.Sprintf("HTTP %d", resp.StatusCode))
		return errors.Errorf("unable to upload chart package: HTTP %d", resp.StatusCode)
	}

	fi, err := grpcplugins.GetArtifactoryFileInfo(context.TODO(), &p.Common, rtConfig, repository, path.Join(chart.Metadata.Name, filepath.Base(chartPackagePath)))
	if err != nil {
		return errors.Errorf("unable to get Artifactory file info %s: %v", chartPackagePath, err)
	}

	maturity := integration.Config[sdk.ArtifactoryConfigPromotionLowMaturity].Value
	localRepository := repository + "-" + maturity

	result.ArtifactManagerMetadata = &sdk.V2WorkflowRunResultArtifactManagerMetadata{}
	result.ArtifactManagerMetadata.Set("repository", repository) // This is the virtual repository
	result.ArtifactManagerMetadata.Set("type", "helm")
	result.ArtifactManagerMetadata.Set("maturity", maturity)
	result.ArtifactManagerMetadata.Set("name", chart.Metadata.Name)
	result.ArtifactManagerMetadata.Set("path", fi.Path)
	result.ArtifactManagerMetadata.Set("md5", fi.Checksums.Md5)
	result.ArtifactManagerMetadata.Set("sha1", fi.Checksums.Sha1)
	result.ArtifactManagerMetadata.Set("sha256", fi.Checksums.Sha256)
	result.ArtifactManagerMetadata.Set("uri", fi.URI)
	result.ArtifactManagerMetadata.Set("mimeType", fi.MimeType)
	result.ArtifactManagerMetadata.Set("downloadURI", fi.DownloadURI)
	result.ArtifactManagerMetadata.Set("createdBy", fi.CreatedBy)
	result.ArtifactManagerMetadata.Set("localRepository", localRepository)

	grpcplugins.Success("Done.")

	return nil
}

func (p *helmPushPlugin) UploadChartPackageToArtifactory(ctx context.Context, repository, chartName, chartPackagePath string, rtConfig grpcplugins.ArtifactoryConfig) (*http.Response, error) {
	f, err := os.Open(chartPackagePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	u, err := url.Parse(rtConfig.URL)
	if err != nil {
		return nil, err
	}

	u.Path = path.Join(u.Path, repository, chartName, filepath.Base(chartPackagePath))
	req, err := buildRequest(ctx, u.String(), f)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rtConfig.Token))
	req.Header.Set("User-Agent", "cds-helm-push-plugin-"+sdk.VERSION)

	grpcplugins.Logf("Pushing %s to %s...\n", filepath.Base(chartPackagePath), u.String())
	return p.HTTPClient.Do(req)
}

func buildRequest(ctx context.Context, url string, f *os.File) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "PUT", url, fileutils.GetUploadRequestContent(f))
	if err != nil {
		return nil, err
	}

	details, err := fileutils.GetFileDetails(f.Name(), true)
	if err != nil {
		return nil, err
	}

	length := strconv.FormatInt(details.Size, 10)
	req.Header.Set("Content-Length", length)
	req.Header.Set("X-Checksum-Sha1", details.Checksum.Sha1)
	req.Header.Set("X-Checksum-Md5", details.Checksum.Md5)
	if len(details.Checksum.Sha256) > 0 {
		req.Header.Set("X-Checksum", details.Checksum.Sha256)
	}

	return req, nil
}

func getIndexDownloader(client *cm.Client) cmhelm.IndexDownloader {
	return func() ([]byte, error) {
		resp, err := client.DownloadFile("index.yaml")
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != 200 {
			return nil, getChartmuseumError(b, resp.StatusCode)
		}
		return b, nil
	}
}
func getChartmuseumError(b []byte, code int) error {
	var er struct {
		Error string `json:"error"`
	}
	err := json.Unmarshal(b, &er)
	if err != nil || er.Error == "" {
		return fmt.Errorf("%d: could not properly parse response JSON: %s", code, string(b))
	}
	return fmt.Errorf("%d: %s", code, er.Error)
}
