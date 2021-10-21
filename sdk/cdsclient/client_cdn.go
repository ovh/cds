package cdsclient

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rockbears/log"
	"github.com/spf13/afero"

	"github.com/ovh/cds/sdk"
)

func (c *client) CDNItemDownload(ctx context.Context, cdnAddr string, hash string, itemType sdk.CDNItemType, md5Sum string, writer io.WriteSeeker) error {
	currentRetry := 0
	var lastError error
	for i := 0; i <= c.config.Retry; i++ {
		currentRetry++
		if _, err := writer.Seek(0, io.SeekStart); err != nil {
			return sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to reset writer: %v", err)
		}

		reader, _, code, err := c.StreamNoRetry(ctx, c.HTTPNoTimeoutClient(), http.MethodGet, fmt.Sprintf("%s/item/%s/%s/download", cdnAddr, itemType, hash), nil, func(req *http.Request) {
			auth := "Bearer " + c.config.SessionToken
			req.Header.Add("Authorization", auth)
		})
		if code >= 500 {
			lastError = err
			continue
		}

		if err != nil {
			return err
		}

		md5Hash := md5.New()
		multiWriter := io.MultiWriter(md5Hash, writer)

		if _, err := io.Copy(multiWriter, reader); err != nil {
			lastError = fmt.Errorf("unable to read cdn response: %v", err)
			log.Error(ctx, "%v", lastError)
			continue
		}

		md5S := hex.EncodeToString(md5Hash.Sum(nil))
		if md5S != md5Sum {
			lastError = fmt.Errorf("ms5 doesn't match: Want %s Got %s", md5Sum, md5S)
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
			body, _ := io.ReadAll(reader)
			var errSdk sdk.Error
			if err := sdk.JSONUnmarshal(body, &errSdk); err == nil && errSdk.Message != "" {
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
			savedError = err
			time.Sleep(1 * time.Second)
			continue
		}
		if code >= 400 {
			bts, err := io.ReadAll(body)
			if err != nil {
				return time.Since(t0), err
			}
			if err := sdk.DecodeError(bts); err != nil {
				return time.Since(t0), err
			}
			return time.Since(t0), newAPIError(fmt.Errorf("HTTP %d", code))
		}
		savedError = nil
		break
	}
	return time.Since(t0), savedError
}
