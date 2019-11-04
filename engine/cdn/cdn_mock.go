package cdn

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
)

func init() {
	log.Initialize(&log.Conf{Level: "debug"})
}

func InitMock(t *testing.T) {
	privKey, _ := jws.NewRandomRSAKey()
	privKeyPEM, _ := jws.ExportPrivateKey(privKey)
	pubKey, _ := jws.ExportPublicKey(privKey)

	require.NoError(t, authentication.Init("cds-test", privKeyPEM))
	id := sdk.UUID()
	consumerID := sdk.UUID()
	hatcheryAuthenticationToken, _ := authentication.NewSessionJWT(&sdk.AuthSession{
		ID:         id,
		ConsumerID: consumerID,
		ExpireAt:   time.Now().Add(time.Hour),
	})

	var checkRequest gock.ObserverFunc = func(request *http.Request, mock gock.Mock) {
		if request.Body == nil {
			return
		}
		bodyContent, err := ioutil.ReadAll(request.Body)
		assert.NoError(t, err)
		request.Body = ioutil.NopCloser(bytes.NewReader(bodyContent))
		if mock != nil {
			t.Logf("%s %s - Body: %s", mock.Request().Method, mock.Request().URLStruct.String(), string(bodyContent))
		}
	}

	gock.New("http://cdstest.test").Post("/auth/consumer/builtin/signin").
		Reply(201).
		JSON(
			sdk.AuthConsumerSigninResponse{
				Token: hatcheryAuthenticationToken,
				User: &sdk.AuthentifiedUser{
					Username: "admin",
				},
			},
		).AddHeader("X-Api-Pub-Signing-Key", base64.StdEncoding.EncodeToString(pubKey))

	gock.New("http://cdstest.test").Post("/services/register").
		HeaderPresent("Authorization").
		Reply(200).
		JSON(sdk.Service{})

	gock.New("http://cdstest.test").Post("/services/heartbeat").
		HeaderPresent("Authorization").
		Reply(204)

	gock.New("http://cdstest.test").Post("/project/test/storage/shared.infra/artifact/test/url/callback").
		HeaderPresent("Authorization").
		Reply(200)

	gock.Observe(checkRequest)

}

func newMockSSERoundTripper(t *testing.T, ctx context.Context) *MockSSERoundTripper {
	var m = MockSSERoundTripper{
		t: t,
		c: ctx,
	}
	m.reader, m.writer = io.Pipe()
	return &m
}

type MockSSERoundTripper struct {
	t      *testing.T
	c      context.Context
	writer *io.PipeWriter
	reader *io.PipeReader
}

func (m *MockSSERoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	dump, _ := httputil.DumpRequest(req, false)
	m.t.Logf(string(dump))
	resp := new(http.Response)
	resp.Header = http.Header{
		"Content-Type": []string{"text/event-stream"},
	}

	resp.Body = m.reader

	go func() {
		m.writer.Write([]byte(fmt.Sprintf("data: ACK: %s \n\n", sdk.UUID())))

		time.Sleep(5 * time.Second)

		m.t.Logf("sending event on SSE")

		msg, _ := json.Marshal(sdk.Event{
			EventType: fmt.Sprintf("%T", sdk.EventRunWorkflowJob{}),
			Payload: map[string]interface{}{
				"ID":     float64(1),
				"Status": sdk.StatusWaiting,
			},
		})

		m.writer.Write([]byte("data: "))
		m.writer.Write(msg)
		m.writer.Write([]byte("\n\n"))

		<-m.c.Done()
		m.writer.Close()
	}()

	return resp, nil
}
