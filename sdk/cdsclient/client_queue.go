package cdsclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
)

func (c *client) QueuePolling(ctx context.Context, jobs chan<- sdk.WorkflowNodeJobRun, pbjobs chan<- sdk.PipelineBuildJob, errs chan<- error, delay time.Duration, graceTime int, exceptWfJobID *int64) error {
	t0 := time.Unix(0, 0)
	jobsTicker := time.NewTicker(delay)
	pbjobsTicker := time.NewTicker(delay)
	oldJobsTicker := time.NewTicker(delay * 10)

	for {
		select {
		case <-ctx.Done():
			jobsTicker.Stop()
			pbjobsTicker.Stop()
			oldJobsTicker.Stop()
			if jobs != nil {
				close(jobs)
			}
			if pbjobs != nil {
				close(pbjobs)
			}
			return ctx.Err()
		case <-oldJobsTicker.C:
			if c.config.Verbose {
				fmt.Println("oldJobsTicker")
			}

			if jobs != nil {
				queue := []sdk.WorkflowNodeJobRun{}
				if _, err := c.GetJSON("/queue/workflows", &queue); err != nil {
					errs <- sdk.WrapError(err, "Unable to load old jobs")
				}
				for _, j := range queue {
					jobs <- j
				}
			}
		case <-jobsTicker.C:
			if c.config.Verbose {
				fmt.Println("jobsTicker")
			}

			if jobs != nil {
				queue := []sdk.WorkflowNodeJobRun{}
				_, header, _, errReq := c.RequestJSON(http.MethodGet, "/queue/workflows", nil, &queue, SetHeader(RequestedIfModifiedSinceHeader, t0.Format(time.RFC1123)))
				if errReq != nil {
					errs <- sdk.WrapError(errReq, "Unable to load jobs")
					continue
				}

				apiTimeHeader := header.Get(ResponseAPITimeHeader)
				apiTime, errParse := time.Parse(time.RFC3339, apiTimeHeader)
				if errParse != nil {
					errs <- sdk.WrapError(errParse, "Unable to load jobs, failed to parse API Time")
					continue
				}

				// Gracetime to remove, see https://github.com/ovh/cds/issues/1214
				t0 = apiTime.Add(-time.Duration(graceTime) * time.Second)

				for _, j := range queue {
					// if there is a grace time, check it
					if j.QueuedSeconds > int64(graceTime) {
						if c.config.Verbose {
							fmt.Printf("job %d send on chan\n", j.ID)
						}
						// Useful to not relaunch a job on a worker with bad requirements
						if exceptWfJobID == nil || *exceptWfJobID != j.ID {
							jobs <- j
						}
					} else {
						if c.config.Verbose {
							fmt.Printf("job %d too new\n", j.ID)
						}
					}
				}
			}
		case <-pbjobsTicker.C:
			if c.config.Verbose {
				fmt.Println("pbjobsTicker")
			}

			if pbjobs != nil {
				queue := []sdk.PipelineBuildJob{}
				if _, err := c.GetJSON("/queue?status=all", &queue); err != nil {
					errs <- sdk.WrapError(err, "Unable to load pipeline build jobs")
				}
				for _, j := range queue {
					pbjobs <- j
				}
			}
		}
	}
}

func (c *client) QueueWorkflowNodeJobRun() ([]sdk.WorkflowNodeJobRun, error) {
	wJobs := []sdk.WorkflowNodeJobRun{}
	if _, err := c.GetJSON("/queue/workflows", &wJobs); err != nil {
		return nil, err
	}
	return wJobs, nil
}

func (c *client) QueueCountWorkflowNodeJobRun(since *time.Time, until *time.Time) (sdk.WorkflowNodeJobRunCount, error) {
	if since == nil {
		since = new(time.Time)
	}
	if until == nil {
		now := time.Now()
		until = &now
	}
	countWJobs := sdk.WorkflowNodeJobRunCount{}
	_, _, err := c.GetJSONWithHeaders("/queue/workflows/count", &countWJobs, SetHeader(RequestedIfModifiedSinceHeader, since.Format(time.RFC1123)), SetHeader("X-CDS-Until", until.Format(time.RFC1123)))
	return countWJobs, err
}

func (c *client) QueuePipelineBuildJob() ([]sdk.PipelineBuildJob, error) {
	pbJobs := []sdk.PipelineBuildJob{}
	if _, err := c.GetJSON("/queue?status=all", &pbJobs); err != nil {
		return nil, err
	}
	return pbJobs, nil
}

func (c *client) QueueTakeJob(job sdk.WorkflowNodeJobRun, isBooked bool) (*worker.WorkflowNodeJobRunInfo, error) {
	in := sdk.WorkerTakeForm{
		Time:    time.Now(),
		Version: sdk.VERSION,
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}
	if isBooked {
		in.BookedJobID = job.ID
	}

	path := fmt.Sprintf("/queue/workflows/%d/take", job.ID)
	var info worker.WorkflowNodeJobRunInfo

	if _, err := c.PostJSON(path, &in, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// QueueJobInfo returns information about a job
func (c *client) QueueJobInfo(id int64) (*sdk.WorkflowNodeJobRun, error) {
	path := fmt.Sprintf("/queue/workflows/%d/infos", id)
	var job sdk.WorkflowNodeJobRun

	if _, err := c.GetJSON(path, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

// QueueJobSendSpawnInfo sends a spawn info on a job
func (c *client) QueueJobSendSpawnInfo(isWorkflowJob bool, id int64, in []sdk.SpawnInfo) error {
	path := fmt.Sprintf("/queue/workflows/%d/spawn/infos", id)
	if !isWorkflowJob {
		// DEPRECATED code -> it's for pipelineBuildJob
		path = fmt.Sprintf("/queue/%d/spawn/infos", id)
	}

	_, err := c.PostJSON(path, &in, nil)
	return err
}

// QueueJobIncAttempts add hatcheryID that cannot run this job and return the spawn attempts list
func (c *client) QueueJobIncAttempts(jobID int64) ([]int64, error) {
	var spawnAttempts []int64
	path := fmt.Sprintf("/queue/workflows/%d/attempt", jobID)
	_, err := c.PostJSON(path, nil, &spawnAttempts)
	return spawnAttempts, err
}

// QueueJobBook books a job for a Hatchery
func (c *client) QueueJobBook(isWorkflowJob bool, id int64) error {
	path := fmt.Sprintf("/queue/workflows/%d/book", id)
	if !isWorkflowJob {
		// DEPRECATED code -> it's for pipelineBuildJob
		path = fmt.Sprintf("/queue/%d/book", id)
	}
	_, err := c.PostJSON(path, nil, nil)
	return err
}

// QueueJobRelease release a job for a worker
func (c *client) QueueJobRelease(isWorkflowJob bool, id int64) error {
	path := fmt.Sprintf("/queue/workflows/%d/book", id)
	if !isWorkflowJob {
		// DEPRECATED code -> it's for pipelineBuildJob
		return fmt.Errorf("Not implemented")
	}

	_, err := c.DeleteJSON(path, nil)
	return err
}

func (c *client) QueueSendResult(id int64, res sdk.Result) error {
	path := fmt.Sprintf("/queue/workflows/%d/result", id)
	_, err := c.PostJSON(path, res, nil)
	return err
}

func (c *client) QueueArtifactUpload(id int64, tag, filePath string) (bool, time.Duration, error) {
	t0 := time.Now()
	store := new(sdk.ArtifactsStore)
	_, _ = c.GetJSON("/artifact/store", store)
	if store.TemporaryURLSupported {
		err := c.queueIndirectArtifactUpload(id, tag, filePath)
		return true, time.Since(t0), err
	}
	err := c.queueDirectArtifactUpload(id, tag, filePath)
	return false, time.Since(t0), err
}

func (c *client) queueIndirectArtifactTempURL(id int64, art *sdk.WorkflowNodeRunArtifact) error {
	var retryURL = 10
	var globalURLErr error
	uri := fmt.Sprintf("/queue/workflows/%d/artifact/%s/url", id, art.Ref)

	for i := 0; i < retryURL; i++ {
		var code int
		code, globalURLErr = c.PostJSON(uri, art, art)
		if code < 300 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if globalURLErr != nil {
		return globalURLErr
	}
	return nil
}

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
			body, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				globalErr = err
				continue
			}

			if resp.StatusCode >= 300 {
				globalErr = fmt.Errorf("[%d] Unable to upload artifact: (HTTP %d) %s", i, resp.StatusCode, string(body))
				continue
			}

			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	return globalErr
}

func (c *client) queueIndirectArtifactUpload(id int64, tag, filePath string) error {
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
		Name:      name,
		Tag:       tag,
		Ref:       ref,
		Size:      stat.Size(),
		Perm:      uint32(stat.Mode().Perm()),
		MD5sum:    md5sum,
		SHA512sum: sha512sum,
		Created:   time.Now(),
	}

	if err := c.queueIndirectArtifactTempURL(id, &art); err != nil {
		return err
	}

	if c.config.Verbose {
		fmt.Printf("Uploading %s with to %s\n", art.Name, art.TempURL)
	}

	//Read the file once
	fileContent, errFileContent := ioutil.ReadAll(f)
	if errFileContent != nil {
		return errFileContent
	}

	if err := c.queueIndirectArtifactTempURLPost(art.TempURL, fileContent); err != nil {
		// If we got a 401 error from the objectstore, ask for a fresh temporary url and repost the artifact
		if strings.Contains(err.Error(), "401 Unauthorized: Temp URL invalid") {
			if err := c.queueIndirectArtifactTempURL(id, &art); err != nil {
				return err
			}

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
		uri := fmt.Sprintf("/queue/workflows/%d/artifact/%s/url/callback", id, art.Ref)
		_, callbackErr = c.PostJSON(uri, &art, nil)
		if callbackErr == nil {
			return nil
		}
	}

	return callbackErr
}

func (c *client) queueDirectArtifactUpload(id int64, tag, filePath string) error {
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
	fileContent, errFileContent := ioutil.ReadAll(f)
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

	writer.WriteField("size", strconv.FormatInt(stat.Size(), 10))
	writer.WriteField("perm", strconv.FormatUint(uint64(stat.Mode().Perm()), 10))
	writer.WriteField("md5sum", md5sum)
	writer.WriteField("sha512sum", sha512sum)

	if errclose := writer.Close(); errclose != nil {
		return errclose
	}

	var err error
	ref := base64.RawURLEncoding.EncodeToString([]byte(tag))
	uri := fmt.Sprintf("/queue/workflows/%d/artifact/%s", id, ref)
	for i := 0; i <= c.config.Retry; i++ {
		var code int
		_, code, err = c.UploadMultiPart("POST", uri, body,
			SetHeader("Content-Disposition", "attachment; filename="+name),
			SetHeader("Content-Type", writer.FormDataContentType()))
		if err == nil && code < 300 {
			return nil
		}
		time.Sleep(3 * time.Second)
	}

	return fmt.Errorf("x%d: %v", c.config.Retry, err)
}

func (c *client) QueueJobTag(jobID int64, tags []sdk.WorkflowRunTag) error {
	path := fmt.Sprintf("/queue/workflows/%d/tag", jobID)
	_, err := c.PostJSON(path, tags, nil)
	return err
}

func (c *client) QueueServiceLogs(logs []sdk.ServiceLog) error {
	status, err := c.PostJSON("/queue/workflows/log/service", logs, nil)
	if status >= 400 {
		return fmt.Errorf("Error: HTTP code %d", status)
	}

	return err
}
