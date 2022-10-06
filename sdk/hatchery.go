package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type Hatchery struct {
	ID            string         `json:"id" db:"id"`
	Name          string         `json:"name" db:"name"`
	Config        HatcheryConfig `json:"config" db:"config"`
	LastHeartBeat time.Time      `json:"last_heartbeat" db:"last_heartbeat"`
}

type HatcheryConfig map[string]interface{}

func (hc HatcheryConfig) Value() (driver.Value, error) {
	j, err := json.Marshal(hc)
	return j, WrapError(err, "cannot marshal HatcheryConfig")
}

func (hc *HatcheryConfig) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, hc), "cannot unmarshal HatcheryConfig")
}
