package vcs

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

// Test_authMiddleware_exposesCallIdentityAsLogFields checks the credentials that served a vcs call
// are turned into log fields. Without them, an error coming back from a forge cannot be told apart
// from a permission problem, which is what made the "branch not found in repository" incident
// impossible to diagnose. The token must never become a log field.
func Test_authMiddleware_exposesCallIdentityAsLogFields(t *testing.T) {
	s := new(Service)

	req, err := http.NewRequest("GET", "/vcs/my-bitbucket/repos/MANAGER/manager-core-manifests/branches/?branch=master", nil)
	require.NoError(t, err)
	req = mux.SetURLVars(req, map[string]string{
		"name":  "my-bitbucket",
		"owner": "MANAGER",
		"repo":  "manager-core-manifests",
	})
	req.Header.Set(sdk.HeaderXVCSType, base64.StdEncoding.EncodeToString([]byte(sdk.VCSTypeBitbucketServer)))
	req.Header.Set(sdk.HeaderXVCSURL, base64.StdEncoding.EncodeToString([]byte("https://bitbucket.example.com")))
	req.Header.Set(sdk.HeaderXVCSUsername, base64.StdEncoding.EncodeToString([]byte("cds-bot")))
	req.Header.Set(sdk.HeaderXVCSToken, base64.StdEncoding.EncodeToString([]byte("super-secret-token")))
	req.Header.Set(sdk.HeaderXVCSProjectKey, base64.StdEncoding.EncodeToString([]byte("MANAGER")))

	ctx, err := s.authMiddleware(context.TODO(), httptest.NewRecorder(), req, nil)
	require.NoError(t, err)

	assert.Equal(t, "cds-bot", cdslog.ContextValue(ctx, cdslog.VCSUsername))
	assert.Equal(t, sdk.VCSTypeBitbucketServer, cdslog.ContextValue(ctx, cdslog.VCSType))
	assert.Equal(t, "https://bitbucket.example.com", cdslog.ContextValue(ctx, cdslog.VCSURL))
	assert.Equal(t, "MANAGER", cdslog.ContextValue(ctx, cdslog.Project))
	assert.Equal(t, "my-bitbucket", cdslog.ContextValue(ctx, cdslog.VCSServer))
	assert.Equal(t, "MANAGER/manager-core-manifests", cdslog.ContextValue(ctx, cdslog.Repository))

	// a field that is not registered is silently dropped by rockbears/log, even when it sits in the
	// context: without this check the fields above could be added and never emitted
	registered := map[log.Field]bool{}
	for _, f := range log.GetRegisteredFields() {
		registered[f] = true
	}
	for _, f := range []log.Field{cdslog.VCSUsername, cdslog.VCSType, cdslog.VCSURL, cdslog.Project, cdslog.VCSServer, cdslog.Repository} {
		assert.True(t, registered[f], "log field %q is not registered, it would never be emitted", f)
	}

	for _, f := range log.GetRegisteredFields() {
		assert.NotEqual(t, "super-secret-token", cdslog.ContextValue(ctx, f), "the vcs token must never be exposed as a log field %q", f)
	}
}

// logCapture collects the entries emitted through rockbears/log. Factory() is called once per entry,
// so every entry gets its own wrapper writing into the shared sink.
type logCapture struct {
	mutex   sync.Mutex
	entries []map[string]string
}

func (c *logCapture) factory() log.Wrapper {
	entry := map[string]string{}
	c.mutex.Lock()
	c.entries = append(c.entries, entry)
	c.mutex.Unlock()
	return &captureWrapper{capture: c, entry: entry}
}

// entriesWith returns the captured entries holding every given field.
func (c *logCapture) entriesWith(fields ...log.Field) []map[string]string {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	var found []map[string]string
	for _, e := range c.entries {
		complete := true
		for _, f := range fields {
			if _, has := e[string(f)]; !has {
				complete = false
				break
			}
		}
		if complete {
			found = append(found, e)
		}
	}
	return found
}

type captureWrapper struct {
	capture *logCapture
	entry   map[string]string
}

func (w *captureWrapper) GetLevel() log.Level { return log.LevelDebug }
func (w *captureWrapper) WithField(key string, value interface{}) {
	w.capture.mutex.Lock()
	defer w.capture.mutex.Unlock()
	w.entry[key] = fmt.Sprintf("%v", value)
}

// captureMessageKey holds the formatted message of an entry, alongside its fields.
const captureMessageKey = "_message"

func (w *captureWrapper) record(format string, args ...interface{}) {
	w.capture.mutex.Lock()
	defer w.capture.mutex.Unlock()
	if len(args) == 0 {
		w.entry[captureMessageKey] = format
		return
	}
	w.entry[captureMessageKey] = fmt.Sprintf(format, args...)
}

func (w *captureWrapper) Debugf(format string, args ...interface{}) { w.record(format, args...) }
func (w *captureWrapper) Infof(format string, args ...interface{})  { w.record(format, args...) }
func (w *captureWrapper) Warnf(format string, args ...interface{})  { w.record(format, args...) }
func (w *captureWrapper) Fatalf(format string, args ...interface{}) { w.record(format, args...) }
func (w *captureWrapper) Errorf(format string, args ...interface{}) { w.record(format, args...) }
func (w *captureWrapper) Panicf(format string, args ...interface{}) { w.record(format, args...) }

// Test_getBranchHandler_logsWhichCredentialsWereUsed is the end of the chain the incident exposed: on
// the real router, the log line that carries the stack trace must also carry which credentials, which
// project and which repository the failing call used.
func Test_getBranchHandler_logsWhichCredentialsWereUsed(t *testing.T) {
	s, err := newTestService(t)
	require.NoError(t, err)

	fake := &fakeBitbucket{
		listing:       listingWithoutMaster,
		defaultBranch: `{"id":"refs/heads/main","displayId":"main","latestChangeset":"deadbeef","isDefault":true}`,
	}
	srv := fake.server(t)
	defer srv.Close()

	capture := &logCapture{}
	previousFactory := log.Factory
	log.Factory = capture.factory
	defer func() { log.Factory = previousFactory }()

	rec := callGetBranchHandler(t, s, srv.URL)
	require.Equal(t, 404, rec.Code, "body: %s", rec.Body.String())

	// the line that carries the stack trace is the one an operator reads when the error is reported
	entries := capture.entriesWith(cdslog.Stacktrace, cdslog.VCSUsername, cdslog.Project, cdslog.VCSServer, cdslog.Repository)
	require.NotEmpty(t, entries, "no log entry carries both the stack trace and the call identity")

	e := entries[0]
	assert.Equal(t, "cds-bot", e[string(cdslog.VCSUsername)])
	assert.Equal(t, "MANAGER", e[string(cdslog.Project)])
	assert.Equal(t, "my-bitbucket", e[string(cdslog.VCSServer)])
	assert.Equal(t, "MANAGER/manager-core-manifests", e[string(cdslog.Repository)])
	assert.Equal(t, sdk.VCSTypeBitbucketServer, e[string(cdslog.VCSType)])
	assert.NotContains(t, e, "vcs_token")

	// the details must reach the logged message too, not only the json payload returned to the api:
	// this is the line an operator greps when the error is reported
	assert.Contains(t, e[captureMessageKey], `filterText="master"`)
	assert.Contains(t, e[captureMessageKey], `default branch is "main"`)
}
