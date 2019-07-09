package feature

import (
	"strings"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk/izanami"
	"github.com/ovh/cds/sdk/log"
)

const (
	// FeatEnableTracing is the opencensus tracing feature id
	FeatEnableTracing = "cds:tracing"

	cacheFeatureKey = "feature:"
)

var izanamiClient *izanami.Client

// CheckContext represents the context send to Izanami to check if the feature is enabled
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
	return []string{FeatEnableTracing}
}

// Init initialize Izanami client
func Init(apiURL, clientID, clientSecret string) error {
	izc, err := izanami.New(apiURL, clientID, clientSecret)
	SetClient(izc)
	return err
}

// SetClient set a client driver for Izanami
func SetClient(c *izanami.Client) {
	izanamiClient = c
}

// GetFeatures tree for the given project from cache, if not found in cache init from Izanami.
func GetFeatures(store cache.Store, projectKey string) map[string]bool {
	projFeats := ProjectFeatures{}

	if store.Get(cacheFeatureKey+projectKey, &projFeats) {
		// if missing features, invalidate cache and rebuild data from Izanami
		var missingFeature bool
		for _, f := range List() {
			if _, ok := projFeats.Features[f]; !ok {
				missingFeature = true
				break
			}
		}
		if !missingFeature {
			return projFeats.Features
		}
	}

	// get all features from Izanami and store in cache
	projFeats = ProjectFeatures{Key: projectKey, Features: make(map[string]bool)}
	for _, f := range List() {
		projFeats.Features[f] = getStatusFromIzanami(f, projectKey)
	}

	// no expiration delay is set, the cache is cleared by Izanami calls on /feature/clean
	store.Set(cacheFeatureKey+projectKey, projFeats)

	return projFeats.Features
}

// IsEnabled check if feature is enabled for the given project.
func IsEnabled(store cache.Store, featureID string, projectKey string) bool {
	fs := GetFeatures(store, projectKey)

	if v, ok := fs[featureID]; ok {
		return v
	}

	// if features not in cache, it means that it's not a key from listed in List() func
	// try to get a value from Izanami
	return getStatusFromIzanami(featureID, projectKey)
}

func getStatusFromIzanami(featureID string, projectKey string) bool {
	// no feature flipping always return active.
	if izanamiClient == nil || izanamiClient.Feature() == nil {
		return true
	}

	// get from Izanami
	resp, errCheck := izanamiClient.Feature().CheckWithContext(featureID, CheckContext{projectKey})
	if errCheck != nil {
		if !strings.Contains(errCheck.Error(), "404") {
			log.Warning("Feature.IsEnabled > Cannot check feature %s: %s", featureID, errCheck)
			return false
		}
		resp.Active = true
	}

	return resp.Active
}

// Clean the feature cache
func Clean(store cache.Store) {
	keys := cache.Key(cacheFeatureKey, "*")
	store.DeleteAll(keys)
}
