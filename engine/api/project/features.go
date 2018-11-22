package project

import (
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/feature"
	"github.com/ovh/cds/sdk"
)

// LoadFeatures loads features into a project from the feature flipping provider.
func LoadFeatures(store cache.Store, p *sdk.Project) {
	p.Features = feature.GetFeatures(store, p.Key)
}
