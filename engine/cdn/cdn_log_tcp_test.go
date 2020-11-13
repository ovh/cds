package cdn

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/mitchellh/hashstructure"
	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/sdk/log/hook"
	"io"
	"io/ioutil"
	"strconv"
	"testing"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/log"
)

func TestWorkerLogCDNEnabled(t *testing.T) {
	defer gock.Off()
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)
	cfg := test.LoadTestingConf(t, sdk.TypeCDN)
	db, factory, store, cancel := test.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(cancel)
	jobQueueKey := cache.Key(keyJobLogQueue, "1")
	defer store.Delete(jobQueueKey)
	heatbeatKey := cache.Key(keyJobHearbeat, "1")
	defer store.Delete(heatbeatKey)
	defer logCache.Flush()

	// Create worker private key
	key, err := jws.NewRandomSymmetricKey(32)
	require.NoError(t, err)

	// Create worker signer
	sign, err := jws.NewHMacSigner(key)
	require.NoError(t, err)

	// Create cdn service
	s := Service{
		Cache:               store,
		Mapper:              m,
		DBConnectionFactory: factory,
	}
	tmpDir, err := ioutil.TempDir("", t.Name()+"-cdn-1-*")
	require.NoError(t, err)
	cdnUnits, err := storage.Init(context.TODO(), m, db.DbMap, sdk.NewGoRoutines(), storage.Configuration{
		HashLocatorSalt: "thisismysalt",
		Buffer: storage.BufferConfiguration{
			Name: "redis_buffer",
			Redis: storage.RedisBufferConfiguration{
				Host:     cfg["redisHost"],
				Password: cfg["redisPassword"],
			},
		},
		Storages: []storage.StorageConfiguration{
			{
				Name: "local_storage",
				Local: &storage.LocalStorageConfiguration{
					Path: tmpDir,
				},
			},
		},
	})
	require.NoError(t, err)
	s.Units = cdnUnits

	s.Cfg.Log.StepMaxSize = 1000
	s.GoRoutines = sdk.NewGoRoutines()

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

	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	t.Cleanup(cancel)

	go s.waitingJobs(ctx)

	require.NoError(t, s.handleLogMessage(context.TODO(), []byte(message)))

	<-ctx.Done()

	done := gock.IsDone()
	if !done {
		t.Error("Gock is not done")
		for _, m := range gock.Pending() {
			t.Errorf("PENDING %s %s", m.Request().Method, m.Request().URLStruct.String())
		}
		t.Fail()
	}

	// Check that service log is disabled
	featureEnabled, has := logCache.Get("cdn-job-logs-enabled-project-PKEY")
	require.True(t, has)
	require.True(t, featureEnabled.(bool))
}

func TestServiceLogCDNDisabled(t *testing.T) {
	defer gock.Off()
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)
	cfg := test.LoadTestingConf(t, sdk.TypeCDN)
	db, factory, store, cancel := test.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(cancel)
	jobQueueKey := cache.Key(keyJobLogQueue, "1")
	defer store.Delete(jobQueueKey)
	heatbeatKey := cache.Key(keyJobHearbeat, "1")
	defer store.Delete(heatbeatKey)
	defer logCache.Flush()

	// Create hatchery private key
	key, err := jws.NewRandomRSAKey()
	require.NoError(t, err)

	// Create worker signer
	sign, err := jws.NewSigner(key)
	require.NoError(t, err)

	// Create cdn service
	s := Service{
		Cache:               store,
		DBConnectionFactory: factory,
		Mapper:              m,
	}
	s.Cfg.Log.StepMaxSize = 1000
	s.GoRoutines = sdk.NewGoRoutines()
	tmpDir, err := ioutil.TempDir("", t.Name()+"-cdn-1-*")
	require.NoError(t, err)
	cdnUnits, err := storage.Init(context.TODO(), m, db.DbMap, sdk.NewGoRoutines(), storage.Configuration{
		HashLocatorSalt: "thisismysalt",
		Buffer: storage.BufferConfiguration{
			Name: "redis_buffer",
			Redis: storage.RedisBufferConfiguration{
				Host:     cfg["redisHost"],
				Password: cfg["redisPassword"],
			},
		},
		Storages: []storage.StorageConfiguration{
			{
				Name: "local_storage",
				Local: &storage.LocalStorageConfiguration{
					Path: tmpDir,
				},
			},
		},
	})
	require.NoError(t, err)
	s.Units = cdnUnits

	signature := log.Signature{
		Service: &log.SignatureService{
			WorkerName:      "my-worker-name",
			HatcheryID:      1,
			HatcheryName:    "my-hatchery-name",
			RequirementID:   1,
			RequirementName: "service-1",
		},
		ProjectKey: "PKEY",
		JobID:      2,
		NodeRunID:  2,
		Timestamp:  time.Now().UnixNano(),
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

	s.Client = cdsclient.New(cdsclient.Config{
		Host: "http://lolcat.host",
	})

	gock.InterceptClient(s.Client.(cdsclient.Raw).HTTPClient())

	gock.New("http://lolcat.host").Post("/queue/workflows/log/service").Reply(200)
	gock.New("http://lolcat.host").Post("/feature/enabled/cdn-job-logs").Reply(200).JSON(sdk.FeatureEnabledResponse{Name: "cdn-job-logs", Enabled: false})

	t0 := time.Now()
	require.NoError(t, s.handleLogMessage(context.TODO(), []byte(message)))

	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	t.Cleanup(cancel)

	go s.waitingJobs(ctx)

	<-ctx.Done()

	done := gock.IsDone()
	if !done {
		t.Logf("GOCK NOT ENDED %s", time.Now().Sub(t0))
		ps := gock.Pending()
		for i := range ps {
			r := ps[i]
			t.Logf("pending [%s] %s", r.Request().Method, r.Request().URLStruct.String())
		}
	}

	// Check that service log is disabled
	featureEnabled, has := logCache.Get("cdn-job-logs-enabled-project-PKEY")
	require.True(t, has)
	require.False(t, featureEnabled.(bool))
}

func TestStoreTruncatedLogs(t *testing.T) {
	t.SkipNow()
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)

	log.SetLogger(t)
	db, factory, cache, cancel := test.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(cancel)

	cdntest.ClearItem(t, context.TODO(), m, db)

	// Create cdn service
	s := Service{
		DBConnectionFactory: factory,
		Cache:               cache,
		Mapper:              m,
	}
	s.GoRoutines = sdk.NewGoRoutines()

	ctx, ccl := context.WithCancel(context.TODO())
	t.Cleanup(ccl)
	cdnUnits := newRunningStorageUnits(t, m, db.DbMap, ctx, 10)
	s.Units = cdnUnits

	hm := handledMessage{
		Msg: hook.Message{
			Full: "Bim bam boum",
		},
		IsTerminated: false,
		Signature: log.Signature{
			ProjectKey:   sdk.RandomString(10),
			WorkflowID:   1,
			WorkflowName: "MyWorklow",
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
	hashRef, err := hashstructure.Hash(apiRef, nil)
	require.NoError(t, err)

	it := sdk.CDNItem{
		Status:     sdk.CDNStatusItemIncoming,
		APIRefHash: strconv.FormatUint(hashRef, 10),
		APIRef:     apiRef,
		Type:       sdk.CDNTypeItemStepLog,
	}
	require.NoError(t, item.Insert(context.TODO(), m, db, &it))
	defer func() {
		_ = item.DeleteByID(db, it.ID)

	}()
	content := buildMessage(hm)
	err = s.storeLogs(context.TODO(), sdk.CDNTypeItemStepLog, hm.Signature, hm.IsTerminated, content, 0)
	require.NoError(t, err)

	hm.IsTerminated = true
	hm.Msg.Full = "End of step"

	content = buildMessage(hm)
	err = s.storeLogs(context.TODO(), sdk.CDNTypeItemStepLog, hm.Signature, hm.IsTerminated, content, 1)
	require.NoError(t, err)

	itemDB, err := item.LoadByID(context.TODO(), s.Mapper, db, it.ID)
	require.NoError(t, err)
	require.NotNil(t, itemDB)
	require.Equal(t, sdk.CDNStatusItemCompleted, itemDB.Status)
	require.NotEmpty(t, itemDB.Hash)
	require.NotEmpty(t, itemDB.MD5)
	require.NotZero(t, itemDB.Size)

	unit, err := storage.LoadUnitByName(context.TODO(), m, db, s.Units.Buffer.Name())
	require.NoError(t, err)
	require.NotNil(t, unit)

	itemUnit, err := storage.LoadItemUnitByUnit(context.TODO(), m, db, unit.ID, itemDB.ID)
	require.NoError(t, err)
	require.NotNil(t, itemUnit)

	_, lineCount, rc, _, err := s.getItemLogValue(ctx, sdk.CDNTypeItemStepLog, strconv.FormatUint(hashRef, 10), sdk.CDNReaderFormatText, 0, 10000, 1)
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, rc)
	require.NoError(t, err)

	require.Equal(t, "[EMERGENCY] Bim bam boum\n...truncated\n", buf.String())
	require.Equal(t, int64(2), lineCount)
}
