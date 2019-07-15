package sdk

import (
	"bytes"
	"html/template"
	"time"
)

// Worker represents instances of CDS workers living to serve.
type Worker struct {
	ID         string    `json:"id" cli:"-" db:"id"`
	Name       string    `json:"name" cli:"name,key" db:"name"`
	LastBeat   time.Time `json:"lastbeat" cli:"lastbeat" db:"last_beat"`
	ModelID    int64     `json:"model_id" cli:"-"  db:"model_id"`
	JobRunID   *int64    `json:"job_run_id" cli:"-"  db:"job_run_id"`
	Status     string    `json:"status" cli:"status" db:"status"` // Waiting, Building, Disabled, Unknown
	HatcheryID int64     `json:"hatchery_id" cli:"-" db:"hatchery_id"`
	Uptodate   bool      `json:"uptodate" cli:"-" db:"-"`
	ConsumerID string    `json:"-" cli:"-"  db:"auth_consumer_id"`
	Version    string    `json:"version" cli:"version"  db:"version"`
	OS         string    `json:"os" cli:"os"  db:"os"`
	Arch       string    `json:"arch" cli:"arch"  db:"arch"`
}

// WorkerRegistrationForm represents the arguments needed to register a worker
type WorkerRegistrationForm struct {
	BinaryCapabilities []string
	Version            string
	OS                 string
	Arch               string
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
	Model           string `json:"model"`
	HatcheryName    string `json:"hatchery_name"`
	WorkflowJobID   int64  `json:"workflow_job_id"`
	TTL             int    `json:"ttl"`
	FromWorkerImage bool   `json:"from_worker_image"`
	//Graylog params
	GraylogHost       string `json:"graylog_host"`
	GraylogPort       int    `json:"graylog_port"`
	GraylogExtraKey   string `json:"graylog_extra_key"`
	GraylogExtraValue string `json:"graylog_extra_value"`
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
