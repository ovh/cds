package stash

import (
	"encoding/json"
	"fmt"
)

type HookDetail struct {
	Key           string `json:"key"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Description   string `json:"description"`
	Version       string `json:"version"`
	ConfigFormKey string `json:"configFormKey"`
}

type Hook struct {
	Enabled bool        `json:"enabled"`
	Details *HookDetail `json:"details"`
}

type HookResource struct {
	client *Client
}

// Enable hook for named repository
func (r *HookResource) CreateHook(project, slug, hook_key, method, link, branchFilter, tagFilter, userFilter string) (*Hook, error) {
	hookConfig := map[string]string{
		"httpMethod":   method,
		"url":          link,
		"branchFilter": branchFilter,
		"tagFilter":    tagFilter,
		"userFilter":   userFilter,
	}

	values, err := json.Marshal(hookConfig)
	if err != nil {
		return nil, err
	}

	hook := Hook{}

	// Set hook
	updatePath := fmt.Sprintf("/projects/%s/repos/%s/settings/hooks/%s/settings",
		project, slug, hook_key)
	if err := r.client.do("PUT", "core", updatePath, nil, values, &hook); err != nil {

		return nil, err
	}

	// Enable hook
	enablePath := GetEnablePath(project, slug, hook_key)
	if err := r.client.do("PUT", "core", enablePath, nil, values, &hook); err != nil {
		return nil, err
	}

	return &hook, nil
}

// Disable hook for named repository
func (r *HookResource) DeleteHook(project, slug, hook_key, link string) error {
	hookConfig := map[string]string{"url": link}
	values, err := json.Marshal(hookConfig)
	if err != nil {
		return err
	}

	hook := Hook{}

	enablePath := GetEnablePath(project, slug, hook_key)
	if err := r.client.do("DELETE", "core", enablePath, nil, values, &hook); err != nil {
		return err
	}

	// TODO check that enabled is False?
	return nil
}

func GetEnablePath(project, slug, hook_key string) string {
	return fmt.Sprintf("/projects/%s/repos/%s/settings/hooks/%s/enabled",
		project, slug, hook_key)
}
