package feature

import (
	"strings"

	"github.com/ovhlabs/izanami-go-client"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk/log"
)

const (
	// FeatWorkflowAsCode is workflow as code feature id
	FeatWorkflowAsCode = "cds:wasc"

	cacheFeatureKey = "feature:"
)

var izanami *client.Client

// CheckContext represents the context send to izanami to check if the feature is enabled
type CheckContext struct {
	Key string `json:"key"`
}

// ProjectFeatures represents a project and the feature states
type ProjectFeatures struct {
	Key      string          `json:"key"`
	Features map[string]bool `json:"features"`
}

// List all features
func List() []string {
	return []string{FeatWorkflowAsCode}
}

// Init initialize izanami client
func Init(apiURL, clientID, clientSecret string) error {
	var errC error
	izanami, errC = client.New(apiURL, clientID, clientSecret)
	return errC
}

// GetFromCache get feature tree for the given project from cache
func GetFromCache(store cache.Store, projectKey string) map[string]bool {
	projFeats := ProjectFeatures{}
	store.Get(cacheFeatureKey+projectKey, &projFeats)
	return projFeats.Features
}

// IsEnabled check if feature is enabled for the given project
func IsEnabled(cache cache.Store, featureID string, projectKey string) bool {
	projFeats := ProjectFeatures{Key: projectKey, Features: make(map[string]bool)}
	// No feature flipping
	if izanami == nil {
		projFeats.Features[featureID] = true
		cache.Set(cacheFeatureKey+projectKey, projFeats)
		return true
	}

	// Get from cache
	if !cache.Get(cacheFeatureKey+projectKey, &projFeats) {
		if v, ok := projFeats.Features[featureID]; ok {
			return v
		}
	}

	// Get from izanami
	resp, errCheck := izanami.Feature().CheckWithContext(featureID, CheckContext{projectKey})
	if errCheck != nil {
		if !strings.Contains(errCheck.Error(), "404") {
			log.Warning("Feature.IsEnabled > Cannot check feature %s: %s", featureID, errCheck)
			return false
		}
		resp.Active = true
	}
	projFeats.Key = projectKey
	if projFeats.Features == nil {
		projFeats.Features = make(map[string]bool)
	}
	projFeats.Features[featureID] = resp.Active

	// Push in cache
	cache.Set(cacheFeatureKey+projectKey, projFeats)

	return resp.Active
}

// Clean the feature cache
func Clean(store cache.Store) {
	keys := cache.Key(cacheFeatureKey, "*")
	store.DeleteAll(keys)
}
