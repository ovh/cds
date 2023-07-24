package sdk

type CheckProjectAccess struct {
	ProjectKey string `json:"project_key"`
	Role       string `json:"role"`
	SessionID  string `json:"session_id"`
}
