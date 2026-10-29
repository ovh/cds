package microvm

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

// pluginBinaryCache holds the sha256 of the plugin binaries served by the
// API, by plugin, os/arch and published sha512. A plugin is downloaded by the
// hatchery once per published binary, and by the platform once per node.
type pluginBinaryCache struct {
	mu      sync.Mutex
	sha256s map[string]string
}

// pluginDownloadURL is the API route serving a plugin binary, without
// authentication
func pluginDownloadURL(api, name, osArch string) string {
	return api + "/download/plugin/" + name + "/binary/" + osArch
}

// pluginFiles returns the guest files of the preloaded plugins of the model
// os/arch, in the worker basedir: the worker finds them there under the
// published name and sha512, and does not download them. A plugin whose
// binary is unavailable is left out: the worker downloads it.
func (h *HatcheryMicroVM) pluginFiles(ctx context.Context, osArch, basedir, api string, insecure bool) []interface{} {
	goos, arch, ok := strings.Cut(osArch, "/")
	if !ok {
		return nil
	}
	var files []interface{}
	for _, name := range h.Config.PreloadedPlugins {
		p, err := h.CDSClientV2().PluginsGet(name)
		if err != nil {
			log.Warn(ctx, "hatchery> microvm> cannot get plugin %s, the worker downloads it: %v", name, err)
			continue
		}
		b := p.GetBinary(goos, arch)
		if b == nil {
			continue
		}
		sum, err := h.pluginSHA256(ctx, api, insecure, name, osArch, b.SHA512sum)
		if err != nil {
			log.Warn(ctx, "hatchery> microvm> cannot compute the sha256 of plugin %s %s, the worker downloads it: %v", name, osArch, err)
			continue
		}
		files = append(files, map[string]interface{}{
			"path":   path.Join(basedir, b.Name),
			"mode":   fmt.Sprintf("%04o", b.Perm&0o7777),
			"source": map[string]interface{}{"url": pluginDownloadURL(api, name, osArch), "sha256": sum},
		})
	}
	return files
}

// pluginSHA256 returns the sha256 of the plugin binary published with
// sha512Sum. The binary is only downloaded when that sha512 was not seen
// yet, and its sha256 is kept only when the content matches it: a binary
// replaced since its descriptor was read is not preloaded.
func (h *HatcheryMicroVM) pluginSHA256(ctx context.Context, api string, insecure bool, name, osArch, sha512Sum string) (string, error) {
	key := name + "/" + osArch + "/" + strings.ToLower(sha512Sum)

	c := &h.pluginBinaries
	// Held during the download: concurrent spawns wait for the hash instead of
	// downloading the binary again
	c.mu.Lock()
	defer c.mu.Unlock()
	if sum, ok := c.sha256s[key]; ok {
		return sum, nil
	}

	httpClient := cdsclient.NewHTTPClient(2*time.Minute, insecure)
	hash256, hash512 := sha256.New(), sha512.New()
	if err := httpGet(ctx, httpClient, pluginDownloadURL(api, name, osArch), func(body io.Reader) error {
		_, err := io.Copy(io.MultiWriter(hash256, hash512), body)
		return err
	}); err != nil {
		return "", err
	}
	if got := hex.EncodeToString(hash512.Sum(nil)); !strings.EqualFold(got, sha512Sum) {
		return "", sdk.WithStack(fmt.Errorf("sha512 mismatch: published %s, downloaded %s", sha512Sum, got))
	}
	sum := hex.EncodeToString(hash256.Sum(nil))
	if c.sha256s == nil {
		c.sha256s = map[string]string{}
	}
	c.sha256s[key] = sum
	return sum, nil
}
