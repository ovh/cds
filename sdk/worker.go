package sdk

import (
	"bytes"
	"html/template"
	"time"
)

// Worker represents instances of CDS workers living to serve.
type Worker struct {
	ID            string    `json:"id" cli:"-"`
	Name          string    `json:"name" cli:"name,key"`
	LastBeat      time.Time `json:"lastbeat" cli:"lastbeat"`
	GroupID       int64     `json:"group_id" cli:"-"`
	ModelID       int64     `json:"model_id" cli:"-"`
	ActionBuildID int64     `json:"action_build_id" cli:"-"`
	Model         *Model    `json:"model" cli:"-"`
	HatcheryName  string    `json:"hatchery_name" cli:"-"`
	JobType       string    `json:"job_type" cli:"-"`    // sdk.JobType...
	Status        Status    `json:"status" cli:"status"` // Waiting, Building, Disabled, Unknown
	Uptodate      bool      `json:"up_to_date" cli:"-"`
}

// WorkerRegistrationForm represents the arguments needed to register a worker
type WorkerRegistrationForm struct {
	RegistrationOnly   bool
	Name               string
	Token              string
	ModelID            int64
	HatcheryName       string
	BinaryCapabilities []string
	Version            string
	OS                 string
	Arch               string
}

// WorkerTakeForm contains booked JobID if exists
type WorkerTakeForm struct {
	BookedJobID int64
	Time        time.Time
	OS          string
	Arch        string
	Version     string
}

// SpawnErrorForm represents the arguments needed to add error registration on worker model
type SpawnErrorForm struct {
	Error string
	Logs  []byte
}

// WorkerArgs is all the args needed to run a worker
type WorkerArgs struct {
	API             string `json:"api"`
	Token           string `json:"token"`
	Name            string `json:"name"`
	BaseDir         string `json:"base_dir"`
	HTTPInsecure    bool   `json:"http_insecure"`
	Model           int64  `json:"model"`
	HatcheryName    string `json:"hatchery_name"`
	WorkflowJobID   int64  `json:"workflow_job_id"`
	TTL             int    `json:"ttl"`
	FromWorkerImage bool   `json:"from_worker_image"`
	//Graylog params
	GraylogHost       string `json:"graylog_host"`
	GraylogPort       int    `json:"graylog_port"`
	GraylogExtraKey   string `json:"graylog_extra_key"`
	GraylogExtraValue string `json:"graylog_extra_value"`
	//GRPC Params
	GrpcAPI      string `json:"grpc_api"`
	GrpcInsecure bool   `json:"grpc_insecure"`
}

// TemplateEnvs return envs interpolated with worker arguments
func TemplateEnvs(args WorkerArgs, envs map[string]string) (map[string]string, error) {
	for name, value := range envs {
		tmpl, errt := template.New("env").Parse(value)
		if errt != nil {
			return envs, errt
		}
		var buffer bytes.Buffer
		if errTmpl := tmpl.Execute(&buffer, args); errTmpl != nil {
			return envs, errTmpl
		}
		envs[name] = buffer.String()
	}

	return envs, nil
}

// WorkflowNodeJobRunData is returned to worker in answer to postTakeWorkflowJobHandler
type WorkflowNodeJobRunData struct {
	NodeJobRun WorkflowNodeJobRun
	Secrets    []Variable
	Number     int64
	SubNumber  int64
}
