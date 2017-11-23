package sdk

import "time"

// MonDBTimes is used by /mon/db/times
type MonDBTimes struct {
	Now                    time.Time `json:"time" cli:"time"`
	Version                string    `json:"version" cli:"version"`
	Hostname               string    `json:"hostname" cli:"hostname"`
	ProjectLoadAll         string    `json:"projectLoadAll" cli:"projectLoadAll"`
	ProjectLoadAllWithApps string    `json:"projectLoadAllWithApps" cli:"projectLoadAllWithApps"`
	ProjectLoadAllRaw      string    `json:"projectLoadAllRaw" cli:"projectLoadAllRaw"`
	ProjectCount           string    `json:"projectCount" cli:"projectCount"`
	QueueWorkflow          string    `json:"queueWorklow" cli:"queueWorklow"`
}
