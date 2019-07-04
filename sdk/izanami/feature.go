package izanami

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// FeaturesResponse represents the http response for listAll
type FeaturesResponse struct {
	Results  []FeatureModel `json:"results"`
	Metadata Metadata       `json:"metadata"`
}

// FeatureCheckResponse represents the hhtp response for a feature check
type FeatureCheckResponse struct {
	Active bool `json:"active"`
}

// Feature represents a feature in izanami point of view
type FeatureModel struct {
	ID         string             `json:"id"`
	Enabled    bool               `json:"enabled"`
	Parameters map[string]string  `json:"parameters"`
	Strategy   ActivationStrategy `json:"activationStrategy"`
}

// ActivationStrategy represents the different way to activate a feature
type ActivationStrategy string

const (
	NoStrategy   ActivationStrategy = "NO_STRATEGY"
	ReleaseDate  ActivationStrategy = "RELEASE_DATE"
	Script       ActivationStrategy = "SCRIPT"
	GlobalScript ActivationStrategy = "GLOBAL_SCRIPT"
)

// List features on the given page.
func (c *FeatureClient) List(page int, pageSize int) (FeaturesResponse, error) {
	var features FeaturesResponse

	httpParams := make(map[string]string)
	httpParams[httpParamPage] = strconv.Itoa(page)
	httpParams[httpParamPageSize] = strconv.Itoa(pageSize)

	res, errListAll := c.client.get("/features", httpParams)
	if errListAll != nil {
		return features, errListAll
	}

	if err := json.Unmarshal(res, &features); err != nil {
		return features, err
	}
	return features, nil
}

// ListAll browses all pages and returns all features
func (c *FeatureClient) ListAll() ([]FeatureModel, error) {
	features := []FeatureModel{}

	currentPage := 1
	pageSize := 20

	for {
		res, err := c.List(currentPage, pageSize)
		if err != nil {
			return features, err
		}
		features = append(features, res.Results...)
		if res.Metadata.Page >= res.Metadata.NbPages {
			break
		}
		currentPage++
	}
	return features, nil
}

// Create a new feature
func (c *FeatureClient) Create(feat FeatureModel) error {
	_, errPost := c.client.post("/features", feat)
	if errPost != nil {
		return errPost
	}
	return nil
}

// Get a feature by its id
func (c *FeatureClient) Get(id string) (FeatureModel, error) {
	var feature FeatureModel
	body, errGet := c.client.get(fmt.Sprintf("/features/%s", id), nil)
	if errGet != nil {
		return feature, errGet
	}
	if err := json.Unmarshal(body, &feature); err != nil {
		return feature, err
	}
	return feature, nil
}

// Update the given feature
func (c *FeatureClient) Update(feat FeatureModel) error {
	_, errPut := c.client.put(fmt.Sprintf("/features/%s", feat.ID), feat)
	if errPut != nil {
		return errPut
	}
	return nil
}

// Delete a feature by its id
func (c *FeatureClient) Delete(id string) error {
	return c.client.delete(fmt.Sprintf("/features/%s", id))
}

// CheckWithoutContext if a feature is enable
func (c *FeatureClient) CheckWithoutContext(id string) (FeatureCheckResponse, error) {
	var checkResp FeatureCheckResponse
	body, errB := c.client.get(fmt.Sprintf("/features/%s/check", id), nil)
	if errB != nil {
		return checkResp, errB
	}
	err := json.Unmarshal(body, &checkResp)
	return checkResp, err
}

// CheckWithContext if a feature is enable for the given context
func (c *FeatureClient) CheckWithContext(id string, context interface{}) (FeatureCheckResponse, error) {
	var checkResp FeatureCheckResponse
	body, errB := c.client.post(fmt.Sprintf("/features/%s/check", id), context)
	if errB != nil {
		return checkResp, errB
	}
	err := json.Unmarshal(body, &checkResp)
	return checkResp, err
}
