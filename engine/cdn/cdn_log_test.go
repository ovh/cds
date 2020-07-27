package cdn

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/log"
)

func TestWorkerLogCDNEnabled(t *testing.T) {
	defer gock.Off()
	m := gorpmapper.New()
	db, cache := test.SetupPGWithMapper(t, m, sdk.TypeCDN)
	defer cache.Delete("cdn:log:job:1")
	defer cache.Delete(keyJobLogIncomingQueue)
	defer logCache.Flush()
	// Create worker private key
	key, err := jws.NewRandomSymmetricKey(32)
	require.NoError(t, err)

	// Create worker signer
	sign, err := jws.NewHMacSigner(key)
	require.NoError(t, err)

	// Create cdn service
	s := Service{
		Db:    db.DbMap,
		Cache: cache,
	}

	signature := log.Signature{
		Worker: &log.SignatureWorker{
			WorkerID:   "abcdef-123456",
			StepOrder:  0,
			WorkerName: "myworker",
		},
		ProjectKey: "PKEY",
		JobID:      1,
		NodeRunID:  1,
		Timestamp:  time.Now().UnixNano(),
	}
	logCache.Set(fmt.Sprintf("worker-%s", signature.Worker.WorkerName), sdk.Worker{
		Name:       signature.Worker.WorkerName,
		ID:         signature.Worker.WorkerID,
		PrivateKey: key,
		JobRunID:   &signature.JobID,
	}, gocache.DefaultExpiration)

	signatureField, err := jws.Sign(sign, signature)
	require.NoError(t, err)

	message := `{"level": 1, "version": "1", "short": "this", "_facility": "fa", "_file": "file",
	"host": "host", "_line":1, "_pid": 1, "_prefix": "prefix", "full_message": "this is my message", "_Signature": "%s"}`
	message = fmt.Sprintf(message, signatureField)

	s.Client = cdsclient.New(cdsclient.Config{
		Host: "http://lolcat.host",
	})

	gock.InterceptClient(s.Client.(cdsclient.Raw).HTTPClient())

	gock.New("http://lolcat.host").Post("/queue/workflows/1/log").Reply(200)
	gock.New("http://lolcat.host").Post("/feature/enabled/cdn-job-logs").Reply(200).JSON(sdk.FeatureEnabledResponse{Name: "cdn-job-logs", Enabled: true})

	require.NoError(t, s.handleLogMessage(context.TODO(), []byte(message)))

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	go s.waitingJobs(ctx)

	cpt := 0
	for {
		done := gock.IsDone()
		if !done {
			if cpt > 20 {
				t.Fail()
				break
			}
			cpt++
			time.Sleep(250 * time.Millisecond)
			continue
		}
		break
	}

	// Check that service log is disabled
	featureEnabled, has := logCache.Get("cdn-job-logs-enabled-project-PKEY")
	require.True(t, has)
	require.True(t, featureEnabled.(bool))

	b, err := s.Cache.Exist(keyJobLogIncomingQueue)
	require.NoError(t, err)
	require.True(t, b)
}

func TestWorkerLogCDNDisabled(t *testing.T) {
	defer gock.Off()
	m := gorpmapper.New()
	db, cache := test.SetupPGWithMapper(t, m, sdk.TypeCDN)
	defer cache.Delete(keyJobLogIncomingQueue)
	defer cache.Delete("cdn:log:job:1")
	defer logCache.Flush()

	// Create worker private key
	key, err := jws.NewRandomSymmetricKey(32)
	require.NoError(t, err)

	// Create worker signer
	sign, err := jws.NewHMacSigner(key)
	require.NoError(t, err)

	// Create cdn service
	s := Service{
		Db:    db.DbMap,
		Cache: cache,
	}

	signature := log.Signature{
		Worker: &log.SignatureWorker{
			WorkerID:   "abcdef-123456",
			StepOrder:  0,
			WorkerName: "myworker",
		},
		ProjectKey: "PKEY",
		JobID:      2,
		NodeRunID:  2,
		Timestamp:  time.Now().UnixNano(),
	}
	logCache.Set(fmt.Sprintf("worker-%s", signature.Worker.WorkerName), sdk.Worker{
		Name:       signature.Worker.WorkerName,
		ID:         signature.Worker.WorkerID,
		PrivateKey: key,
		JobRunID:   &signature.JobID,
	}, gocache.DefaultExpiration)

	signatureField, err := jws.Sign(sign, signature)
	require.NoError(t, err)

	message := `{"level": 1, "version": "1", "short": "this", "_facility": "fa", "_file": "file",
	"host": "host", "_line":1, "_pid": 1, "_prefix": "prefix", "full_message": "this is my message", "_Signature": "%s"}`
	message = fmt.Sprintf(message, signatureField)

	s.Client = cdsclient.New(cdsclient.Config{
		Host: "http://lolcat.host",
	})

	gock.InterceptClient(s.Client.(cdsclient.Raw).HTTPClient())

	gock.New("http://lolcat.host").Post("/queue/workflows/2/log").Reply(200)
	gock.New("http://lolcat.host").Post("/feature/enabled/cdn-job-logs").Reply(200).JSON(sdk.FeatureEnabledResponse{Name: "cdn-job-logs", Enabled: false})

	require.NoError(t, s.handleLogMessage(context.TODO(), []byte(message)))

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	go s.waitingJobs(ctx)

	cpt := 0
	for {
		done := gock.IsDone()
		if !done {
			if cpt > 20 {
				t.Logf("GOCK NOT ENDED")
				ps := gock.Pending()
				for i := range ps {
					r := ps[i]
					t.Logf("[%s] %s", r.Request().Method, r.Request().URLStruct.String())
				}
				t.Fail()
				break
			}
			cpt++
			time.Sleep(250 * time.Millisecond)
			continue
		}
		break
	}

	// Check that service log is disabled
	featureEnabled, has := logCache.Get("cdn-job-logs-enabled-project-PKEY")
	require.True(t, has)
	require.False(t, featureEnabled.(bool))

	b, err := s.Cache.Exist(keyJobLogIncomingQueue)
	require.NoError(t, err)
	require.False(t, b)
}

func TestServiceLog(t *testing.T) {
	defer gock.Off()
	mCDN := gorpmapper.New()
	dbCDN, cacheCDN := test.SetupPGWithMapper(t, mCDN, sdk.TypeCDN)
	defer cacheCDN.Delete(keyServiceLogIncomingQueue)
	defer logCache.Flush()

	// Create hatchery private key
	key, err := jws.NewRandomRSAKey()
	require.NoError(t, err)

	// Create worker signer
	sign, err := jws.NewSigner(key)
	require.NoError(t, err)

	// Create cdn service
	s := Service{
		Db:    dbCDN.DbMap,
		Cache: cacheCDN,
	}
	s.Client = cdsclient.New(cdsclient.Config{
		Host: "http://lolcat.host",
	})
	gock.InterceptClient(s.Client.(cdsclient.Raw).HTTPClient())

	signature := log.Signature{
		Service: &log.SignatureService{
			WorkerName:      "my-worker-name",
			HatcheryID:      1,
			HatcheryName:    "my-hatchery-name",
			RequirementID:   1,
			RequirementName: "service-1",
		},
		JobID:     1,
		NodeRunID: 1,
		Timestamp: time.Now().UnixNano(),
	}

	// Create worker private key
	wKey, err := jws.NewRandomSymmetricKey(32)
	require.NoError(t, err)
	w := sdk.Worker{
		Name:       signature.Service.WorkerName,
		HatcheryID: &signature.Service.HatcheryID,
		PrivateKey: []byte(base64.StdEncoding.EncodeToString(wKey)),
		JobRunID:   &signature.JobID,
	}

	logCache.Set(fmt.Sprintf("hatchery-key-%d", signature.Service.HatcheryID), &key.PublicKey, gocache.DefaultExpiration)
	logCache.Set(fmt.Sprintf("worker-%s", signature.Service.WorkerName), w, gocache.DefaultExpiration)

	signatureField, err := jws.Sign(sign, signature)
	require.NoError(t, err)

	message := `{"level": 1, "version": "1", "short": "this", "_facility": "fa", "_file": "file",
	"host": "host", "_line":1, "_pid": 1, "_prefix": "prefix", "full_message": "this is my service message", "_Signature": "%s"}`
	message = fmt.Sprintf(message, signatureField)

	gock.New("http://lolcat.host").Post("/queue/workflows/log/service").Reply(200)
	gock.New("http://lolcat.host").Post("/feature/enabled/cdn-service-logs").Reply(200).JSON(sdk.FeatureEnabledResponse{Name: "cdn-service-logs", Enabled: false})

	require.NoError(t, s.handleLogMessage(context.TODO(), []byte(message)))

	// Check that service log is disabled
	featureEnabled, has := logCache.Get("cdn-service-logs-enabled")
	require.True(t, has)
	require.False(t, featureEnabled.(bool))

	b, err := s.Cache.Exist(keyServiceLogIncomingQueue)
	require.NoError(t, err)
	require.False(t, b)
}
