package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type WorkerHookProjectIntegrationModel struct {
	ID                        int64                         `json:"id" db:"id"`
	ProjectIntegrationModelID int64                         `json:"project_integration_id" db:"project_integration_id"`
	Configuration             WorkerHookSetupTeardownConfig `json:"configuration" db:"configuration"`
	Disable                   bool                          `json:"disable" db:"disable"`
}

type WorkerHookSetupTeardownConfig struct {
	EnableOnRegions []string                                  `json:"enable_on_regions" yaml:"enable_on_regions"`
	ByCapabilities  map[string]WorkerHookSetupTeardownScripts `json:"by_capabilitites" yaml:"by_capabilitites"`
}

// Value returns driver.Value from WorkerHookSetupTeardownConfig.
func (w WorkerHookSetupTeardownConfig) Value() (driver.Value, error) {
	j, err := json.Marshal(w)
	return j, WrapError(err, "cannot marshal Action")
}

// Scan action.
func (w *WorkerHookSetupTeardownConfig) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, w), "cannot unmarshal WorkerHookSetupTeardownConfig")
}

type WorkerHookSetupTeardownScripts struct {
	Priority int    `json:"int" yaml:"int"`
	Label    string `json:"label" yaml:"label"`
	Setup    string `json:"setup" yaml:"setup"`
	Teardown string `json:"teardown" yaml:"teardown"`
}
