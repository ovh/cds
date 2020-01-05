package sdk

type SigninRequest struct {
	SigninToken string `json:"signin-token"`
}

type SigninResponse struct {
	SessionToken string `json:"session-token"`
}
