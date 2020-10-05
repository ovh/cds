package sdk

type PurgeDryRunRequest struct {
	RetentionPolicy string `json:"retention_policy"`
}

type PurgeRunToDelete struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
	Num    int64  `json:"num"`
}
