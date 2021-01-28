package cdsclient

import (
	"context"
	"fmt"
	"github.com/rockbears/log"
	"github.com/spf13/afero"
	"io"
	"net/http"
	"time"

	"github.com/ovh/cds/sdk"
)

func (c *client) CDNItemDownload(ctx context.Context, cdnAddr string, hash string, itemType sdk.CDNItemType) (io.Reader, error) {
	reader, _, _, err := c.Stream(ctx, http.MethodGet, fmt.Sprintf("%s/item/%s/%s/download", cdnAddr, itemType, hash), nil, true, func(req *http.Request) {
		auth := "Bearer " + c.config.SessionToken
		req.Header.Add("Authorization", auth)
	})
	return reader, err
}

func (c *client) CDNItemUpload(ctx context.Context, cdnAddr string, signature string, fs afero.Fs, path string) (time.Duration, error) {
	log.Error(ctx, ">>>>>>CDN UPLOAD")
	t0 := time.Now()

	var savedError error
	// as *File implement io.ReadSeeker, retry in c.Stream will be skipped
	for i := 0; i < c.config.Retry; i++ {
		log.Error(ctx, ">>>>>>CDN OPEN FILE")
		f, err := fs.Open(path)
		if err != nil {
			log.Error(ctx, ">>>>>>CDN OPEN FILE :%v", err)
			return time.Since(t0), sdk.WithStack(err)
		}
		log.Error(ctx, ">>>>>>CDN YES: %+v", f)
		_, _, _, err = c.Request(ctx, http.MethodPost, fmt.Sprintf("%s/item/upload", cdnAddr), f, SetHeader("X-CDS-WORKER-SIGNATURE", signature))
		if err != nil {
			log.Error(ctx, ">>>>>>CDN UPLOAD FAILD: %v", err)
			savedError = sdk.WrapError(err, "unable to upload file, try %d", i+1)
			time.Sleep(1 * time.Second)
			continue
		}
		savedError = nil
		break
	}
	return time.Since(t0), sdk.WithStack(savedError)
}
