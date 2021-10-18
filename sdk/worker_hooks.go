package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type WorkerHook struct {
	ID            int64                         `json:"id" db:"id"`
	Name          string                        `json:"name" db:"name"`
	Configuration WorkerHookSetupTeardownConfig `json:"configuration" db:"configuration"`
}

type WorkerHookSetupTeardownConfig struct {
	IntegrationModelName string                                    `json:"integration_model_name" yaml:"by_capabilitites"`
	ByCapabilities       map[string]WorkerHookSetupTeardownScripts `json:"by_capabilitites" yaml:"by_capabilitites"`
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
	Setup    string `json:"setup" yaml:"setup"`
	Teardown string `json:"teardown" yaml:"teardown"`
}
