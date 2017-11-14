package cdsclient

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
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
				if _, err := c.GetJSON("/queue/workflows", &queue, SetHeader("If-Modified-Since", t0.Format(time.RFC1123))); err != nil {
					errs <- sdk.WrapError(err, "Unable to load jobs")
				}
				// Gracetime to remove, see https://github.com/ovh/cds/issues/1214
				t0 = time.Now().Add(-time.Duration(graceTime) * time.Second)
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

func (c *client) Queue() ([]sdk.WorkflowNodeJobRun, []sdk.PipelineBuildJob, error) {
	wJobs := []sdk.WorkflowNodeJobRun{}
	if _, err := c.GetJSON("/queue/workflows", &wJobs); err != nil {
		return nil, nil, err
	}

	pbJobs := []sdk.PipelineBuildJob{}
	if _, err := c.GetJSON("/queue?status=all", &pbJobs); err != nil {
		return nil, nil, err
	}

	return wJobs, pbJobs, nil
}

func (c *client) QueueTakeJob(job sdk.WorkflowNodeJobRun, isBooked bool) (*worker.WorkflowNodeJobRunInfo, error) {
	in := worker.TakeForm{Time: time.Now()}
	if isBooked {
		in.BookedJobID = job.ID
	}

	path := fmt.Sprintf("/queue/workflows/%d/take", job.ID)
	var info worker.WorkflowNodeJobRunInfo

	if code, err := c.PostJSON(path, &in, &info); err != nil {
		return nil, err
	} else if code >= 400 {
		return nil, nil
	}

	return &info, nil
}

// QueueJobInfo returns information about a job
func (c *client) QueueJobInfo(id int64) (*sdk.WorkflowNodeJobRun, error) {
	path := fmt.Sprintf("/queue/workflows/%d/infos", id)
	var job sdk.WorkflowNodeJobRun

	if code, err := c.GetJSON(path, &job); err != nil {
		return nil, err
	} else if code >= 400 {
		return nil, fmt.Errorf("HTTP Error: %d", code)
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
	if code, err := c.PostJSON(path, &in, nil); err != nil {
		return err
	} else if code >= 400 {
		return fmt.Errorf("HTTP Error: %d", code)
	}
	return nil
}

// QueueJobBook books a job for a Hatchery
func (c *client) QueueJobBook(isWorkflowJob bool, id int64) error {
	path := fmt.Sprintf("/queue/workflows/%d/book", id)
	if !isWorkflowJob {
		// DEPRECATED code -> it's for pipelineBuildJob
		path = fmt.Sprintf("/queue/%d/book", id)
	}
	if code, err := c.PostJSON(path, nil, nil); err != nil {
		return err
	} else if code >= 400 {
		return fmt.Errorf("HTTP Error: %d", code)
	}
	return nil
}

func (c *client) QueueSendResult(id int64, res sdk.Result) error {
	path := fmt.Sprintf("/queue/workflows/%d/result", id)

	if code, err := c.PostJSON(path, res, nil); err != nil {
		return err
	} else if code >= 400 {
		return fmt.Errorf("HTTP Error: %d", code)
	}
	return nil
}

func (c *client) QueueArtifactUpload(id int64, tag, filePath string) error {
	fileForMD5, errop := os.Open(filePath)
	if errop != nil {
		return errop
	}
	//File stat
	stat, errst := fileForMD5.Stat()
	if errst != nil {
		return errst
	}
	//Compute md5sum
	hash := md5.New()
	if _, errcopy := io.Copy(hash, fileForMD5); errcopy != nil {
		return errcopy
	}
	hashInBytes := hash.Sum(nil)[:16]
	md5sumStr := hex.EncodeToString(hashInBytes)
	fileForMD5.Close()
	//Reopen the file because we already read it for md5
	fileReopen, erro := os.Open(filePath)
	if erro != nil {
		return erro
	}
	defer fileReopen.Close()
	_, name := filepath.Split(filePath)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, errc := writer.CreateFormFile(name, filepath.Base(filePath))
	if errc != nil {
		return errc
	}

	if _, err := io.Copy(part, fileReopen); err != nil {
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
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("x%d: %v", c.config.Retry, err)
}

func (c *client) QueueJobTag(jobID int64, tags []sdk.WorkflowRunTag) error {
	path := fmt.Sprintf("/queue/workflows/%d/tag", jobID)
	if code, err := c.PostJSON(path, tags, nil); err != nil {
		return err
	} else if code >= 400 {
		return fmt.Errorf("HTTP Error: %d", code)
	}
	return nil
}
