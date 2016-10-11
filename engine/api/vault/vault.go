package vault

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ovh/cds/engine/log"
)

const ()

var (
	//ErrLogin failed to login to vault
	ErrLogin = errors.New("Failed to login to vault")
	//ErrInternalServerError other errors
	ErrInternalServerError = errors.New("Internal server error")
	//tokenHeader
	tokenHeader string
)

func doReq(tk, method, host, path string, body map[string]interface{}, ret interface{}) error {
	var bodyBuf *bytes.Buffer
	if body != nil {
		j, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyBuf = bytes.NewBuffer(j)
	} else {
		bodyBuf = bytes.NewBuffer([]byte{})
	}
	r, err := http.NewRequest(method, host+path, bodyBuf)
	if err != nil {
		return err
	}
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}

	if tk != "" {
		r.Header.Set(tokenHeader, tk)
	}

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	respbod, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Warning("Vault.doReq> response status code : %d\n", resp.StatusCode)
		if resp.StatusCode == 401 {
			log.Warning("Vault.doReq> response : %s\n", string(respbod))
			return ErrLogin
		}
		if len(respbod) == 0 {
			log.Warning("Vault.doReq> response response is empty\n")
			return ErrInternalServerError
		}
		e := struct {
			Error string `json:"error"`
		}{}
		err := json.Unmarshal(respbod, &e)
		if err != nil {
			return errors.New(string(respbod))
		}
		return errors.New(e.Error)
	}
	if len(respbod) > 0 {
		err = json.Unmarshal(respbod, &ret)
		if err != nil {
			return err
		}
	}
	return nil
}

func authenticate(apiURL, ak, potp string) (string, error) {
	ret := struct {
		Token string `json:"token"`
	}{}

	err := doReq("", "POST", apiURL, "/auth/app", map[string]interface{}{"app_key": ak, "platform_otp": potp}, &ret)
	if err != nil {
		//FIX ME avoid print it
		log.Warning("Vault.doReq> authentication error : %s\n", err)
	}
	if ret.Token == "" {
		log.Warning("Vault.doReq> authentication error: token is empty\n")
	}
	return ret.Token, err
}

func getAppNamespace(apiURL, tk string) (string, error) {

	ret := struct {
		App *struct {
			Namespace string `json:"namespace"`
		} `json:"app"`
	}{}

	err := doReq(tk, "GET", apiURL, "/me", nil, &ret)
	if err != nil {
		return "", err
	}

	if ret.App == nil {
		log.Warning("vault.getAppNamespace> Unable to get app namespace %s", tk)
		return "", ErrLogin
	}

	return ret.App.Namespace, nil
}

func getSecrets(apiURL, tk, ns string) (map[string]string, error) {

	ret := []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}{}

	err := doReq(tk, "GET", apiURL, fmt.Sprintf("/namespace/%s/secret", ns), nil, &ret)
	if err != nil {
		Status = StatusKO
		return nil, err
	}

	secrets := make(map[string]string)

	if len(ret) == 0 {
		log.Warning("vault.getSecrets> no secrets found with %s %s", ns, tk)
	}

	for _, s := range ret {
		secrets[s.Key] = s.Value
	}

	Status = StatusOK
	return secrets, nil
}

//GetSecrets returns all the secrets as a key/value map for the application namespace
func GetSecrets(apiURL, ak, potp string) (map[string]string, error) {

	tk, err := authenticate(apiURL, ak, potp)
	if err != nil {
		Status = StatusKO
		return nil, err
	}

	ns, err := getAppNamespace(apiURL, tk)
	if err != nil {
		Status = StatusKO
		return nil, err
	}

	return getSecrets(apiURL, tk, ns)
}

//GetSecret returns the key, the value
func GetSecret(apiURL, ak, potp, secretKey string) (string, string, error) {
	tk, err := authenticate(apiURL, ak, potp)
	if err != nil {
		Status = StatusKO
		return "", "", err
	}

	ns, err := getAppNamespace(apiURL, tk)
	if err != nil {
		Status = StatusKO
		return "", "", err
	}

	secrets, err := getSecrets(apiURL, tk, ns)
	if err != nil {
		Status = StatusKO
		return "", "", err
	}
	if s := secrets[secretKey]; s != "" {
		Status = StatusOK
		return secretKey, s, nil
	}
	log.Warning("vault.GetSecret> Unable to find %s\n", secretKey)
	Status = StatusKO
	return "", "", err
}
