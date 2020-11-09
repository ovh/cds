package cdn

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/mitchellh/hashstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"

	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/authentication"
	"github.com/ovh/cds/engine/cdn/redis"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/log/hook"
)

func TestGetItemLogsLinesHandler(t *testing.T) {
	projectKey := sdk.RandomString(10)

	// Create cdn service with need storage and test item
	s, db := newTestService(t)
	s.Client = cdsclient.New(cdsclient.Config{Host: "http://lolcat.api", InsecureSkipVerifyTLS: false})
	gock.InterceptClient(s.Client.(cdsclient.Raw).HTTPClient())
	t.Cleanup(gock.Off)
	gock.New("http://lolcat.api").Get("/project/" + projectKey + "/workflows/MyWorkflow/log/access").Reply(http.StatusOK).JSON(nil)

	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)
	s.Units = newRunningStorageUnits(t, s.Mapper, db.DbMap, ctx, 1000)

	hm := handledMessage{
		Msg: hook.Message{
			Full: "this is a message",
		},
		IsTerminated: sdk.StatusTerminated,
		Line:         2,
		Signature: log.Signature{
			ProjectKey:   projectKey,
			WorkflowID:   1,
			WorkflowName: "MyWorkflow",
			RunID:        1,
			NodeRunID:    1,
			NodeRunName:  "MyPipeline",
			JobName:      "MyJob",
			JobID:        1,
			Worker: &log.SignatureWorker{
				StepName:  "script1",
				StepOrder: 1,
			},
		},
	}

	content := buildMessage(hm)
	err := s.storeLogs(context.TODO(), sdk.CDNTypeItemStepLog, hm.Signature, hm.IsTerminated, content, hm.Line)
	require.NoError(t, err)

	signer, err := authentication.NewSigner("cdn-test", test.SigningKey)
	require.NoError(t, err)
	s.Common.ParsedAPIPublicKey = signer.GetVerifyKey()
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS512, sdk.AuthSessionJWTClaims{
		ID: sdk.UUID(),
		StandardClaims: jwt.StandardClaims{
			Issuer:    "test",
			Subject:   sdk.UUID(),
			Id:        sdk.UUID(),
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(time.Minute).Unix(),
		},
	})
	jwtTokenRaw, err := signer.SignJWT(jwtToken)
	require.NoError(t, err)

	apiRef := sdk.CDNLogAPIRef{
		ProjectKey:     hm.Signature.ProjectKey,
		WorkflowName:   hm.Signature.WorkflowName,
		WorkflowID:     hm.Signature.WorkflowID,
		RunID:          hm.Signature.RunID,
		NodeRunName:    hm.Signature.NodeRunName,
		NodeRunID:      hm.Signature.NodeRunID,
		NodeRunJobName: hm.Signature.JobName,
		NodeRunJobID:   hm.Signature.JobID,
		StepName:       hm.Signature.Worker.StepName,
		StepOrder:      hm.Signature.Worker.StepOrder,
	}
	apiRefHashU, err := hashstructure.Hash(apiRef, nil)
	require.NoError(t, err)
	apiRefHash := strconv.FormatUint(apiRefHashU, 10)

	uri := s.Router.GetRoute("GET", s.getItemLogsLinesHandler, map[string]string{
		"type":   string(sdk.CDNTypeItemStepLog),
		"apiRef": apiRefHash,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtTokenRaw, "GET", uri, nil)
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	assert.Equal(t, "1", rec.Header().Get("X-Total-Count"))

	var lines []redis.Line
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &lines))
	require.Len(t, lines, 1)
	require.Equal(t, int64(2), lines[0].Number)
	require.Equal(t, "[EMERGENCY] this is a message\n", lines[0].Value)

	time.Sleep(1 * time.Second)
}

func TestGetItemLogsStreamHandler(t *testing.T) {
	projectKey := sdk.RandomString(10)

	// Create cdn service with need storage and test item
	s, db := newTestService(t)
	require.NoError(t, s.initWebsocket())
	ts := httptest.NewServer(s.Router.Mux)

	s.Client = cdsclient.New(cdsclient.Config{Host: "http://lolcat.api", InsecureSkipVerifyTLS: false})
	gock.InterceptClient(s.Client.(cdsclient.Raw).HTTPClient())
	t.Cleanup(gock.Off)
	gock.New("http://lolcat.api").Get("/project/" + projectKey + "/workflows/MyWorkflow/log/access").Reply(http.StatusOK).JSON(nil)

	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)
	s.Units = newRunningStorageUnits(t, s.Mapper, db.DbMap, ctx, 1000)

	signature := log.Signature{
		ProjectKey:   projectKey,
		WorkflowID:   1,
		WorkflowName: "MyWorkflow",
		RunID:        1,
		NodeRunID:    1,
		NodeRunName:  "MyPipeline",
		JobName:      "MyJob",
		JobID:        1,
		Worker: &log.SignatureWorker{
			StepName:  "script1",
			StepOrder: 1,
		},
	}

	signer, err := authentication.NewSigner("cdn-test", test.SigningKey)
	require.NoError(t, err)
	s.Common.ParsedAPIPublicKey = signer.GetVerifyKey()
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS512, sdk.AuthSessionJWTClaims{
		ID: sdk.UUID(),
		StandardClaims: jwt.StandardClaims{
			Issuer:    "test",
			Subject:   sdk.UUID(),
			Id:        sdk.UUID(),
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(time.Minute).Unix(),
		},
	})
	jwtTokenRaw, err := signer.SignJWT(jwtToken)
	require.NoError(t, err)

	apiRef := sdk.CDNLogAPIRef{
		ProjectKey:     signature.ProjectKey,
		WorkflowName:   signature.WorkflowName,
		WorkflowID:     signature.WorkflowID,
		RunID:          signature.RunID,
		NodeRunName:    signature.NodeRunName,
		NodeRunID:      signature.NodeRunID,
		NodeRunJobName: signature.JobName,
		NodeRunJobID:   signature.JobID,
		StepName:       signature.Worker.StepName,
		StepOrder:      signature.Worker.StepOrder,
	}
	apiRefHashU, err := hashstructure.Hash(apiRef, nil)
	require.NoError(t, err)
	apiRefHash := strconv.FormatUint(apiRefHashU, 10)

	var messageCounter int64
	sendMessage := func() {
		hm := handledMessage{
			Msg:          hook.Message{Full: fmt.Sprintf("message %d", messageCounter)},
			IsTerminated: sdk.StatusNotTerminated,
			Line:         messageCounter,
			Signature:    signature,
		}
		content := buildMessage(hm)
		err = s.storeLogs(context.TODO(), sdk.CDNTypeItemStepLog, hm.Signature, hm.IsTerminated, content, hm.Line)
		require.NoError(t, err)
		messageCounter++
	}

	client := cdsclient.New(cdsclient.Config{
		Host:                  ts.URL,
		InsecureSkipVerifyTLS: true,
		SessionToken:          jwtTokenRaw,
	})

	uri := s.Router.GetRoute("GET", s.getItemLogsStreamHandler, nil)
	require.NotEmpty(t, uri)

	// Send some messages before stream
	for i := 0; i < 10; i++ {
		sendMessage()
	}

	// Open connection
	ctx, cancel = context.WithTimeout(context.TODO(), time.Second*10)
	t.Cleanup(func() { cancel() })
	chanMsgToSend := make(chan json.RawMessage)
	chanMsgReceived := make(chan json.RawMessage, 10)
	chanErrorReceived := make(chan error, 10)
	go func() {
		chanErrorReceived <- client.RequestWebsocket(ctx, sdk.NewGoRoutines(), uri, chanMsgToSend, chanMsgReceived, chanErrorReceived)
	}()
	buf, err := json.Marshal(sdk.CDNStreamFilter{
		ItemType: sdk.CDNTypeItemStepLog,
		APIRef:   apiRefHash,
		Offset:   0,
	})
	require.NoError(t, err)
	chanMsgToSend <- buf

	var lines []redis.Line
	for ctx.Err() == nil && len(lines) < 10 {
		select {
		case <-ctx.Done():
			break
		case err := <-chanErrorReceived:
			require.NoError(t, err)
			break
		case msg := <-chanMsgReceived:
			var line redis.Line
			require.NoError(t, json.Unmarshal(msg, &line))
			lines = append(lines, line)
		}
	}

	require.Len(t, lines, 10)
	require.Equal(t, "[EMERGENCY] message 0\n", lines[0].Value)
	require.Equal(t, int64(0), lines[0].Number)
	require.Equal(t, "[EMERGENCY] message 9\n", lines[9].Value)
	require.Equal(t, int64(9), lines[9].Number)

	// Send some messages
	for i := 0; i < 10; i++ {
		sendMessage()
	}

	for ctx.Err() == nil && len(lines) < 20 {
		select {
		case <-ctx.Done():
			break
		case err := <-chanErrorReceived:
			require.NoError(t, err)
			break
		case msg := <-chanMsgReceived:
			var line redis.Line
			require.NoError(t, json.Unmarshal(msg, &line))
			lines = append(lines, line)
		}
	}

	require.Len(t, lines, 20)
	require.Equal(t, "[EMERGENCY] message 19\n", lines[19].Value)
	require.Equal(t, int64(19), lines[19].Number)

	// Try another connection with offset
	ctx, cancel = context.WithTimeout(context.TODO(), time.Second*10)
	t.Cleanup(func() { cancel() })
	go func() {
		chanErrorReceived <- client.RequestWebsocket(ctx, sdk.NewGoRoutines(), uri, chanMsgToSend, chanMsgReceived, chanErrorReceived)
	}()
	buf, err = json.Marshal(sdk.CDNStreamFilter{
		ItemType: sdk.CDNTypeItemStepLog,
		APIRef:   apiRefHash,
		Offset:   15,
	})
	require.NoError(t, err)
	chanMsgToSend <- buf

	lines = make([]redis.Line, 0)
	for ctx.Err() == nil && len(lines) < 5 {
		select {
		case <-ctx.Done():
			break
		case err := <-chanErrorReceived:
			require.NoError(t, err)
			break
		case msg := <-chanMsgReceived:
			var line redis.Line
			require.NoError(t, json.Unmarshal(msg, &line))
			lines = append(lines, line)
		}
	}

	require.Len(t, lines, 5)
	require.Equal(t, "[EMERGENCY] message 15\n", lines[0].Value)
	require.Equal(t, int64(15), lines[0].Number)
	require.Equal(t, "[EMERGENCY] message 19\n", lines[4].Value)
	require.Equal(t, int64(19), lines[4].Number)
}
