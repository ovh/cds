package download

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/rockbears/log"
	"github.com/spf13/afero"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

type Conf struct {
	Directory             string
	DownloadFromGitHub    bool
	ArtifactoryURL        string
	ArtifactoryPath       string
	ArtifactoryRepository string
	ArtifactoryToken      string
	SupportedOSArch       []string
	ForceDownloadGitHub   bool
}

func Init(ctx context.Context, conf Conf) error {
	// Checking downloadable binaries
	sdk.InitSupportedOSArch(conf.SupportedOSArch)
	ensureWorkerBinary(ctx, conf)
	resources := sdk.AllDownloadableResourcesWithAvailability(conf.Directory)
	var hasWorker, hasCtl, hasEngine bool
	for _, r := range resources {
		if r.Available != nil && *r.Available {
			switch r.Name {
			case "worker":
				hasWorker = true
			case "cdsctl":
				hasCtl = true
			case "engine":
				hasEngine = true
			}
		}
	}
	if !hasEngine {
		log.Error(ctx, "engine is unavailable for download, this may lead to a poor user experience. Please check your configuration file or the %s directory", conf.Directory)
	}
	if !hasCtl {
		log.Error(ctx, "cdsctl is unavailable for download, this may lead to a poor user experience. Please check your configuration file or the %s directory", conf.Directory)
	}
	if !hasWorker {
		// If no worker, let's exit because CDS for run anything
		log.Error(ctx, "worker is unavailable for download. Please check your configuration file or the %s directory", conf.Directory)
		return errors.New("worker binary unavailable")
	}
	return nil
}

func ensureWorkerBinary(ctx context.Context, conf Conf) error {
	if !conf.DownloadFromGitHub {
		if err := CheckBinary(ctx, conf, "worker", sdk.GOOS, sdk.GOARCH, ""); err != nil {
			return err
		}
	}

	// conf.DownloadFromGitHub true
	// if worker not here, we ask user to download it from GitHub
	filename := sdk.BinaryFilename("worker", sdk.GOOS, sdk.GOARCH, "")
	if sdk.IsDownloadedBinary(conf.Directory, filename) {
		return nil
	}

	if conf.ForceDownloadGitHub {
		return CheckBinary(ctx, conf, "worker", sdk.GOOS, sdk.GOARCH, "")
	}

	ask := fmt.Sprintf("Worker binary %q does not exist into %v\nWhat do you want to do?", filename, conf.Directory)

	answerDoNothing := "Do nothing - exit now"
	answerDownload := "Download from GitHub"
	opts := []string{answerDoNothing, answerDownload}

	selected := cli.AskChoice(ask, opts...)

	switch opts[selected] {
	case answerDoNothing:
		return nil
	case answerDownload:
		if err := CheckBinary(ctx, conf, "worker", sdk.GOOS, sdk.GOARCH, ""); err != nil {
			return err
		}
	}
	return nil
}

// CheckBinary checks if binary exist and download it if needed
func CheckBinary(ctx context.Context, conf Conf, name, osName, arch, variant string) error {
	filename := sdk.BinaryFilename(name, osName, arch, variant)
	if sdk.IsDownloadedBinary(conf.Directory, filename) {
		return nil
	}

	var filenameToDownload string
	if name == "worker" {
		filenameToDownload = "cds-worker-all.tar.gz"
	} else {
		filenameToDownload = filename
	}

	if conf.DownloadFromGitHub {
		log.Info(ctx, "downloading %v from GitHub into %v", filenameToDownload, conf.Directory)
		if err := sdk.DownloadFromGitHub(ctx, conf.Directory, filenameToDownload, sdk.VERSION); err != nil {
			return err
		}
	} else if conf.ArtifactoryURL != "" {
		log.Info(ctx, "downloading %v from Artifactory into %v", filenameToDownload, conf.Directory)
		if err := GetBinaryFromArtifactory(conf, filenameToDownload); err != nil {
			return err
		}
	}

	if strings.HasSuffix(filenameToDownload, ".tar.gz") {
		log.Info(ctx, "untar %v", filenameToDownload)
		srcPath := path.Join(conf.Directory, filenameToDownload)
		src, err := os.Open(srcPath)
		if err != nil {
			return sdk.WrapError(err, "unable to open source file %s", srcPath)
		}
		defer src.Close()
		if err := sdk.UntarGz(afero.NewOsFs(), conf.Directory, src); err != nil {
			return sdk.WrapError(err, "unarchive %s failed", filenameToDownload)
		}
	}

	return nil
}

func GetBinaryFromArtifactory(conf Conf, filename string) error {
	artiClient, err := sdk.NewArtifactoryClient(conf.ArtifactoryURL, conf.ArtifactoryToken)
	if err != nil {
		return sdk.WrapError(err, "unable to create artifactory client")
	}

	params := services.NewDownloadParams()
	params.Pattern = fmt.Sprintf("%s/%s/%s/%s", conf.ArtifactoryRepository, conf.ArtifactoryPath, sdk.VERSION, filename)
	// target must have a '/' at the end. We ensure to have this '/' (and only one)
	params.Target = strings.TrimSuffix(conf.Directory, "/") + "/"
	params.Flat = true

	summary, err := artiClient.DownloadFilesWithSummary(params)
	if err != nil || summary.TotalFailed > 0 {
		return sdk.WrapError(err, "unable to download files %s from artifactory", params.Pattern)
	}
	defer summary.Close()
	return nil
}
