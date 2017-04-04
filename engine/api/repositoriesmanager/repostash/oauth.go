package repostash

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/go-stash/go-stash/oauth1"
	"github.com/go-stash/go-stash/stash"

	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk"
)

//StashConsumer embeds a stash oauth1 consumer
type StashConsumer struct {
	URL              string `json:"-"`
	ConsumerKey      string `json:"consumer_key"`
	PrivateRSAKey    string `json:"private_rsa_key"`
	DisableSetStatus bool   `json:"-"`
	consumer         *oauth1.Consumer
}

//New creates a new StashConsumer
func New(URL, consumerKey, privateKey string) *StashConsumer {
	s := &StashConsumer{
		URL:           URL,
		ConsumerKey:   consumerKey,
		PrivateRSAKey: privateKey,
	}
	s.consumer = &oauth1.Consumer{
		RequestTokenURL:       URL + "/plugins/servlet/oauth/request-token",
		AuthorizationURL:      URL + "/plugins/servlet/oauth/authorize",
		AccessTokenURL:        URL + "/plugins/servlet/oauth/access-token",
		CallbackURL:           oauth1.OOB,
		ConsumerKey:           consumerKey,
		ConsumerPrivateKeyPem: privateKey,
	}
	return s
}

//Data returns a serilized version of specific data
func (s *StashConsumer) Data() string {
	b, _ := json.Marshal(s)
	return string(b)
}

func (s *StashConsumer) requestToken() (*oauth1.RequestToken, error) {
	log.Info("%s\n", s.consumer)
	token, err := s.consumer.RequestToken()
	if err != nil {
		log.Warning("requestToken>%s\n", err)
		return nil, err
	}
	return token, nil
}

//AuthorizeRedirect returns the request token, the Authorize URL
func (s *StashConsumer) AuthorizeRedirect() (string, string, error) {
	requestToken, err := s.requestToken()
	if err != nil {
		log.Warning("requestToken>%s\n", err)
		return "", "", err
	}
	url, err := s.consumer.AuthorizeRedirect(requestToken)
	return requestToken.Token(), url, err
}

//AuthorizeToken returns the authorized token (and its secret)
//from the request token and the verifier got on authorize url
func (s *StashConsumer) AuthorizeToken(strToken, verifier string) (string, string, error) {
	accessTokenURL, _ := url.Parse(s.consumer.AccessTokenURL)
	req := http.Request{
		URL:        accessTokenURL,
		Method:     "POST",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Close:      true,
	}
	t := oauth1.NewAccessToken(strToken, "", map[string]string{})
	err := s.consumer.SignParams(&req, t, map[string]string{"oauth_verifier": verifier})
	if err != nil {
		return "", "", err
	}
	resp, err := http.DefaultClient.Do(&req)
	if err != nil {
		return "", "", err
	}

	accessToken, err := oauth1.ParseAccessToken(resp.Body)
	if err != nil {
		return "", "", err
	}
	return accessToken.Token(), accessToken.Secret(), nil
}

//GetAuthorized returns an authorized client
func (s *StashConsumer) GetAuthorized(accessToken, accessTokenSecret string) (sdk.RepositoriesManagerClient, error) {
	var client = stash.New(
		s.URL,
		s.ConsumerKey,
		accessToken,
		accessTokenSecret,
		s.PrivateRSAKey,
	)
	c := &StashClient{url: s.URL, client: client}
	c.disableSetStatus = s.DisableSetStatus
	return c, nil
}

//HooksSupported returns true if the driver technically support hook
func (s *StashConsumer) HooksSupported() bool {
	return true
}

//PollingSupported returns true if the driver technically support polling
func (s *StashConsumer) PollingSupported() bool {
	return false
}
