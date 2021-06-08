package cdsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/spf13/afero"

	"github.com/ovh/cds/sdk"
)

func (c *client) CDNItemDownload(ctx context.Context, cdnAddr string, hash string, itemType sdk.CDNItemType) (io.Reader, error) {
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
			if err := json.Unmarshal(body, &errSdk); err == nil {
				if errSdk.Message != "" {
					stringBody = errSdk.Error()
				}
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
		if _, _, _, err := c.Stream(ctx, c.HTTPNoTimeoutClient(), http.MethodPost, fmt.Sprintf("%s/item/upload", cdnAddr), f, SetHeader("X-CDS-WORKER-SIGNATURE", signature)); err != nil {
			savedError = newAPIError(fmt.Errorf("unable to upload file, try %d: %v", i+1, err))
			time.Sleep(1 * time.Second)
			continue
		}
		savedError = nil
		break
	}
	return time.Since(t0), savedError
}
