package feature

import (
	"github.com/ovhlabs/izanami-go-client"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const (
	FeatWorkflowAsCode = "cds:wasc"

	cacheFeatureKey = "feature:"
)

var c *client.Client

// CheckContext represents the context send to izanami to check if the feature is enabled
type CheckContext struct {
	Key string `json:"key"`
}

// List all features
func List() []string {
	return []string{FeatWorkflowAsCode}
}

// Init initialize izanami client
func Init(apiURL, clientID, clientSecret string) error {
	var errC error
	c, errC = client.New(apiURL, clientID, clientSecret)
	return errC
}

// GetFromCache get feature tree for the given project from cache
func GetFromCache(store cache.Store, projectKey string) sdk.ProjectFeatures {
	projFeats := sdk.ProjectFeatures{}
	store.Get(cacheFeatureKey+projectKey, &projFeats)
	return projFeats
}

// IsEnabled check if feature is enabled for the given project
func IsEnabled(cache cache.Store, featureID string, projectKey string) bool {
	// No feature flipping
	if c == nil {
		return true
	}

	var projFeats sdk.ProjectFeatures

	// Get from cache
	if !cache.Get(cacheFeatureKey+projectKey, &projFeats) {
		if v, ok := projFeats.Features[featureID]; ok {
			return v
		}
	}

	// Get from izanami
	resp, errCheck := c.Feature().CheckWithContext(featureID, CheckContext{projectKey})
	if errCheck != nil {
		log.Warning("Feature.IsEnabled > Cannot check feature %s: %s", featureID, errCheck)
		return false
	}
	projFeats.Key = projectKey
	if projFeats.Features == nil {
		projFeats.Features = make(map[string]bool)
	}
	projFeats.Features[featureID] = resp.Active

	// Push in cache
	cache.Set(projectKey, projFeats)

	return resp.Active
}

// Clean the feature cache
func Clean(store cache.Store) {
	keys := cache.Key(cacheFeatureKey, "*")
	store.DeleteAll(keys)
}
