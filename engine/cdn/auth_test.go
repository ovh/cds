package cdn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/authentication"
	"github.com/ovh/cds/engine/cdn/item"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
)

func Test_itemAccessMiddleware(t *testing.T) {
	s, db := newTestService(t)

	cdntest.ClearItem(t, context.TODO(), s.Mapper, db)
	myItem := sdk.CDNItem{
		ID:   sdk.UUID(),
		Type: sdk.CDNTypeItemStepLog,
		APIRef: sdk.CDNLogAPIRef{
			RunID:      1,
			WorkflowID: 1,
		},
		APIRefHash: sdk.RandomString(10),
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &myItem))

	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })
	mockClient := mock_cdsclient.NewMockInterface(ctrl)
	s.Client = mockClient

	sessionID := sdk.UUID()
	mockClient.EXPECT().
		WorkflowLogAccess(gomock.Any(), gomock.Any(), gomock.Any(), sessionID).
		DoAndReturn(func(ctx context.Context, projectKey, workflowName, sessionID string) error { return nil }).
		Times(1)

	signer, err := authentication.NewSigner("cdn-test", test.SigningKey)
	require.NoError(t, err)
	s.Common.ParsedAPIPublicKey = signer.GetVerifyKey()
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS512, sdk.AuthSessionJWTClaims{
		ID: sessionID,
		StandardClaims: jwt.StandardClaims{
			Issuer:    "test",
			Subject:   sdk.UUID(),
			Id:        sessionID,
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(time.Minute).Unix(),
		},
	})
	jwtTokenRaw, err := signer.SignJWT(jwtToken)
	require.NoError(t, err)

	config := &service.HandlerConfig{}

	req := assets.NewRequest(t, http.MethodGet, "", nil)
	w := httptest.NewRecorder()
	ctx, err := s.jwtMiddleware(context.TODO(), w, req, config)
	require.NoError(t, err)
	ctx, err = s.itemAccessMiddleware(ctx, w, req, config)
	assert.Error(t, err, "an error should be returned because no jwt was given")

	req = assets.NewJWTAuthentifiedRequest(t, jwtTokenRaw, http.MethodGet, "", nil)
	w = httptest.NewRecorder()
	ctx, err = s.jwtMiddleware(context.TODO(), w, req, config)
	require.NoError(t, err)
	require.NotEmpty(t, s.sessionID(ctx))
	t.Log("sessionID:", s.sessionID(ctx))

	err = s.itemAccessCheck(ctx, myItem)
	assert.NoError(t, err, "no error should be returned because a valid jwt was given")

	req = assets.NewJWTAuthentifiedRequest(t, jwtTokenRaw, http.MethodGet, "", nil)
	w = httptest.NewRecorder()
	ctx, err = s.jwtMiddleware(context.TODO(), w, req, config)
	require.NotEmpty(t, s.sessionID(ctx))

	require.NoError(t, err)
	err = s.itemAccessCheck(ctx, myItem)
	assert.NoError(t, err, "no error should be returned because a valid jwt was given and permission validated from cache")
}
