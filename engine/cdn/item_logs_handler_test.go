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

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/sdk/cdn"

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
	"github.com/ovh/cds/sdk/log/hook"
)

func TestGetItemsAllLogsLinesHandler(t *testing.T) {
	projectKey := sdk.RandomString(10)
	s, db := newTestService(t)

	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	t.Cleanup(cancel)

	cdntest.ClearItem(t, ctx, s.Mapper, db)
	cdntest.ClearSyncRedisSet(t, s.Cache, "local_storage")

	s.Units = newRunningStorageUnits(t, s.Mapper, db.DbMap, ctx, s.Cache)

	// Add step 1
	hm1 := handledMessage{
		Msg: hook.Message{
			Full: "this is a message",
		},
		IsTerminated: sdk.StatusTerminated,
		Signature: cdn.Signature{
			ProjectKey:   projectKey,
			WorkflowID:   1,
			WorkflowName: "MyWorkflow",
			RunID:        1,
			NodeRunID:    1,
			NodeRunName:  "MyPipeline",
			JobName:      "MyJob",
			JobID:        1,
			Worker: &cdn.SignatureWorker{
				StepName:  "script1",
				StepOrder: 0,
			},
		},
	}
	content := buildMessage(hm1)
	err := s.storeLogs(context.TODO(), sdk.CDNTypeItemStepLog, hm1.Signature, hm1.IsTerminated, content)
	require.NoError(t, err)
	hash1 := sdk.CDNLogAPIRef{
		ProjectKey:     hm1.Signature.ProjectKey,
		WorkflowName:   hm1.Signature.WorkflowName,
		WorkflowID:     hm1.Signature.WorkflowID,
		RunID:          hm1.Signature.RunID,
		NodeRunName:    hm1.Signature.NodeRunName,
		NodeRunID:      hm1.Signature.NodeRunID,
		NodeRunJobName: hm1.Signature.JobName,
		NodeRunJobID:   hm1.Signature.JobID,
		StepOrder:      hm1.Signature.Worker.StepOrder,
		StepName:       hm1.Signature.Worker.StepName,
	}
	hashRef1, err := hash1.ToHash()
	require.NoError(t, err)

	// Add step 2
	hm2 := hm1
	hm2.Signature.Worker.StepOrder = 1
	content = buildMessage(hm2)
	err = s.storeLogs(context.TODO(), sdk.CDNTypeItemStepLog, hm2.Signature, hm2.IsTerminated, content)
	require.NoError(t, err)
	hash2 := sdk.CDNLogAPIRef{
		ProjectKey:     hm2.Signature.ProjectKey,
		WorkflowName:   hm2.Signature.WorkflowName,
		WorkflowID:     hm2.Signature.WorkflowID,
		RunID:          hm2.Signature.RunID,
		NodeRunName:    hm2.Signature.NodeRunName,
		NodeRunID:      hm2.Signature.NodeRunID,
		NodeRunJobName: hm2.Signature.JobName,
		NodeRunJobID:   hm2.Signature.JobID,
		StepOrder:      hm2.Signature.Worker.StepOrder,
		StepName:       hm2.Signature.Worker.StepName,
	}
	hashRef2, err := hash2.ToHash()
	require.NoError(t, err)

	// Add step 3
	hm3 := hm1
	hm3.Signature.Worker.StepOrder = 2
	hm3.Msg.Full = "First Line"
	hm3.IsTerminated = false
	content = buildMessage(hm3)
	err = s.storeLogs(context.TODO(), sdk.CDNTypeItemStepLog, hm3.Signature, hm3.IsTerminated, content)
	require.NoError(t, err)
	hash3 := sdk.CDNLogAPIRef{
		ProjectKey:     hm3.Signature.ProjectKey,
		WorkflowName:   hm3.Signature.WorkflowName,
		WorkflowID:     hm3.Signature.WorkflowID,
		RunID:          hm3.Signature.RunID,
		NodeRunName:    hm3.Signature.NodeRunName,
		NodeRunID:      hm3.Signature.NodeRunID,
		NodeRunJobName: hm3.Signature.JobName,
		NodeRunJobID:   hm3.Signature.JobID,
		StepOrder:      hm3.Signature.Worker.StepOrder,
		StepName:       hm3.Signature.Worker.StepName,
	}

	hm4 := hm3
	hm4.Msg.Full = "Second Line"
	hm4.IsTerminated = true
	content = buildMessage(hm4)
	err = s.storeLogs(context.TODO(), sdk.CDNTypeItemStepLog, hm4.Signature, hm4.IsTerminated, content)
	require.NoError(t, err)

	hashRef3, err := hash3.ToHash()
	require.NoError(t, err)

	it, err := item.LoadByAPIRefHashAndType(ctx, s.Mapper, s.mustDBWithCtx(ctx), hashRef3, sdk.CDNTypeItemStepLog)
	require.NoError(t, err)

	unit, err := storage.LoadUnitByName(ctx, s.Mapper, s.mustDBWithCtx(ctx), s.Units.Storages[0].Name())
	require.NoError(t, err)

	require.NoError(t, s.Units.FillWithUnknownItems(ctx, s.Units.Storages[0], 1000))
	require.NoError(t, s.Units.FillSyncItemChannel(ctx, s.Units.Storages[0], 1000))
	time.Sleep(1 * time.Second)

	cpt := 0
	for {
		cpt++
		_, err = storage.LoadItemUnitByUnit(ctx, s.Mapper, s.mustDBWithCtx(ctx), unit.ID, it.ID)
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			if cpt == 10 {
				t.Fail()
			}
			time.Sleep(250 * time.Millisecond)
			continue
		}
		if err != nil {
			t.Fail()
		}
		break
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

	uri := fmt.Sprintf("%s?apiRefHash=%s&apiRefHash=%s&apiRefHash=%s", s.Router.GetRoute("GET", s.getItemsAllLogsLinesHandler, map[string]string{
		"type": string(sdk.CDNTypeItemStepLog),
	}), hashRef1, hashRef2, hashRef3)
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtTokenRaw, "GET", uri, nil)
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	var logslines []sdk.CDNLogsLines
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &logslines))
	require.Len(t, logslines, 3)

	require.Equal(t, int64(1), logslines[0].LinesCount)
	require.Equal(t, int64(1), logslines[1].LinesCount)
	require.Equal(t, int64(2), logslines[2].LinesCount)
}

func TestGetItemLogsLinesHandler(t *testing.T) {
	projectKey := sdk.RandomString(10)

	// Create cdn service with need storage and test item
	s, db := newTestService(t)
	s.Client = cdsclient.New(cdsclient.Config{Host: "http://lolcat.api", InsecureSkipVerifyTLS: false})
	gock.InterceptClient(s.Client.(cdsclient.Raw).HTTPClient())
	t.Cleanup(gock.Off)
	gock.New("http://lolcat.api").Get("/project/" + projectKey + "/workflows/1/type/step-log/access").Reply(http.StatusOK).JSON(nil)

	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)
	s.Units = newRunningStorageUnits(t, s.Mapper, db.DbMap, ctx, s.Cache)

	hm := handledMessage{
		Msg: hook.Message{
			Full: "this is a message",
		},
		IsTerminated: sdk.StatusTerminated,
		Signature: cdn.Signature{
			ProjectKey:   projectKey,
			WorkflowID:   1,
			WorkflowName: "MyWorkflow",
			RunID:        1,
			NodeRunID:    1,
			NodeRunName:  "MyPipeline",
			JobName:      "MyJob",
			JobID:        1,
			Worker: &cdn.SignatureWorker{
				StepName:  "script1",
				StepOrder: 1,
			},
		},
	}

	content := buildMessage(hm)
	err := s.storeLogs(context.TODO(), sdk.CDNTypeItemStepLog, hm.Signature, hm.IsTerminated, content)
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
	require.Equal(t, int64(0), lines[0].Number)
	require.Equal(t, "[EMERGENCY] this is a message\n", lines[0].Value)

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
	gock.New("http://lolcat.api").Get("/project/" + projectKey + "/workflows/1/type/step-log/access").Times(1).Reply(http.StatusOK).JSON(nil)

	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)
	s.Units = newRunningStorageUnits(t, s.Mapper, db.DbMap, ctx, s.Cache)

	signature := cdn.Signature{
		ProjectKey:   projectKey,
		WorkflowID:   1,
		WorkflowName: "MyWorkflow",
		RunID:        1,
		NodeRunID:    1,
		NodeRunName:  "MyPipeline",
		JobName:      "MyJob",
		JobID:        123456789,
		Worker: &cdn.SignatureWorker{
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

	var messageCounter int64
	sendMessage := func() {
		hm := handledMessage{
			Msg:          hook.Message{Full: fmt.Sprintf("message %d", messageCounter)},
			IsTerminated: sdk.StatusNotTerminated,
			Signature:    signature,
		}
		content := buildMessage(hm)
		err = s.storeLogs(context.TODO(), sdk.CDNTypeItemStepLog, hm.Signature, hm.IsTerminated, content)
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
		chanErrorReceived <- client.RequestWebsocket(ctx, sdk.NewGoRoutines(ctx), uri, chanMsgToSend, chanMsgReceived, chanErrorReceived)
	}()
	buf, err := json.Marshal(sdk.CDNStreamFilter{
		JobRunID: signature.JobID,
	})
	require.NoError(t, err)
	chanMsgToSend <- buf

	var lines []redis.Line
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
	require.Equal(t, "[EMERGENCY] message 5\n", lines[0].Value)
	require.Equal(t, int64(5), lines[0].Number)
	require.Equal(t, "[EMERGENCY] message 9\n", lines[4].Value)
	require.Equal(t, int64(9), lines[4].Number)

	// Send some messages
	for i := 0; i < 10; i++ {
		sendMessage()
	}

	for ctx.Err() == nil && len(lines) < 15 {
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

	require.Len(t, lines, 15)
	require.Equal(t, "[EMERGENCY] message 19\n", lines[14].Value)
	require.Equal(t, int64(19), lines[14].Number)
}
