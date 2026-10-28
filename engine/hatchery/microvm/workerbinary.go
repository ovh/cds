package microvm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

// workerBinaryCache holds the sha256 of the worker binaries served by the
// API, by os/arch, for one API build. The platform caches guest files by
// sha256 on each node, so the binary is downloaded once per node and per
// API build instead of at every boot.
type workerBinaryCache struct {
	mu      sync.Mutex
	build   string
	sha256s map[string]string
}

// workerDownloadURL is the API route serving the worker binary
func workerDownloadURL(api, osArch string) string {
	return api + "/download/worker/" + osArch
}

// workerSHA256 returns the sha256 of the worker binary served by the API for
// osArch. The binary is only downloaded when the API build changed since the
// last call.
func (h *HatcheryMicroVM) workerSHA256(ctx context.Context, api string, insecure bool, osArch string) (string, error) {
	httpClient := cdsclient.NewHTTPClient(2*time.Minute, insecure)

	var version sdk.Version
	if err := httpGet(ctx, httpClient, api+"/mon/version", func(body io.Reader) error {
		return json.NewDecoder(body).Decode(&version)
	}); err != nil {
		return "", err
	}
	// Development builds share a version: the git hash and the build time tell them apart
	build := version.Version + "/" + version.GitHash + "/" + version.BuildTime

	c := &h.workerBinaries
	// Held during the download: concurrent spawns wait for the hash instead of
	// downloading the binary again
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.build != build {
		c.build = build
		c.sha256s = map[string]string{}
	}
	if sum, ok := c.sha256s[osArch]; ok {
		return sum, nil
	}

	hash := sha256.New()
	if err := httpGet(ctx, httpClient, workerDownloadURL(api, osArch), func(body io.Reader) error {
		_, err := io.Copy(hash, body)
		return err
	}); err != nil {
		return "", err
	}
	sum := hex.EncodeToString(hash.Sum(nil))
	c.sha256s[osArch] = sum
	return sum, nil
}

func httpGet(ctx context.Context, httpClient *http.Client, url string, read func(io.Reader) error) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return sdk.WithStack(err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return sdk.WrapError(err, "cannot get %s", url)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return sdk.WithStack(fmt.Errorf("cannot get %s: HTTP %d", url, resp.StatusCode))
	}
	if err := read(resp.Body); err != nil {
		return sdk.WrapError(err, "cannot read %s", url)
	}
	return nil
}
