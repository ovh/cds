package sdk

import jwt "github.com/dgrijalva/jwt-go"

type CDNObjectType string

const (
	CDNArtifactType CDNObjectType = "CDNArtifactType"
)

type CDNRequest struct {
	Type       CDNObjectType     `json:"type" yaml:"type"`
	ProjectKey string            `json:"project_key,omitempty" yaml:"project_key,omitempty"`
	Config     map[string]string `json:"config,omitempty" yaml:"config,omitempty"`
}

// CDNJWTClaims is the specific claims format for Worker JWT
type CDNJWTClaims struct {
	jwt.StandardClaims
	ServiceName string
	CDNRequest  CDNRequest
}
