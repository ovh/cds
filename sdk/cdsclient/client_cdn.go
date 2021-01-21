package cdsclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/ovh/cds/sdk"
)

func (c *client) CDNArtifactDownload(ctx context.Context, cdnAddr string, hash string, w io.Writer) error {
	reader, _, _, err := c.Stream(ctx, http.MethodGet, fmt.Sprintf("%s/item/%s/%s/download", cdnAddr, sdk.CDNTypeItemArtifact, hash), nil, true, func(req *http.Request) {
		auth := "Bearer " + c.config.SessionToken
		req.Header.Add("Authorization", auth)
	})
	if err != nil {
		return err
	}
	defer reader.Close()
	_, err = io.Copy(w, reader)
	return err
}

func (c *client) CDNArtifactUpdload(ctx context.Context, cdnAddr string, signature string, path string) (time.Duration, error) {
	t0 := time.Now()

	var savedError error
	for i := 0; i < c.config.Retry; i++ {
		f, err := os.Open(path)
		if err != nil {
			return time.Since(t0), sdk.WithStack(err)
		}
		_, _, _, err = c.Request(ctx, http.MethodPost, fmt.Sprintf("%s/item/upload", cdnAddr), f, SetHeader("X-CDS-WORKER-SIGNATURE", signature))
		if err != nil {
			savedError = sdk.WrapError(err, "unable to upload file, try %d", i+1)
			time.Sleep(1 * time.Second)
			continue
		}
		savedError = nil
		break
	}
	return time.Since(t0), sdk.WithStack(savedError)
}
