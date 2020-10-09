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
