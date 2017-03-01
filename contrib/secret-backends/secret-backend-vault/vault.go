package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ovh/cds/engine/api/secret/secretbackend"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/vault/api"
)

var (
	vaultAPI   string
	vaultToken string
	vaultNS    string
)

//Vault is the type implementing secretbackend.Driver interface
type Vault struct{}

//Init initialize the plugin
func (v *Vault) Init(opts secretbackend.MapVar) error {
	//Get value in options
	vaultAPI = opts.Get("vault_addr")
	if vaultAPI == "" {
		//Or the defauult env variable
		vaultAPI = os.Getenv("VAULT_ADDR")
	}
	if vaultAPI == "" {
		//Or the default dev server
		vaultAPI = "http://127.0.0.1:8200"
	}
	//Get value in options
	vaultToken = opts.Get("vault_token")
	if vaultToken == "" {
		//Else get default env variable
		vaultToken = os.Getenv("VAULT_TOKEN")
	}

	vaultNS = opts.Get("vault_namespace")
	if vaultNS == "" {
		return errors.New("Vault namespace is mandatory")
	}

	log.Printf("Plugin initialized : %s\n", vaultNS)

	return nil
}

//Name returns the plugin name
func (v *Vault) Name() string {
	return fmt.Sprintf("Vault [%s]", vaultAPI)
}

//GetSecrets returns all the secrets
func (v *Vault) GetSecrets() secretbackend.Secrets {
	//returned value
	result := secretbackend.NewSecrets(map[string]string{})

	//Set config
	config := &api.Config{
		Address:    vaultAPI,
		HttpClient: cleanhttp.DefaultClient(),
		MaxRetries: 3,
	}

	//Set http client behavior
	config.HttpClient.Timeout = time.Second * 60
	transport := config.HttpClient.Transport.(*http.Transport)
	transport.TLSHandshakeTimeout = 10 * time.Second
	transport.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	client, err := api.NewClient(config)
	if err != nil {
		log.Printf("Cannot connect on Vault :%s\n", err)
		result.Error = secretbackend.Error(err)
		return *result
	}
	log.Printf("Connected on Vault %s\n", vaultAPI)

	client.SetToken(vaultToken)
	c := client.Logical()

	log.Printf("Loading secret from path : %s\n", vaultNS)

	s, err := c.List(vaultNS)
	if err != nil {
		log.Printf("Cannot list secret :%s\n", err)
		result.Error = secretbackend.Error(err)
		return *result
	}
	if s == nil {
		result.Error = secretbackend.Error(fmt.Errorf("No value found at %s on %s\n", vaultNS, vaultAPI))
		return *result
	}

	res := map[string]string{}
	nst := strings.Split(vaultNS, "/")
	prefix := nst[len(nst)-1]

	keys, ok := s.Data["keys"].([]interface{})
	if !ok {
		result.Error = secretbackend.Error(fmt.Errorf("Unsupported data type %T", s.Data["keys"]))
		return *result
	}

	for i := range keys {
		s := fmt.Sprintf("%v", keys[i])
		k, err := c.Read(vaultNS + "/" + s)
		if err != nil {
			log.Println(err)
		}

		for k, v := range k.Data {
			if k == s {
				res[prefix+"/"+s] = fmt.Sprintf("%v", v)
			}
		}
	}

	result.Data = res
	return *result
}

func main() {
	p := Vault{}
	secretbackend.Serve(os.Args[0], &p)
}
