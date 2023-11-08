package cdn

type Signature struct {
	Worker          *SignatureWorker
	Service         *SignatureService
	HatcheryService *SignatureHatcheryService // V2 required for serviceLogs
	JobName         string                    // V2 required
	JobID           int64
	RunJobID        string // V2 required
	ProjectKey      string // V2 required
	WorkflowName    string // V2 required
	WorkflowID      int64
	RunID           int64
	WorkflowRunID   string // V2 required
	RunNumber       int64  // V2 required
	RunAttempt      int64  // V2 required
	Region          string // V2 required
	NodeRunName     string
	NodeRunID       int64
	Timestamp       int64
}

type SignatureWorker struct {
	WorkerID      string // V2 required
	WorkerName    string // V2 required
	StepOrder     int64
	StepName      string
	FileName      string
	FilePerm      uint32
	CacheTag      string
	RunResultID   string // V2Runresult required
	RunResultName string // V2Runresult required
	RunResultType string // V2Runresult required
}

type SignatureHatcheryService struct {
	HatcheryID   string
	HatcheryName string
	WorkerName   string
	ServiceName  string
}

type SignatureService struct {
	HatcheryID      int64
	HatcheryName    string
	RequirementID   int64
	RequirementName string
	WorkerName      string
}
