package sdk

import (
	"database/sql/driver"
	json "encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/hashstructure"

	"github.com/ovh/cds/sdk/cdn"
)

type CDNItem struct {
	ID           string       `json:"id" db:"id"`
	Created      time.Time    `json:"created" db:"created"`
	LastModified time.Time    `json:"last_modified" db:"last_modified"`
	Hash         string       `json:"hash" db:"cipher_hash" gorpmapping:"encrypted,ID,APIRefHash,Type"`
	APIRef       CDNLogAPIRef `json:"api_ref" db:"api_ref"`
	APIRefHash   string       `json:"api_ref_hash" db:"api_ref_hash"`
	Status       string       `json:"status" db:"status"`
	Type         CDNItemType  `json:"type" db:"type"`
	Size         int64        `json:"size" db:"size"`
	MD5          string       `json:"md5" db:"md5"`
	ToDelete     bool         `json:"to_delete" db:"to_delete"`
}

type CDNItemUnit struct {
	ID           string      `json:"id" db:"id"`
	Type         CDNItemType `json:"type" db:"type"`
	ItemID       string      `json:"item_id" db:"item_id"`
	UnitID       string      `json:"unit_id" db:"unit_id"`
	LastModified time.Time   `json:"last_modified" db:"last_modified"`
	Locator      string      `json:"locator" db:"cipher_locator" gorpmapping:"encrypted,UnitID,ItemID"`
	HashLocator  string      `json:"hash_locator" db:"hash_locator"`
	Item         *CDNItem    `json:"-" db:"-"`
	ToDelete     bool        `json:"to_delete" db:"to_delete"`
}

type CDNUnit struct {
	ID      string        `json:"id" db:"id"`
	Created time.Time     `json:"created" db:"created"`
	Name    string        `json:"name" db:"name"`
	Config  ServiceConfig `json:"config" db:"config"`
}

type CDNLogsLines struct {
	APIRef     string `json:"api_ref"`
	LinesCount int64  `json:"lines_count"`
}

type CDNLogLinks struct {
	CDNURL   string           `json:"cdn_url,omitempty"`
	ItemType CDNItemType      `json:"item_type"`
	Data     []CDNLogLinkData `json:"datas"`
}

type CDNLogLinkData struct {
	APIRef    string `json:"api_ref"`
	StepOrder int64  `json:"step_order"`
}

type CDNLogLink struct {
	CDNURL   string      `json:"cdn_url,omitempty"`
	ItemType CDNItemType `json:"item_type"`
	APIRef   string      `json:"api_ref"`
}

type CDNMarkDelete struct {
	RunID int64 `json:"run_id,omitempty"`
}

type CDNLogAPIRef struct {
	ProjectKey     string `json:"project_key"`
	WorkflowName   string `json:"workflow_name"`
	WorkflowID     int64  `json:"workflow_id"`
	RunID          int64  `json:"run_id"`
	NodeRunID      int64  `json:"node_run_id"`
	NodeRunName    string `json:"node_run_name"`
	NodeRunJobID   int64  `json:"node_run_job_id"`
	NodeRunJobName string `json:"node_run_job_name"`

	// for workers
	StepOrder    int64   `json:"step_order"`
	StepName     string  `json:"step_name,omitempty"`
	ArtifactName *string `json:"artifact_name,omitempty"`

	// for hatcheries
	RequirementServiceID   int64  `json:"service_id,omitempty"`
	RequirementServiceName string `json:"service_name,omitempty"`
}

func NewCDNApiRef(t CDNItemType, signature cdn.Signature) CDNLogAPIRef {
	// Build cds api ref
	apiRef := CDNLogAPIRef{
		ProjectKey:     signature.ProjectKey,
		WorkflowName:   signature.WorkflowName,
		WorkflowID:     signature.WorkflowID,
		RunID:          signature.RunID,
		NodeRunJobName: signature.JobName,
		NodeRunJobID:   signature.JobID,
	}
	if signature.Worker != nil {
		apiRef.StepName = signature.Worker.StepName
		apiRef.StepOrder = signature.Worker.StepOrder
	}
	if signature.Service != nil {
		apiRef.RequirementServiceID = signature.Service.RequirementID
		apiRef.RequirementServiceName = signature.Service.RequirementName
	}

	switch t {
	case CDNTypeItemStepLog, CDNTypeItemServiceLog:
		apiRef.NodeRunName = signature.NodeRunName
		apiRef.NodeRunID = signature.NodeRunID
	case CDNTypeItemArtifact:
		apiRef.ArtifactName = &signature.Worker.ArtifactName
	}
	return apiRef
}

type CDNItemResume struct {
	CDNItem
	Location map[string]CDNItemUnit `json:"location,omitempty"`
}

func (a CDNLogAPIRef) ToFilename() string {
	jobName := strings.Replace(a.NodeRunJobName, " ", "", -1)

	isService := a.RequirementServiceID > 0 && a.RequirementServiceName != ""
	var suffix string
	if isService {
		suffix = fmt.Sprintf("service.%s", a.RequirementServiceName)
	} else {
		suffix = fmt.Sprintf("step.%d", a.StepOrder)
	}

	return fmt.Sprintf("project.%s-workflow.%s-pipeline.%s-job.%s-%s.log",
		a.ProjectKey,
		a.WorkflowName,
		a.NodeRunName,
		jobName,
		suffix,
	)
}

func (a CDNLogAPIRef) ToHash() (string, error) {
	hashRefU, err := hashstructure.Hash(a, nil)
	if err != nil {
		return "", WithStack(err)
	}
	return strconv.FormatUint(hashRefU, 10), nil
}

// Value returns driver.Value from CDNLogAPIRef.
func (a CDNLogAPIRef) Value() (driver.Value, error) {
	j, err := json.Marshal(a)
	return j, WrapError(err, "cannot marshal CDNLogAPIRef")
}

// Scan CDNLogAPIRef.
func (a *CDNLogAPIRef) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(json.Unmarshal(source, a), "cannot unmarshal CDNLogAPIRef")
}

type CDNItemType string

func (t CDNItemType) Validate() error {
	switch t {
	case CDNTypeItemStepLog, CDNTypeItemServiceLog, CDNTypeItemArtifact:
		return nil
	}
	return NewErrorFrom(ErrWrongRequest, "invalid item type")
}

func (t CDNItemType) IsLog() bool {
	switch t {
	case CDNTypeItemStepLog, CDNTypeItemServiceLog:
		return true
	}
	return false
}

const (
	CDNTypeItemStepLog     CDNItemType = "step-log"
	CDNTypeItemServiceLog  CDNItemType = "service-log"
	CDNTypeItemArtifact    CDNItemType = "artifact"
	CDNStatusItemIncoming              = "Incoming"
	CDNStatusItemCompleted             = "Completed"
)

type CDNReaderFormat string

const (
	CDNReaderFormatJSON CDNReaderFormat = "json"
	CDNReaderFormatText CDNReaderFormat = "text"
)

type CDNWSEvent struct {
	ItemType CDNItemType `json:"item_type"`
	APIRef   string      `json:"api_ref"`
}

type CDNStreamFilter struct {
	ItemType CDNItemType `json:"item_type"`
	APIRef   string      `json:"api_ref"`
	Offset   int64       `json:"offset"`
}

func (f CDNStreamFilter) Validate() error {
	if !f.ItemType.IsLog() {
		return NewErrorFrom(ErrWrongRequest, "invalid item log type")
	}
	if f.APIRef == "" {
		return NewErrorFrom(ErrWrongRequest, "invalid given api ref")
	}
	return nil
}
