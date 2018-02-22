package cdsclient

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
)

func (c *client) QueuePolling(ctx context.Context, jobs chan<- sdk.WorkflowNodeJobRun, pbjobs chan<- sdk.PipelineBuildJob, errs chan<- error, delay time.Duration, graceTime int) error {
	t0 := time.Unix(0, 0)
	jobsTicker := time.NewTicker(delay)
	pbjobsTicker := time.NewTicker(delay)
	oldJobsTicker := time.NewTicker(delay * 60)

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
						jobs <- j
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

func (c *client) QueueCountWorkflowNodeJobRun() (sdk.WorkflowNodeJobRunCount, error) {
	countWJobs := sdk.WorkflowNodeJobRunCount{}
	_, err := c.GetJSON("/queue/workflows/count", &countWJobs)
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

func (c *client) queueIndirectArtifactUpload(id int64, tag, filePath string) error {
	f, errop := os.Open(filePath)
	if errop != nil {
		return errop
	}
	//File stat
	stat, errst := f.Stat()
	if errst != nil {
		return errst
	}

	//Read the file once
	fileContent, errFileContent := ioutil.ReadAll(f)
	if errFileContent != nil {
		return errFileContent
	}

	//Compute md5sum
	hash := md5.New()
	if _, errcopy := io.Copy(hash, bytes.NewBuffer(fileContent)); errcopy != nil {
		return errcopy
	}
	hashInBytes := hash.Sum(nil)[:16]
	md5sumStr := hex.EncodeToString(hashInBytes)
	_, name := filepath.Split(filePath)

	art := sdk.WorkflowNodeRunArtifact{
		Name:    name,
		Tag:     tag,
		Size:    stat.Size(),
		Perm:    uint32(stat.Mode().Perm()),
		MD5sum:  md5sumStr,
		Created: time.Now(),
	}

	uri := fmt.Sprintf("/queue/workflows/%d/artifact/%s/url", id, tag)
	if _, err := c.PostJSON(uri, &art, &art); err != nil {
		return err
	}

	if c.config.Verbose {
		fmt.Printf("Uploading %s with to %s\n", art.Name, art.TempURL)
	}

	//Post the file to the temporary URL
	var retry = 10
	var globalErr error
	var body []byte
	for i := 0; i < retry; i++ {
		req, errRequest := http.NewRequest("PUT", art.TempURL, bytes.NewBuffer(fileContent))
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
	}

	if globalErr != nil {
		return globalErr
	}

	//Try 50 times to make the callback
	var callbackErr error
	retry = 50
	for i := 0; i < retry; i++ {
		uri := fmt.Sprintf("/queue/workflows/%d/artifact/%s/url/callback", id, tag)
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
	//File stat
	stat, errst := f.Stat()
	if errst != nil {
		return errst
	}

	//Read the file once
	fileContent, errFileContent := ioutil.ReadAll(f)
	if errFileContent != nil {
		return errFileContent
	}

	//Compute md5sum
	hash := md5.New()
	if _, errcopy := io.Copy(hash, bytes.NewBuffer(fileContent)); errcopy != nil {
		return errcopy
	}
	hashInBytes := hash.Sum(nil)[:16]
	md5sumStr := hex.EncodeToString(hashInBytes)
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
	writer.WriteField("md5sum", md5sumStr)

	if errclose := writer.Close(); errclose != nil {
		return errclose
	}

	var err error
	uri := fmt.Sprintf("/queue/workflows/%d/artifact/%s", id, tag)
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
