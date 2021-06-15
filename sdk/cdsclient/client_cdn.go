package cdsclient

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/rockbears/log"
	"github.com/spf13/afero"

	"github.com/ovh/cds/sdk"
)

func (c *client) CDNItemDownload(ctx context.Context, cdnAddr string, hash string, itemType sdk.CDNItemType, fs afero.Fs, file File) error {
	currentRetry := 0
	var lastError error
	for i := 0; i <= c.config.Retry; i++ {
		currentRetry++
		f, err := fs.OpenFile(file.DestinationPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(file.Perm))
		if err != nil {
			newError := sdk.NewError(sdk.ErrUnknownError, fmt.Errorf("cannot create file (OpenFile) %s: %s", file.DestinationPath, err))
			return newError
		}

		reader, _, code, err := c.StreamNoRetry(ctx, c.HTTPNoTimeoutClient(), http.MethodGet, fmt.Sprintf("%s/item/%s/%s/download", cdnAddr, itemType, hash), nil, func(req *http.Request) {
			auth := "Bearer " + c.config.SessionToken
			req.Header.Add("Authorization", auth)
		})
		if code >= 500 {
			lastError = err
			_ = f.Close()
			continue
		}

		if err != nil {
			_ = f.Close()
			return err
		}

		md5Hash := md5.New()
		multiWriter := io.MultiWriter(md5Hash, f)

		if _, err := io.Copy(multiWriter, reader); err != nil {
			lastError = fmt.Errorf("unable to read cdn response: %v", err)
			log.Error(ctx, "%v", lastError)
			_ = f.Close()
			continue
		}

		md5S := hex.EncodeToString(md5Hash.Sum(nil))
		if md5S != file.MD5 {
			lastError = fmt.Errorf("ms5 doesn't match: Want %s Got %s", file.MD5, md5S)
			log.Error(ctx, "%v", lastError)
			_ = f.Close()
			continue
		}

		if err = f.Close(); err != nil {
			lastError = fmt.Errorf("unable to close file %s: %v", file.DestinationPath, err)
			log.Error(ctx, "%v", lastError)
			continue
		}
		return nil
	}
	return fmt.Errorf("unable to get data after %d retries: %v", currentRetry, lastError)
}

func (c *client) CDNItemStream(ctx context.Context, cdnAddr string, hash string, itemType sdk.CDNItemType) (io.Reader, error) {
	reader, _, code, err := c.Stream(ctx, c.HTTPNoTimeoutClient(), http.MethodGet, fmt.Sprintf("%s/item/%s/%s/download", cdnAddr, itemType, hash), nil, func(req *http.Request) {
		auth := "Bearer " + c.config.SessionToken
		req.Header.Add("Authorization", auth)
	})
	if err != nil {
		return nil, err
	}
	if code >= 400 {
		var stringBody string
		if reader != nil {
			body, _ := ioutil.ReadAll(reader)
			var errSdk sdk.Error
			if err := json.Unmarshal(body, &errSdk); err == nil && errSdk.Message != "" {
				stringBody = errSdk.Error()
			}
			if stringBody == "" {
				stringBody = string(body)
			}
		}
		return nil, newAPIError(fmt.Errorf("%s", stringBody))
	}
	return reader, err
}

func (c *client) CDNItemUpload(ctx context.Context, cdnAddr string, signature string, fs afero.Fs, path string) (time.Duration, error) {
	t0 := time.Now()

	var savedError error
	// as *File implement io.ReadSeeker, retry in c.Stream will be skipped
	for i := 0; i < c.config.Retry; i++ {
		f, err := fs.Open(path)
		if err != nil {
			return time.Since(t0), err
		}
		body, _, code, err := c.Stream(ctx, c.HTTPNoTimeoutClient(), http.MethodPost, fmt.Sprintf("%s/item/upload", cdnAddr), f, SetHeader("X-CDS-WORKER-SIGNATURE", signature))
		if err != nil {
			savedError = newAPIError(fmt.Errorf("unable to upload file, try %d: %v", i+1, err))
			time.Sleep(1 * time.Second)
			continue
		}
		if code >= 400 {
			bts, err := ioutil.ReadAll(body)
			if err != nil {
				return time.Since(t0), err
			}
			var errSdk sdk.Error
			if json.Unmarshal(bts, &errSdk); err != nil {
				return time.Since(t0), fmt.Errorf("%s", string(bts))
			}
			return time.Since(t0), fmt.Errorf("%v", errSdk)
		}
		savedError = nil
		break
	}
	return time.Since(t0), savedError
}
