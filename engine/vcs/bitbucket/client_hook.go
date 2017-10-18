package bitbucket

import (
	"encoding/json"
	"fmt"

	"github.com/ovh/cds/sdk"
)

const bitbucketHookKey string = "de.aeffle.stash.plugin.stash-http-get-post-receive-hook%3Ahttp-get-post-receive-hook"

func (b *bitbucketClient) getHooksConfig(repo string) (HooksConfig, error) {
	oldHookConfig := HooksConfig{}

	project, slug, err := getRepo(repo)
	if err != nil {
		return oldHookConfig, err
	}

	getPath := fmt.Sprintf("/projects/%s/repos/%s/settings/hooks/%s/settings", project, slug, bitbucketHookKey)
	if err := b.do("GET", "core", getPath, nil, nil, &oldHookConfig); err != nil {
		return oldHookConfig, err
	}

	return oldHookConfig, nil
}

func (g *bitbucketClient) GetHook(repo, url string) (sdk.VCSHook, error) {
	hcfg, err := g.getHooksConfig(repo)
	if err != nil {
		return sdk.VCSHook{}, err
	}

	for i, h := range hcfg.Details {
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

func (g *bitbucketClient) CreateHook(repo string, hook sdk.VCSHook) error {
	project, slug, err := getRepo(repo)
	if err != nil {
		return err
	}

	hcfg, err := g.getHooksConfig(repo)
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
	if err := g.do("PUT", "core", updatePath, nil, values, &hook); err != nil {
		return err
	}

	//If it's the first hook, let's enable the plugin
	if len(hcfg.Details) == 1 {
		enablePath := fmt.Sprintf("/projects/%s/repos/%s/settings/hooks/%s/enabled", project, slug, bitbucketHookKey)
		if err := g.do("PUT", "core", enablePath, nil, values, &hook); err != nil {
			return err
		}
	}

	return nil
}

func (g *bitbucketClient) UpdateHook(repo, url string, hook sdk.VCSHook) error {
	project, slug, err := getRepo(repo)
	if err != nil {
		return err
	}

	hcfg, err := g.getHooksConfig(repo)
	if err != nil {
		return err
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
		return err
	}

	updatePath := fmt.Sprintf("/projects/%s/repos/%s/settings/hooks/%s/settings", project, slug, bitbucketHookKey)
	if err := g.do("PUT", "core", updatePath, nil, values, &hook); err != nil {
		return err
	}

	return nil
}

func (g *bitbucketClient) DeleteHook(repo string, hook sdk.VCSHook) error {
	project, slug, err := getRepo(repo)
	if err != nil {
		return err
	}

	hcfg, err := g.getHooksConfig(repo)
	if err != nil {
		return err
	}

	for i, h := range hcfg.Details {
		if hook.URL == h.URL {
			hcfg.Details = append(hcfg.Details[:i], hcfg.Details[i+1:]...)
			break
		}
	}

	values, err := json.Marshal(&hcfg)
	if err != nil {
		return err
	}

	updatePath := fmt.Sprintf("/projects/%s/repos/%s/settings/hooks/%s/settings", project, slug, bitbucketHookKey)
	if err := g.do("PUT", "core", updatePath, nil, values, &hook); err != nil {
		return err
	}

	return nil
}
