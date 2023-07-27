package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/hashstructure"

	"github.com/ovh/cds/sdk/cdn"
)

const (
	CDSSessionID = "X-CDS-Session-ID"
)

type CDNItem struct {
	ID           string          `json:"id" db:"id"`
	Created      time.Time       `json:"created" db:"created"`
	LastModified time.Time       `json:"last_modified" db:"last_modified"`
	Hash         string          `json:"hash" db:"cipher_hash" gorpmapping:"encrypted,ID,APIRefHash,Type"`
	APIRefRaw    json.RawMessage `json:"api_ref" db:"-"`
	APIRef       CDNApiRef       `json:"-" db:"-"`
	APIRefHash   string          `json:"api_ref_hash" db:"api_ref_hash"`
	Status       string          `json:"status" db:"status"`
	Type         CDNItemType     `json:"type" db:"type"`
	Size         int64           `json:"size" db:"size"`
	MD5          string          `json:"md5" db:"md5"`
	ToDelete     bool            `json:"to_delete" db:"to_delete"`
}

type CDNItemLinks struct {
	CDNHttpURL string    `json:"cdn_http_url"`
	Items      []CDNItem `json:"items"`
}

func (c CDNItem) MarshalJSON() ([]byte, error) {
	type Alias CDNItem // prevent recursion
	itemalias := Alias(c)
	apiRefBts, err := json.Marshal(itemalias.APIRef)
	if err != nil {
		return nil, WithStack(err)
	}
	itemalias.APIRefRaw = apiRefBts

	bts, err := json.Marshal(itemalias)
	return bts, WithStack(err)
}

func (c *CDNItem) UnmarshalJSON(data []byte) error {
	type Alias CDNItem // prevent recursion
	var itemAlias Alias
	if err := JSONUnmarshal(data, &itemAlias); err != nil {
		return WithStack(err)
	}

	switch itemAlias.Type {
	case CDNTypeItemStepLog, CDNTypeItemServiceLog:
		var apiRef CDNLogAPIRef
		if err := JSONUnmarshal(itemAlias.APIRefRaw, &apiRef); err != nil {
			return WithStack(err)
		}
		itemAlias.APIRef = &apiRef
	case CDNTypeItemJobStepLog:
		var apiRef CDNLogAPIRefV2
		if err := JSONUnmarshal(itemAlias.APIRefRaw, &apiRef); err != nil {
			return WithStack(err)
		}
		itemAlias.APIRef = &apiRef
	case CDNTypeItemRunResult:
		var apiRef CDNRunResultAPIRef
		if err := JSONUnmarshal(itemAlias.APIRefRaw, &apiRef); err != nil {
			return WithStack(err)
		}
		itemAlias.APIRef = &apiRef
	case CDNTypeItemWorkerCache:
		var apiRef CDNWorkerCacheAPIRef
		if err := JSONUnmarshal(itemAlias.APIRefRaw, &apiRef); err != nil {
			return WithStack(err)
		}
		itemAlias.APIRef = &apiRef
	}
	*c = CDNItem(itemAlias)
	return nil
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
	ItemType CDNItemType `json:"item_type"`
	APIRef   string      `json:"api_ref"`
}

type CDNMarkDelete struct {
	RunID int64 `json:"run_id,omitempty"`
}

type CDNApiRef interface {
	ToHash() (string, error)
	ToFilename() string
}

type CDNLogAPIRefV2 struct {
	ProjectKey   string `json:"project_key"`
	WorkflowName string `json:"workflow_name"`
	RunID        string `json:"run_id"`
	RunJobID     string `json:"run_job_id"`
	RunJobName   string `json:"run_job_name"`
	RunNumber    int64  `json:"run_number"`
	RunAttempt   int64  `json:"run_attempt"`

	// for workers
	StepOrder int64  `json:"step_order"`
	StepName  string `json:"step_name,omitempty"`

	// for hatcheries
	RequirementServiceName string `json:"service_name,omitempty"`
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
	StepOrder int64  `json:"step_order"`
	StepName  string `json:"step_name,omitempty"`

	// for hatcheries
	RequirementServiceID   int64  `json:"service_id,omitempty"`
	RequirementServiceName string `json:"service_name,omitempty"`
}

type CDNRunResultAPIRef struct {
	ProjectKey    string                `json:"project_key"`
	WorkflowName  string                `json:"workflow_name"`
	WorkflowID    int64                 `json:"workflow_id"`
	RunID         int64                 `json:"run_id"`
	RunJobID      int64                 `json:"run_job_id"`
	RunJobName    string                `json:"run_job_name"`
	RunNodeID     int64                 `json:"run_node_id"`
	ArtifactName  string                `json:"artifact_name"`
	Perm          uint32                `json:"perm"`
	RunResultType WorkflowRunResultType `json:"type"`
}

type CDNWorkerCacheAPIRef struct {
	ProjectKey string    `json:"project_key"`
	CacheTag   string    `json:"cache_tag"`
	ExpireAt   time.Time `json:"expire_at"`
}

func NewCDNApiRef(t CDNItemType, signature cdn.Signature) (CDNApiRef, error) {
	switch t {
	case CDNTypeItemStepLog, CDNTypeItemServiceLog:
		return NewCDNLogApiRef(signature), nil
	case CDNTypeItemRunResult:
		return NewCDNRunResultApiRef(signature), nil
	case CDNTypeItemWorkerCache:
		return NewCDNWorkerCacheApiRef(signature), nil
	case CDNTypeItemJobStepLog:
		return NewCDNLogApiRefV2(signature), nil
	}
	return nil, WrapError(ErrInvalidData, "item type unknown")
}

func NewCDNWorkerCacheApiRef(signature cdn.Signature) CDNApiRef {
	apiRef := CDNWorkerCacheAPIRef{
		ProjectKey: signature.ProjectKey,
		ExpireAt:   time.Now().AddDate(0, 6, 0),
		CacheTag:   signature.Worker.CacheTag,
	}
	return &apiRef
}

func NewCDNRunResultApiRef(signature cdn.Signature) CDNApiRef {
	// Build cds api ref
	apiRef := CDNRunResultAPIRef{
		ProjectKey:    signature.ProjectKey,
		WorkflowName:  signature.WorkflowName,
		WorkflowID:    signature.WorkflowID,
		RunID:         signature.RunID,
		RunNodeID:     signature.NodeRunID,
		RunJobName:    signature.JobName,
		RunJobID:      signature.JobID,
		ArtifactName:  signature.Worker.FileName,
		Perm:          signature.Worker.FilePerm,
		RunResultType: WorkflowRunResultType(signature.Worker.RunResultType),
	}
	return &apiRef
}

func NewCDNLogApiRefV2(signature cdn.Signature) CDNApiRef {
	// Build cds api ref
	apiRef := CDNLogAPIRefV2{
		ProjectKey:   signature.ProjectKey,
		WorkflowName: signature.WorkflowName,
		RunID:        signature.WorkflowRunID,
		RunJobName:   signature.JobName,
		RunJobID:     signature.RunJobID,
		RunNumber:    signature.RunNumber,
		RunAttempt:   signature.RunAttempt,
	}
	if signature.Worker != nil {
		apiRef.StepName = signature.Worker.StepName
		apiRef.StepOrder = signature.Worker.StepOrder
	}
	if signature.Service != nil {
		apiRef.RequirementServiceName = signature.Service.RequirementName
	}
	return &apiRef
}

func NewCDNLogApiRef(signature cdn.Signature) CDNApiRef {
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
	apiRef.NodeRunName = signature.NodeRunName
	apiRef.NodeRunID = signature.NodeRunID
	return &apiRef
}

type CDNItemResume struct {
	CDNItem  CDNItem                `json:"item"` // Here we can't use nested struct because of the custom CDNItem marshaller
	Location map[string]CDNItemUnit `json:"item_units"`
}

func (a *CDNLogAPIRefV2) ToHash() (string, error) {
	hashRefU, err := hashstructure.Hash(a, nil)
	if err != nil {
		return "", WithStack(err)
	}
	return strconv.FormatUint(hashRefU, 10), nil
}

func (a *CDNLogAPIRefV2) ToFilename() string {
	jobName := strings.Replace(a.RunJobName, " ", "", -1)

	isService := a.RequirementServiceName != ""
	var suffix string
	if isService {
		suffix = fmt.Sprintf("service.%s", a.RequirementServiceName)
	} else {
		suffix = fmt.Sprintf("step.%d", a.StepOrder)
	}

	return fmt.Sprintf("project.%s-workflow.%s-job.%s-%s.log",
		a.ProjectKey,
		a.WorkflowName,
		jobName,
		suffix,
	)
}

func (a *CDNLogAPIRef) ToFilename() string {
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

func (c CDNItem) GetCDNLogApiRefV2() (*CDNLogAPIRefV2, bool) {
  apiRef, has := c.APIRef.(*CDNLogAPIRefV2)
  return apiRef, has
}

func (c CDNItem) GetCDNLogApiRef() (*CDNLogAPIRef, bool) {
	apiRef, has := c.APIRef.(*CDNLogAPIRef)
	return apiRef, has
}

func (c CDNItem) GetCDNRunResultApiRef() (*CDNRunResultAPIRef, bool) {
	apiRef, has := c.APIRef.(*CDNRunResultAPIRef)
	return apiRef, has
}

func (c CDNItem) GetCDNWorkerCacheApiRef() (*CDNWorkerCacheAPIRef, bool) {
	apiRef, has := c.APIRef.(*CDNWorkerCacheAPIRef)
	return apiRef, has
}

func (a *CDNWorkerCacheAPIRef) ToHash() (string, error) {
	m := make(map[string]string, 7)
	m["project_key"] = a.ProjectKey
	m["cache_tag"] = a.CacheTag
	m["expireAt"] = a.ExpireAt.String()

	hashRefU, err := hashstructure.Hash(m, nil)
	if err != nil {
		return "", WithStack(err)
	}
	return strconv.FormatUint(hashRefU, 10), nil
}

func (a *CDNWorkerCacheAPIRef) ToFilename() string {
	return a.CacheTag
}

func (a *CDNLogAPIRef) ToHash() (string, error) {
	hashRefU, err := hashstructure.Hash(a, nil)
	if err != nil {
		return "", WithStack(err)
	}
	return strconv.FormatUint(hashRefU, 10), nil
}

func (a *CDNRunResultAPIRef) ToFilename() string {
	return a.ArtifactName
}

func (a *CDNRunResultAPIRef) ToHash() (string, error) {
	m := make(map[string]string, 7)
	m["project_key"] = a.ProjectKey
	m["workflow_name"] = a.WorkflowName
	m["workflow_id"] = strconv.Itoa(int(a.WorkflowID))
	m["run_id"] = strconv.Itoa(int(a.RunID))
	m["run_job_id"] = strconv.Itoa(int(a.RunJobID))
	m["run_job_name"] = a.RunJobName
	m["artifact_name"] = a.ArtifactName

	hashRefU, err := hashstructure.Hash(m, nil)
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
	return WrapError(JSONUnmarshal(source, a), "cannot unmarshal CDNLogAPIRef")
}

type CDNItemType string

func (t CDNItemType) Validate() error {
	switch t {
	case CDNTypeItemStepLog, CDNTypeItemServiceLog, CDNTypeItemRunResult, CDNTypeItemWorkerCache, CDNTypeItemJobStepLog:
		return nil
	}
	return NewErrorFrom(ErrWrongRequest, "invalid item type")
}

func (t CDNItemType) IsLog() bool {
	switch t {
	case CDNTypeItemStepLog, CDNTypeItemServiceLog, CDNTypeItemJobStepLog:
		return true
	}
	return false
}

const (
	CDNTypeItemStepLog     CDNItemType = "step-log"
	CDNTypeItemJobStepLog  CDNItemType = "job-step-log"
	CDNTypeItemServiceLog  CDNItemType = "service-log"
	CDNTypeItemRunResult   CDNItemType = "run-result"
	CDNTypeItemWorkerCache CDNItemType = "worker-cache"
	CDNStatusItemIncoming              = "Incoming"
	CDNStatusItemCompleted             = "Completed"
)

type CDNReaderFormat string

const (
	CDNReaderFormatJSON CDNReaderFormat = "json"
	CDNReaderFormatText CDNReaderFormat = "text"
)

type CDNWSEvent struct {
	ItemType   CDNItemType `json:"item_type"`
	JobRunID   int64       `json:"job_run_id"`
	ItemUnitID string      `json:"new_item_unit_id"`
}

type CDNStreamFilter struct {
	JobRunID int64 `json:"job_run_id"`
}

func (f CDNStreamFilter) Validate() error {
	if f.JobRunID == 0 {
		return NewErrorFrom(ErrWrongRequest, "invalid given job run identifier")
	}
	return nil
}

type CDNUnitHandlerRequest struct {
	ID      string `json:"id" cli:"id"`
	Name    string `json:"name" cli:"name"`
	NbItems int64  `json:"nb_items" cli:"nb_items"`
}
