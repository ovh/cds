package project

import (
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/feature"
	"github.com/ovh/cds/sdk"
)

// LoadFeatures loads features into a project from the feature flipping provider
func LoadFeatures(store cache.Store, p *sdk.Project) error {
	for _, f := range feature.List() {
		// force load cache for the given project
		feature.IsEnabled(store, f, p.Key)
	}
	p.Features = feature.GetFromCache(store, p.Key)
	return nil
}
