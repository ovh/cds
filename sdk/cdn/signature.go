package cdn

type Signature struct {
	Worker        *SignatureWorker
	Service       *SignatureService
	JobName       string
	JobID         int64
	RunJobID      string
	ProjectKey    string
	WorkflowName  string
	WorkflowID    int64
	RunID         int64
	WorkflowRunID string
	RunNumber     int64
	RunAttempt    int64
	NodeRunName   string
	NodeRunID     int64
	Timestamp     int64
}

type SignatureWorker struct {
	WorkerID      string
	WorkerName    string
	StepOrder     int64
	StepName      string
	FileName      string
	FilePerm      uint32
	CacheTag      string
	RunResultType string
}

type SignatureService struct {
	HatcheryID      int64
	HatcheryName    string
	RequirementID   int64
	RequirementName string
	WorkerName      string
}
