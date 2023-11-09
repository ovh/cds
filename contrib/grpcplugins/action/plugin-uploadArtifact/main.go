package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/srerickson/checksum"

	"github.com/ovh/cds/contrib/grpcplugins"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/glob"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

/* Inside contrib/grpcplugins/action
 */

type runActionUploadArtifactPlugin struct {
	actionplugin.Common
}

func main() {
	actPlugin := runActionUploadArtifactPlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
}

func (actPlugin *runActionUploadArtifactPlugin) Manifest(_ context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "uploadArtifact",
		Author:      "François SAMIN <francois.samin@corp.ovh.com>",
		Description: "This uploads artifacts from your workflow allowing you to share data between jobs and store data once a workflow is complete.",
		Version:     sdk.VERSION,
	}, nil
}

func (actPlugin *runActionUploadArtifactPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	res := &actionplugin.ActionResult{
		Status: sdk.StatusSuccess,
	}
	path := q.GetOptions()["path"]
	ifNoFilesFound := q.GetOptions()["if-no-files-found"]

	workDirs, err := grpcplugins.GetWorkerDirectories(ctx, &actPlugin.Common)
	if err != nil {
		err := fmt.Errorf("unable to get working directory: %v", err)
		res.Status = sdk.StatusFail
		res.Status = err.Error()
		return res, err
	}

	var dirFS = os.DirFS(workDirs.WorkingDir)

	if err := perform(ctx, &actPlugin.Common, dirFS, path, ifNoFilesFound); err != nil {
		res.Status = sdk.StatusFail
		res.Status = err.Error()
		return res, err
	}

	return res, nil
}

func perform(ctx context.Context, c *actionplugin.Common, dirFS fs.FS, path, ifNoFilesFound string) error {
	results, err := glob.Glob(dirFS, ".", path)
	if err != nil {
		return err
	}

	var message string
	switch len(results) {
	case 0:
		message = fmt.Sprintf("No files were found with the provided path: %q. No artifacts will be uploaded.", path)
	case 1:
		message = fmt.Sprintf("With the provided pattern %q, there will be %d file uploaded.", path, len(results))
	default:
		message = fmt.Sprintf("With the provided pattern %q, there will be %d files uploaded.", path, len(results))
	}

	if len(results) == 0 {
		switch strings.ToUpper(ifNoFilesFound) {
		case "ERROR":
			grpcplugins.Error(message)
			return errors.New("no files were found")
		case "WARN":
			grpcplugins.Warn(message)
		default:
			grpcplugins.Log(message)
		}
	} else {
		grpcplugins.Log(message)
	}

	var files []string
	var sizes = map[string]int64{}
	var permissions = map[string]os.FileMode{}
	var openFiles = map[string]fs.File{}
	for _, r := range results {
		files = append(files, r.Path)
		f, err := dirFS.Open(r.Path)
		if err != nil {
			grpcplugins.Errorf("unable to open file %q: %v", r.Path, err)
			continue
		}
		stat, err := f.Stat()
		if err != nil {
			grpcplugins.Errorf("unable to stat file %q: %v", r.Path, err)
			f.Close()
			continue
		}
		defer f.Close()
		sizes[r.Path] = stat.Size()
		permissions[r.Path] = stat.Mode()
		openFiles[r.Path] = f
	}

	checksums, err := checksums(ctx, dirFS, files...)
	if err != nil {
		return err
	}

	for _, r := range results {
		message = fmt.Sprintf("\nStarting upload of file %q as %q \n  Size: %d, MD5: %s, sh1: %s, SHA256: %s, Mode: %v", r.Path, r.Result, sizes[r.Path], checksums[r.Path].md5, checksums[r.Path].sha1, checksums[r.Path].sha256, permissions[r.Path])
		grpcplugins.Log(message)

		// Create run result at status "pending"
		var runResultRequest = workerruntime.V2RunResultRequest{
			RunResult: &sdk.V2WorkflowRunResult{
				IssuedAt: time.Now(),
				Type:     sdk.V2WorkflowRunResultTypeGeneric,
				Status:   sdk.V2WorkflowRunResultStatusPending,
				Detail: sdk.V2WorkflowRunResultDetail{
					Data: sdk.V2WorkflowRunResultGenericDetail{
						Name:   r.Result,
						Size:   sizes[r.Path],
						Mode:   permissions[r.Path],
						MD5:    checksums[r.Path].md5,
						SHA1:   checksums[r.Path].sha1,
						SHA256: checksums[r.Path].sha256,
					},
				},
			},
		}

		response, err := grpcplugins.CreateRunResult(ctx, c, &runResultRequest)
		if err != nil {
			grpcplugins.Error(err.Error())
			return err
		}

		// Upload the file to an artifactory or CDN
		reader, ok := openFiles[r.Path].(io.ReadSeeker)
		var d time.Duration
		var item *sdk.CDNItem
		if ok {
			item, d, err = CDNItemUpload(ctx, c, response.CDNAddress, response.CDNSignature, reader)
			if err != nil {
				grpcplugins.Error("An error occured during file upload upload: " + err.Error())
				continue
			}

		} else {
			// unable to cast the file
			return fmt.Errorf("unable to cast reader")
		}

		// Update the run result status
		runResultRequest = workerruntime.V2RunResultRequest{RunResult: response.RunResult}
		runResultRequest.RunResult.ArtifactManagerMetadata = sdk.V2WorkflowRunResultArtifactManagerMetadataFromCDNItemLink(sdk.CDNItemLink{CDNHttpURL: response.CDNAddress, Item: *item})
		updateResponse, err := grpcplugins.UpdateRunResult(ctx, c, &runResultRequest)
		if err != nil {
			grpcplugins.Error(err.Error())
			return err
		}

		grpcplugins.Logf("  %d bytes uploaded in %.3fs", sizes[r.Path], d.Seconds())

		if _, err := updateResponse.RunResult.GetDetail(); err != nil {
			grpcplugins.Error(err.Error())
			return err
		}

		grpcplugins.Logf("  Result %s (%s) created", updateResponse.RunResult.Name(), updateResponse.RunResult.ID)
	}

	return nil
}

type checksumResult struct {
	md5    string
	sha1   string
	sha256 string
}

func checksums(ctx context.Context, dir fs.FS, path ...string) (map[string]checksumResult, error) {
	pipe, err := checksum.NewPipe(dir, checksum.WithCtx(ctx), checksum.WithMD5(), checksum.WithSHA1(), checksum.WithSHA256())
	if err != nil {
		return nil, err
	}

	go func() {
		for _, p := range path {
			if err := pipe.Add(p); err != nil {
				grpcplugins.Error(p)
			}
		}
		pipe.Close()
	}()

	var result = map[string]checksumResult{}

	for out := range pipe.Out() {
		md5, err := out.Sum(checksum.MD5)
		if err != nil {
			grpcplugins.Error(err.Error())
			continue
		}
		sha1, err := out.Sum(checksum.SHA1)
		if err != nil {
			grpcplugins.Error(err.Error())
			continue
		}
		sha256, err := out.Sum(checksum.SHA256)
		if err != nil {
			grpcplugins.Error(err.Error())
			continue
		}
		result[out.Path()] = checksumResult{
			md5:    hex.EncodeToString(md5),
			sha1:   hex.EncodeToString(sha1),
			sha256: hex.EncodeToString(sha256),
		}
	}

	return result, nil
}

func CDNItemUpload(ctx context.Context, c *actionplugin.Common, cdnAddr string, signature string, reader io.ReadSeeker) (*sdk.CDNItem, time.Duration, error) {
	t0 := time.Now()

	for i := 0; i < 3; i++ {
		reader.Seek(0, io.SeekStart)

		req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/item/upload", cdnAddr), reader)
		if err != nil {
			return nil, time.Since(t0), err
		}
		req.Header.Set("X-CDS-WORKER-SIGNATURE", signature)

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, time.Since(t0), err
		}

		if resp.StatusCode >= 200 && resp.StatusCode <= 204 {
			btes, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, time.Since(t0), err
			}
			var item sdk.CDNItem
			if err := sdk.JSONUnmarshal(btes, &item); err != nil {
				return nil, time.Since(t0), err
			}
			return &item, time.Since(t0), nil
		} else {
			bts, err := io.ReadAll(resp.Body)
			if err != nil {
				grpcplugins.Error(err.Error())
			}
			if err := sdk.DecodeError(bts); err != nil {
				grpcplugins.Error(err.Error())
			}
			grpcplugins.Error(fmt.Sprintf("HTTP %d", resp.StatusCode))
		}

		grpcplugins.Log("retrying file upload...")
	}

	return nil, time.Since(t0), errors.New("unable to upload artifact")
}
