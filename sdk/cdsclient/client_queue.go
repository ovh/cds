package cdsclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/sguiheux/go-coverage"
)

// shrinkQueue is used to shrink the polled queue 200% of the channel capacity (l)
// it returns as reference date the date of the last element in the shrinkked queue
func shrinkQueue(queue *sdk.WorkflowQueue, nbJobsToKeep int) time.Time {
	if len(*queue) == 0 {
		return time.Time{}
	}

	if nbJobsToKeep < 1 {
		nbJobsToKeep = 1
	}

	// nbJobsToKeep is by default the concurrent max worker provisionning.
	// we keep 2x this number
	nbJobsToKeep = nbJobsToKeep * 2

	queue.Sort()

	if len(*queue) > nbJobsToKeep {
		newQueue := (*queue)[:nbJobsToKeep]
		*queue = newQueue
	}
	t0 := time.Now()
	for _, q := range *queue {
		if q.Queued.Before(t0) {
			t0 = q.Queued
		}
	}
	return t0
}

func (c *client) QueuePolling(ctx context.Context, goRoutines *sdk.GoRoutines, jobs chan<- sdk.WorkflowNodeJobRun, errs chan<- error, delay time.Duration, ms ...RequestModifier) error {
	jobsTicker := time.NewTicker(delay)

	// This goroutine call the SSE route
	chanMessageReceived := make(chan sdk.WebsocketEvent, 10)
	chanMessageToSend := make(chan []sdk.WebsocketFilter, 10)
	goRoutines.Exec(ctx, "RequestWebsocket", func(ctx context.Context) {
		c.WebsocketEventsListen(ctx, goRoutines, chanMessageToSend, chanMessageReceived, errs)
	})
	chanMessageToSend <- []sdk.WebsocketFilter{{
		Type: sdk.WebsocketFilterTypeQueue,
	}}

	for {
		select {
		case <-ctx.Done():
			jobsTicker.Stop()
			if jobs != nil {
				close(jobs)
			}
			return ctx.Err()
		case wsEvent := <-chanMessageReceived:
			if jobs == nil {
				continue
			}
			if wsEvent.Event.EventType == "sdk.EventRunWorkflowJob" && wsEvent.Event.Status == sdk.StatusWaiting {
				var jobEvent sdk.EventRunWorkflowJob
				if err := sdk.JSONUnmarshal(wsEvent.Event.Payload, &jobEvent); err != nil {
					errs <- newError(fmt.Errorf("unable to unmarshal job %v: %v", wsEvent.Event.Payload, err))
					continue
				}
				job, err := c.QueueJobInfo(ctx, jobEvent.ID)
				// Do not log the error if the job does not exist
				if sdk.ErrorIs(err, sdk.ErrWorkflowNodeRunJobNotFound) {
					continue
				}

				if err != nil {
					errs <- newError(fmt.Errorf("unable to get job %v info: %v", jobEvent.ID, err))
					continue
				}
				// push the job in the channel
				if job.Status == sdk.StatusWaiting && job.BookedBy.Name == "" {
					job.Header["WS"] = "true"
					jobs <- *job
				}
			}
		case <-jobsTicker.C:
			if c.config.Verbose {
				fmt.Println("jobsTicker")
			}

			if jobs == nil {
				continue
			}

			ctxt, cancel := context.WithTimeout(ctx, 10*time.Second)
			queue := sdk.WorkflowQueue{}
			if _, err := c.GetJSON(ctxt, "/queue/workflows", &queue, ms...); err != nil && !sdk.ErrorIs(err, sdk.ErrUnauthorized) {
				errs <- newError(fmt.Errorf("unable to load jobs: %v", err))
				cancel()
				continue
			} else if sdk.ErrorIs(err, sdk.ErrUnauthorized) {
				cancel()
				continue
			}

			cancel()

			if c.config.Verbose {
				fmt.Println("Jobs Queue size: ", len(queue))
			}

			shrinkQueue(&queue, cap(jobs))
			for _, j := range queue {
				jobs <- j
			}
		}
	}
}

func (c *client) QueueWorkflowNodeJobRun(ms ...RequestModifier) ([]sdk.WorkflowNodeJobRun, error) {
	wJobs := []sdk.WorkflowNodeJobRun{}
	url, _ := url.Parse("/queue/workflows")
	if _, err := c.GetJSON(context.Background(), url.String(), &wJobs, ms...); err != nil {
		return nil, err
	}
	return wJobs, nil
}

func (c *client) QueueCountWorkflowNodeJobRun(since *time.Time, until *time.Time, modelType string, ratioService *int) (sdk.WorkflowNodeJobRunCount, error) {
	if since == nil {
		since = new(time.Time)
	}
	if until == nil {
		now := time.Now()
		until = &now
	}
	url, _ := url.Parse("/queue/workflows/count")
	q := url.Query()
	if ratioService != nil {
		q.Add("ratioService", fmt.Sprintf("%d", *ratioService))
	}
	if modelType != "" {
		q.Add("modelType", modelType)
	}
	url.RawQuery = q.Encode()

	countWJobs := sdk.WorkflowNodeJobRunCount{}
	_, _, err := c.GetJSONWithHeaders(url.String(),
		&countWJobs,
		SetHeader(RequestedIfModifiedSinceHeader, since.Format(time.RFC1123)),
		SetHeader("X-CDS-Until", until.Format(time.RFC1123)))
	return countWJobs, err
}

func (c *client) QueueTakeJob(ctx context.Context, job sdk.WorkflowNodeJobRun) (*sdk.WorkflowNodeJobRunData, error) {
	path := fmt.Sprintf("/queue/workflows/%d/take", job.ID)
	var info sdk.WorkflowNodeJobRunData
	if _, err := c.PostJSON(ctx, path, nil, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// QueueJobInfo returns information about a job
func (c *client) QueueJobInfo(ctx context.Context, id int64) (*sdk.WorkflowNodeJobRun, error) {
	path := fmt.Sprintf("/queue/workflows/%d/infos", id)
	var job sdk.WorkflowNodeJobRun

	if _, err := c.GetJSON(context.Background(), path, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

// QueueJobSendSpawnInfo sends a spawn info on a job
func (c *client) QueueJobSendSpawnInfo(ctx context.Context, id int64, in []sdk.SpawnInfo) error {
	path := fmt.Sprintf("/queue/workflows/%d/spawn/infos", id)
	_, err := c.PostJSON(ctx, path, &in, nil)
	return err
}

// QueueJobBook books a job for a Hatchery
func (c *client) QueueJobBook(ctx context.Context, id int64) (sdk.WorkflowNodeJobRunBooked, error) {
	var resp sdk.WorkflowNodeJobRunBooked
	path := fmt.Sprintf("/queue/workflows/%d/book", id)
	_, err := c.PostJSON(ctx, path, nil, &resp)
	return resp, err
}

func (c *client) QueueWorkflowRunResultsAdd(ctx context.Context, jobID int64, addRequest sdk.WorkflowRunResult) error {
	uri := fmt.Sprintf("/queue/workflows/%d/run/results", jobID)
	if _, err := c.PostJSON(ctx, uri, addRequest, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) QueueWorkflowRunResultCheck(ctx context.Context, jobID int64, runResultCheck sdk.WorkflowRunResultCheck) (int, error) {
	uri := fmt.Sprintf("/queue/workflows/%d/run/results/check", jobID)
	code, err := c.PostJSON(ctx, uri, runResultCheck, nil)
	return code, err
}

// QueueJobRelease release a job for a worker
func (c *client) QueueJobRelease(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/queue/workflows/%d/book", id)
	_, err := c.DeleteJSON(context.Background(), path, nil)
	return err
}

func (c *client) QueueSendResult(ctx context.Context, id int64, res sdk.Result) error {
	path := fmt.Sprintf("/queue/workflows/%d/result", id)
	b, err := json.Marshal(res)
	if err != nil {
		return newError(err)
	}
	result, _, code, err := c.Stream(ctx, c.HTTPNoTimeoutClient(), "POST", path, bytes.NewBuffer(b), nil)
	if err != nil {
		return err
	}
	defer result.Close()
	if code >= 300 {
		return newError(fmt.Errorf("unable to send job result. HTTP code error : %d", code))
	}
	return nil
}

func (c *client) QueueSendCoverage(ctx context.Context, id int64, report coverage.Report) error {
	path := fmt.Sprintf("/queue/workflows/%d/coverage", id)
	_, err := c.PostJSON(ctx, path, report, nil)
	return err
}

func (c *client) QueueSendUnitTests(ctx context.Context, id int64, report sdk.JUnitTestsSuites) error {
	path := fmt.Sprintf("/queue/workflows/%d/test", id)
	_, err := c.PostJSON(ctx, path, report, nil)
	return err
}

func (c *client) QueueSendVulnerability(ctx context.Context, id int64, report sdk.VulnerabilityWorkerReport) error {
	path := fmt.Sprintf("/queue/workflows/%d/vulnerability", id)
	_, err := c.PostJSON(ctx, path, report, nil)
	return err
}

func (c *client) QueueSendStepResult(ctx context.Context, id int64, res sdk.StepStatus) error {
	path := fmt.Sprintf("/queue/workflows/%d/step", id)
	_, err := c.PostJSON(ctx, path, res, nil)
	return err
}

func (c *client) QueueArtifactUpload(ctx context.Context, projectKey, integrationName string, nodeJobRunID int64, tag, filePath, fileType string) (bool, time.Duration, error) {
	t0 := time.Now()
	store := new(sdk.ArtifactsStore)
	uri := fmt.Sprintf("/project/%s/storage/%s", projectKey, integrationName)
	_, _ = c.GetJSON(ctx, uri, store)
	if store.TemporaryURLSupported {
		err := c.queueIndirectArtifactUpload(ctx, projectKey, integrationName, nodeJobRunID, tag, filePath, fileType)
		return true, time.Since(t0), err
	}
	err := c.queueDirectArtifactUpload(projectKey, integrationName, nodeJobRunID, tag, filePath, fileType)
	return false, time.Since(t0), err
}

// DEPRECATED
// TODO: remove this code after CDN would be mandatory
func (c *client) queueIndirectArtifactTempURL(ctx context.Context, projectKey, integrationName string, art *sdk.WorkflowNodeRunArtifact) error {
	var retryURL = 10
	var globalURLErr error
	uri := fmt.Sprintf("/project/%s/storage/%s/artifact/%s/url", projectKey, integrationName, art.Ref)

	for i := 0; i < retryURL; i++ {
		var code int
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		code, globalURLErr = c.PostJSON(ctx, uri, art, art)
		if code < 300 {
			cancel()
			break
		}
		cancel()
		time.Sleep(500 * time.Millisecond)
	}

	return globalURLErr
}

// DEPRECATED
// TODO: remove this code after CDN would be mandatory
func (c *client) queueIndirectArtifactTempURLPost(url string, content []byte) error {
	//Post the file to the temporary URL
	var retry = 10
	var globalErr error
	var body []byte
	for i := 0; i < retry; i++ {
		req, errRequest := http.NewRequest("PUT", url, bytes.NewBuffer(content))
		if errRequest != nil {
			return errRequest
		}

		var resp *http.Response
		resp, globalErr = http.DefaultClient.Do(req)
		if globalErr == nil {
			defer resp.Body.Close()

			var err error
			body, err = io.ReadAll(resp.Body)
			if err != nil {
				globalErr = err
				time.Sleep(1 * time.Second)
				continue
			}

			if resp.StatusCode >= 300 {
				globalErr = newError(fmt.Errorf("[%d] Unable to upload artifact: (HTTP %d) %s", i, resp.StatusCode, string(body)))
				time.Sleep(1 * time.Second)
				continue
			}

			break
		}
		time.Sleep(1 * time.Second)
	}

	return globalErr
}

// DEPRECATED
// TODO: remove this code after CDN would be mandatory
func (c *client) queueIndirectArtifactUpload(ctx context.Context, projectKey, integrationName string, nodeJobRunID int64, tag, filePath, fileType string) error {
	f, errop := os.Open(filePath)
	if errop != nil {
		return errop
	}
	defer f.Close()

	//File stat
	stat, errst := f.Stat()
	if errst != nil {
		return errst
	}

	sha512sum, err512 := sdk.FileSHA512sum(filePath)
	if err512 != nil {
		return err512
	}

	md5sum, errmd5 := sdk.FileMd5sum(filePath)
	if errmd5 != nil {
		return errmd5
	}

	_, name := filepath.Split(filePath)

	ref := base64.RawURLEncoding.EncodeToString([]byte(tag))
	art := sdk.WorkflowNodeRunArtifact{
		Name:                 name,
		Tag:                  tag,
		Ref:                  ref,
		Size:                 stat.Size(),
		Perm:                 uint32(stat.Mode().Perm()),
		MD5sum:               md5sum,
		SHA512sum:            sha512sum,
		Created:              time.Now(),
		WorkflowNodeJobRunID: nodeJobRunID,
	}

	if err := c.queueIndirectArtifactTempURL(ctx, projectKey, integrationName, &art); err != nil {
		return err
	}

	if c.config.Verbose {
		fmt.Printf("Uploading %s with to %s\n", art.Name, art.TempURL)
	}

	//Read the file once
	fileContent, errFileContent := io.ReadAll(f)
	if errFileContent != nil {
		return errFileContent
	}

	if err := c.queueIndirectArtifactTempURLPost(art.TempURL, fileContent); err != nil {
		// If we got a 401 error from the objectstore, probably because temporary URL is not
		// replicated on all cluster. Wait 5s before use it
		if strings.Contains(err.Error(), "401 Unauthorized: Temp URL invalid") {
			time.Sleep(5 * time.Second)
			if err := c.queueIndirectArtifactTempURLPost(art.TempURL, fileContent); err != nil {
				return err
			}
		}
		return err
	}

	//Try 50 times to make the callback
	var callbackErr error
	retry := 50
	for i := 0; i < retry; i++ {
		uri := fmt.Sprintf("/project/%s/storage/%s/artifact/%s/url/callback", projectKey, integrationName, art.Ref)
		ctxt, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, callbackErr = c.PostJSON(ctxt, uri, &art, nil)
		if callbackErr == nil {
			cancel()
			return nil
		}
		cancel()
	}

	return callbackErr
}

// DEPRECATED
// TODO: remove this code after CDN would be mandatory
func (c *client) queueDirectArtifactUpload(projectKey, integrationName string, nodeJobRunID int64, tag, filePath, fileType string) error {
	f, errop := os.Open(filePath)
	if errop != nil {
		return errop
	}
	defer f.Close()
	//File stat
	stat, errst := f.Stat()
	if errst != nil {
		return errst
	}

	sha512sum, err512 := sdk.FileSHA512sum(filePath)
	if err512 != nil {
		return err512
	}

	md5sum, errmd5 := sdk.FileMd5sum(filePath)
	if errmd5 != nil {
		return errmd5
	}

	//Read the file once
	fileContent, errFileContent := io.ReadAll(f)
	if errFileContent != nil {
		return errFileContent
	}

	_, name := filepath.Split(filePath)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, errc := writer.CreateFormFile(name, filepath.Base(filePath))
	if errc != nil {
		return errc
	}

	if _, err := io.Copy(part, bytes.NewBuffer(fileContent)); err != nil {
		return err
	}

	writer.WriteField("size", strconv.FormatInt(stat.Size(), 10))                 // nolint
	writer.WriteField("perm", strconv.FormatUint(uint64(stat.Mode().Perm()), 10)) // nolint
	writer.WriteField("md5sum", md5sum)                                           // nolint
	writer.WriteField("sha512sum", sha512sum)                                     // nolint
	writer.WriteField("nodeJobRunID", fmt.Sprintf("%d", nodeJobRunID))            // nolint

	if errclose := writer.Close(); errclose != nil {
		return errclose
	}

	var err error
	ref := base64.RawURLEncoding.EncodeToString([]byte(tag))
	uri := fmt.Sprintf("/project/%s/storage/%s/artifact/%s", projectKey, integrationName, ref)
	for i := 0; i <= c.config.Retry; i++ {
		var code int
		_, code, err = c.UploadMultiPart("POST", uri, body,
			SetHeader("Content-Disposition", "attachment; filename="+name),
			SetHeader("Content-Type", writer.FormDataContentType()))
		if err == nil {
			if code < 400 {
				return nil
			}
			err = newAPIError(fmt.Errorf("Error: HTTP code status %d", code))
		}
		time.Sleep(3 * time.Second)
	}

	return newError(fmt.Errorf("x%d: %v", c.config.Retry, err))
}

func (c *client) QueueWorkerCacheLink(ctx context.Context, jobID int64, tag string) (sdk.CDNItemLinks, error) {
	var result sdk.CDNItemLinks
	path := fmt.Sprintf("/queue/workflows/%d/cache/%s/links", jobID, tag)
	_, err := c.GetJSON(ctx, path, &result, nil)
	return result, err
}

func (c *client) QueueJobTag(ctx context.Context, jobID int64, tags []sdk.WorkflowRunTag) error {
	path := fmt.Sprintf("/queue/workflows/%d/tag", jobID)
	_, err := c.PostJSON(ctx, path, tags, nil)
	return err
}

func (c *client) QueueJobSetVersion(ctx context.Context, jobID int64, version sdk.WorkflowRunVersion) error {
	path := fmt.Sprintf("/queue/workflows/%d/version", jobID)
	_, err := c.PostJSON(ctx, path, version, nil)
	return err
}

//  STATIC FILES -----
// DEPRECATED
// TODO: remove this code after CDN would be mandatory
func (c *client) QueueStaticFilesUpload(ctx context.Context, projectKey, integrationName string, nodeJobRunID int64, name, entrypoint, staticKey string, tarContent io.Reader) (string, bool, time.Duration, error) {
	t0 := time.Now()
	staticFile := sdk.StaticFiles{
		EntryPoint:   entrypoint,
		StaticKey:    staticKey,
		Name:         name,
		NodeJobRunID: nodeJobRunID,
	}
	var store sdk.ArtifactsStore

	uri := fmt.Sprintf("/project/%s/storage/%s", projectKey, integrationName)
	_, _ = c.GetJSON(ctx, uri, &store)
	// TODO: to uncomment when swift will be available with auto-extract and temporary url middlewares
	// if store.TemporaryURLSupported {
	// 	publicURL, err := c.queueIndirectStaticFilesUpload(...)
	// }
	publicURL, err := c.queueDirectStaticFilesUpload(projectKey, integrationName, &staticFile, tarContent)
	return publicURL, false, time.Since(t0), err
}

// DEPRECATED
// TODO: remove this code after CDN would be mandatory
func (c *client) queueDirectStaticFilesUpload(projectKey, integrationName string, staticFile *sdk.StaticFiles, tarContent io.Reader) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, errc := writer.CreateFormFile("archive.tar", "archive.tar")
	if errc != nil {
		return "", errc
	}

	if _, err := io.Copy(part, tarContent); err != nil {
		return "", err
	}

	writer.WriteField("name", staticFile.Name)                                    // nolint
	writer.WriteField("entrypoint", staticFile.EntryPoint)                        // nolint
	writer.WriteField("static_key", staticFile.StaticKey)                         // nolint
	writer.WriteField("nodeJobRunID", fmt.Sprintf("%d", staticFile.NodeJobRunID)) // nolint

	if errclose := writer.Close(); errclose != nil {
		return "", errclose
	}

	var err error
	var respBody []byte
	uri := fmt.Sprintf("/project/%s/storage/%s/staticfiles/%s", projectKey, integrationName, url.PathEscape(staticFile.Name))
	var staticFileResp sdk.StaticFiles
	for i := 0; i <= c.config.Retry; i++ {
		var code int
		respBody, code, err = c.UploadMultiPart("POST", uri, body,
			SetHeader("Content-Disposition", "attachment; filename=archive.tar"),
			SetHeader("Content-Type", writer.FormDataContentType()))
		if err == nil && code < 300 {
			if err := sdk.JSONUnmarshal(respBody, &staticFileResp); err != nil {
				return "", newError(fmt.Errorf("unable to unmarshal body: %v: %v", string(respBody), err))
			}
			fmt.Printf("Files uploaded with public URL: %s\n", staticFileResp.PublicURL)
			return staticFileResp.PublicURL, nil
		}
		if c.config.Verbose {
			fmt.Printf("queueDirectStaticFilesUpload> Retry %d for status code %d : %v\n", i, code, err)
		}
		time.Sleep(3 * time.Second)
	}

	fmt.Printf("Files uploaded after retries with public URL: %s\n", staticFileResp.PublicURL)
	if err != nil {
		return "", newError(fmt.Errorf("cannot upload static files after %d retry: %v", c.config.Retry, err))
	}
	return staticFileResp.PublicURL, nil
}
