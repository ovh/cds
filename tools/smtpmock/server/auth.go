package main

import (
	"time"

	"github.com/gofrs/uuid"
	jwt "github.com/golang-jwt/jwt"
	"github.com/pkg/errors"
)

type tokenType string

const (
	signinToken  tokenType = "signin"
	sessionToken tokenType = "session"
)

type Claims struct {
	jwt.StandardClaims
	Type tokenType `json:"type"`
}

func NewSigninToken() (string, string, error) {
	subjectID := uuid.Must(uuid.NewV4()).String()

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS512, Claims{
		Type: signinToken,
		StandardClaims: jwt.StandardClaims{
			Issuer:   "smtpmock",
			Id:       subjectID,
			IssuedAt: time.Now().Unix(),
		},
	})

	token, err := sign(jwtToken)
	if err != nil {
		return "", "", err
	}

	return subjectID, token, nil
}

func CheckSigninToken(token string) (string, error) {
	jwtToken, err := jwt.ParseWithClaims(token, &Claims{}, verify)
	if err != nil {
		return "", errors.WithStack(err)
	}

	if claims, ok := jwtToken.Claims.(*Claims); ok && jwtToken.Valid && claims.Type == signinToken {
		return claims.Id, nil
	}

	return "", errors.New("invalid given signin jwt")
}

func NewSessionToken(subjectID string) (string, string, error) {
	sessionID := uuid.NewV4().String()

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS512, Claims{
		Type: sessionToken,
		StandardClaims: jwt.StandardClaims{
			Issuer:    "smtpmock",
			Subject:   subjectID,
			Id:        sessionID,
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(time.Minute * 5).Unix(),
		},
	})

	token, err := sign(jwtToken)
	if err != nil {
		return "", "", err
	}

	return sessionID, token, nil
}

func CheckSessionToken(token string) (string, error) {
	jwtToken, err := jwt.ParseWithClaims(token, &Claims{}, verify)
	if err != nil {
		return "", errors.WithStack(err)
	}

	if claims, ok := jwtToken.Claims.(*Claims); ok && jwtToken.Valid && claims.Type == sessionToken {
		return claims.Id, nil
	}

	return "", errors.New("invalid given session jwt")
}
