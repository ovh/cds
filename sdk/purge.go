package sdk

type PurgeDryRunRequest struct {
	RetentionPolicy string `json:"retention_policy"`
}

type PurgeDryRunResponse struct {
	NbRunsToAnalize int64 `json:"nb_runs_to_analyze"`
}

type WorkflowRunToKeep struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
	Num    int64  `json:"num"`
}

type UpdateMaxRunRequest struct {
	MaxRuns int64 `json:"max_runs"`
}

type PurgeReport struct {
	ID        string                `json:"id"`
	Workflows []WorkflowPurgeReport `json:"workflows,omitempty"`
}

type WorkflowPurgeReport struct {
	WorkflowName string                   `json:"workflow_name"`
	Refs         []WorkflowRefPurgeReport `json:"ref_report"`
	Error        string                   `json:"error,omitempty"`
}

type WorkflowRefPurgeReport struct {
	RefName      string                       `json:"ref_name"`
	DeletedDatas []WorkflowRefDataPurgeReport `json:"deleted_datas,omitempty"`
	Error        string                       `json:"error,omitempty"`
}

type WorkflowRefDataPurgeReport struct {
	RunID     string `json:"run_id"`
	RunNumber int64  `json:"run_number"`
}

func (pr *PurgeReport) ComputeStatus() PurgeStatus {
	for _, w := range pr.Workflows {
		if w.Error != "" {
			return PurgeStatusFail
		}
		for _, r := range w.Refs {
			if r.Error != "" {
				return PurgeStatusFail
			}
		}
	}
	return PurgeStatusSuccess
}
