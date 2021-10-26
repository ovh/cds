package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type WorkerHookProjectIntegrationModel struct {
	ID                        int64                         `json:"id" db:"id" cli:"id,key"`
	ProjectIntegrationModelID int64                         `json:"project_integration_id" db:"project_integration_id" cli:"-"`
	Configuration             WorkerHookSetupTeardownConfig `json:"configuration" db:"configuration" cli:"configuration"`
	Disable                   bool                          `json:"disable" db:"disable" cli:"disable"`
}

type WorkerHookSetupTeardownConfig struct {
	DisableOnRegions []string                                  `json:"disable_on_regions" yaml:"disable_on_regions"`
	ByCapabilities   map[string]WorkerHookSetupTeardownScripts `json:"by_capabilities" yaml:"by_capabilities"`
}

// Value returns driver.Value from WorkerHookSetupTeardownConfig.
func (w WorkerHookSetupTeardownConfig) Value() (driver.Value, error) {
	j, err := json.Marshal(w)
	return j, WrapError(err, "cannot marshal WorkerHookSetupTeardownConfig")
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
	Priority int    `json:"priority" yaml:"priority"`
	Label    string `json:"label" yaml:"label"`
	Setup    string `json:"setup" yaml:"setup"`
	Teardown string `json:"teardown" yaml:"teardown"`
}
