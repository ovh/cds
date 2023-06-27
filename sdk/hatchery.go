package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type AuthConsumerHatcherySigninRequest struct {
	Token        string        `json:"token"`
	Name         string        `json:"name"`
	HatcheryType string        `json:"type"`
	HTTPURL      string        `json:"http_url"`
	Config       ServiceConfig `json:"config" db:"config" cli:"-" mapstructure:"config"`
	PublicKey    []byte        `json:"public_key"`
	Version      string        `json:"version"`
}

type AuthConsumerHatcherySigninResponse struct {
	Uptodate bool     `json:"up_to_date"`
	APIURL   string   `json:"api_url"`
	Token    string   `json:"token"`
	Hatchery Hatchery `json:"hatchery"`
}

type HatcheryStatus struct {
	ID         int64            `json:"id" db:"id" cli:"id,key"`
	HatcheryID string           `json:"hatchery_id" db:"hatchery_id" cli:"hatchery_id"`
	SessionID  string           `json:"session_id" db:"session_id" cli:"session_id"`
	Status     MonitoringStatus `json:"monitoring_status" db:"monitoring_status"`
}

type Hatchery struct {
	ID            string        `json:"id" db:"id" cli:"id,key"`
	Name          string        `json:"name" db:"name" cli:"name"`
	ModelType     string        `json:"model_type" db:"model_type" cli:"model_type"`
	Config        ServiceConfig `json:"config" db:"config"`
	LastHeartbeat time.Time     `json:"last_heartbeat,omitempty" db:"last_heartbeat" cli:"last_heartbeat"`
	PublicKey     []byte        `json:"public_key" db:"public_key"`
	HTTPURL       string        `json:"http_url" db:"http_url"`

	// On signup / regen
	Token string `json:"token,omitempty" db:"-" cli:"token,omitempty"`
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
