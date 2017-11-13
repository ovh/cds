package bitbucket

import (
	"encoding/json"
	"fmt"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const bitbucketHookKey string = "de.aeffle.stash.plugin.stash-http-get-post-receive-hook%3Ahttp-get-post-receive-hook"

func (b *bitbucketClient) getHooksConfig(repo string) (HooksConfig, error) {
	oldHookConfig := HooksConfig{}

	project, slug, err := getRepo(repo)
	if err != nil {
		return oldHookConfig, sdk.WrapError(err, "vcs> bitbucket> getHooksConfig>")
	}

	getPath := fmt.Sprintf("/projects/%s/repos/%s/settings/hooks/%s/settings", project, slug, bitbucketHookKey)
	if err := b.do("GET", "core", getPath, nil, nil, &oldHookConfig); err != nil {
		return oldHookConfig, sdk.WrapError(err, "vcs> bitbucket> getHooksConfig> Unable to get hook config")
	}

	return oldHookConfig, nil
}

func (b *bitbucketClient) GetHook(repo, url string) (sdk.VCSHook, error) {
	hcfg, err := b.getHooksConfig(repo)
	if err != nil {
		return sdk.VCSHook{}, err
	}

	for i, h := range hcfg.Details {
		log.Debug("vcs> bitbucket> GetHook> %+v", h)
		if h.URL == url {
			return sdk.VCSHook{
				ContentType: h.PostContentType,
				Disable:     false,
				Events:      []string{"push"},
				ID:          fmt.Sprintf("%d", i),
				InsecureSSL: false,
				Method:      h.Method,
				Name:        fmt.Sprintf("Location %d", i),
				URL:         h.URL,
				Body:        h.PostData,
			}, nil
		}
	}

	return sdk.VCSHook{}, sdk.ErrHookNotFound
}

func (b *bitbucketClient) CreateHook(repo string, hook sdk.VCSHook) error {
	project, slug, err := getRepo(repo)
	if err != nil {
		return err
	}

	hcfg, err := b.getHooksConfig(repo)
	if err != nil {
		return err
	}

	hcfg.Details = append(hcfg.Details, HookConfigDetail{
		Method:          hook.Method,
		URL:             hook.URL,
		PostContentType: hook.ContentType,
		PostData:        hook.Body,
	})
	hcfg.LocationCount = fmt.Sprintf("%d", len(hcfg.Details))

	values, err := json.Marshal(&hcfg)
	if err != nil {
		return err
	}

	updatePath := fmt.Sprintf("/projects/%s/repos/%s/settings/hooks/%s/settings", project, slug, bitbucketHookKey)
	if err := b.do("PUT", "core", updatePath, nil, values, &hook); err != nil {
		return sdk.WrapError(err, "vcs> bitbucket> CreateHook> Unable to update hook config")
	}

	//If it's the first hook, let's enable the plugin
	if len(hcfg.Details) == 1 {
		enablePath := fmt.Sprintf("/projects/%s/repos/%s/settings/hooks/%s/enabled", project, slug, bitbucketHookKey)
		if err := b.do("PUT", "core", enablePath, nil, values, &hook); err != nil {
			return sdk.WrapError(err, "vcs> bitbucket> CreateHook> Unable to get enable hook")
		}
	}

	return nil
}

func (b *bitbucketClient) UpdateHook(repo, url string, hook sdk.VCSHook) error {
	project, slug, err := getRepo(repo)
	if err != nil {
		return sdk.WrapError(err, "vcs> bitbucket> UpdateHook>")
	}

	hcfg, err := b.getHooksConfig(repo)
	if err != nil {
		return sdk.WrapError(err, "vcs> bitbucket> UpdateHook>")
	}

	for i := range hcfg.Details {
		h := &hcfg.Details[i]
		if h.URL == url {
			h.Method = hook.Method
			h.PostContentType = hook.ContentType
			h.PostData = hook.Body
			break
		}
	}

	values, err := json.Marshal(&hcfg)
	if err != nil {
		return sdk.WrapError(err, "vcs> bitbucket> UpdateHook> Unable to unmarshal hooks config")
	}

	updatePath := fmt.Sprintf("/projects/%s/repos/%s/settings/hooks/%s/settings", project, slug, bitbucketHookKey)
	if err := b.do("PUT", "core", updatePath, nil, values, &hook); err != nil {
		return sdk.WrapError(err, "vcs> bitbucket> CreateHook> Unable to update hook config")
	}

	return nil
}

func (b *bitbucketClient) DeleteHook(repo string, hook sdk.VCSHook) error {
	project, slug, err := getRepo(repo)
	if err != nil {
		return sdk.WrapError(err, "vcs> bitbucket> DeleteHook>")
	}

	hcfg, err := b.getHooksConfig(repo)
	if err != nil {
		return sdk.WrapError(err, "vcs> bitbucket> DeleteHook>")
	}

	for i, h := range hcfg.Details {
		if hook.URL == h.URL {
			hcfg.Details = append(hcfg.Details[:i], hcfg.Details[i+1:]...)
			break
		}
	}

	values, err := json.Marshal(&hcfg)
	if err != nil {
		return sdk.WrapError(err, "vcs> bitbucket> DeleteHook> Unable to unmarshal hooks config")
	}

	updatePath := fmt.Sprintf("/projects/%s/repos/%s/settings/hooks/%s/settings", project, slug, bitbucketHookKey)
	if err := b.do("PUT", "core", updatePath, nil, values, &hook); err != nil {
		return sdk.WrapError(err, "vcs> bitbucket> DeleteHook> Unable to update hook config")
	}

	return nil
}
