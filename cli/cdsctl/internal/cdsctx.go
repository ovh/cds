package internal

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ovh/cds/sdk"
	toml "github.com/pelletier/go-toml"
)

// CDSContext represents a CDS Context store in .cds/cdsrc file or in keychain
type CDSContext struct {
	Context               string `json:"context" cli:"context"`
	Host                  string `json:"host" cli:"host"`
	InsecureSkipVerifyTLS bool   `json:"-" cli:"-"`
	Verbose               bool   `json:"-" cli:"-"`
	SessionToken          string `json:"-" cli:"-"`
	User                  string `json:"user" cli:"user"`
}

// CDSConfigFile represents a CDS Config File
type CDSConfigFile struct {
	Current  string
	Contexts map[string]CDSContext
}

// IsKeychainEnabled returns true is keychain is enable
func IsKeychainEnabled() bool {
	return keychainEnabled
}

// GetCurrentContextName return the current contextName
func GetCurrentContextName(reader io.Reader) (string, error) {
	tomlConf, err := read(reader)
	if err != nil {
		return "", err
	}

	if tomlConf.Current == "" {
		return "", fmt.Errorf("no current context")
	}
	return tomlConf.Current, nil
}

// GetConfigFile returns the config file
func GetConfigFile(reader io.Reader) (*CDSConfigFile, error) {
	return read(reader)
}

// GetCurrentContext return the current context
func GetCurrentContext(reader io.Reader) (*CDSContext, error) {
	tomlConf, err := read(reader)
	if err != nil {
		return nil, err
	}

	if tomlConf.Current == "" {
		return nil, fmt.Errorf("no current context")
	}
	return getContext(tomlConf, tomlConf.Current)
}

// SetCurrentContext set the current context and returns it
func SetCurrentContext(reader io.Reader, writer io.Writer, contextName string) error {
	tomlConf, err := read(reader)
	if err != nil {
		return err
	}

	var found bool
	for _, c := range tomlConf.Contexts {
		if c.Context == contextName {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("context %s does not exist", contextName)
	}
	tomlConf.Current = contextName

	return writeToml(writer, tomlConf)
}

// GetContext return the a CDSContext from a contextName
func GetContext(reader io.Reader, contextName string) (*CDSContext, error) {
	tomlConf, err := read(reader)
	if err != nil {
		return nil, err
	}

	return getContext(tomlConf, contextName)
}

// StoreContext writes the .cdsrc file
func StoreContext(reader io.Reader, writer io.Writer, cdsContext CDSContext) error {
	tomlConf, err := read(reader)
	if err != nil {
		return err
	}

	var found bool
	for i := range tomlConf.Contexts {
		if tomlConf.Contexts[i].Context == cdsContext.Context {
			tomlConf.Contexts[i] = cdsContext
			found = true
			break
		}
	}
	tomlConf.Current = cdsContext.Context
	if tomlConf.Contexts == nil {
		tomlConf.Contexts = make(map[string]CDSContext, 1)
	}
	if !found {
		tomlConf.Contexts[cdsContext.Context] = cdsContext
	}

	if err := storeToken(cdsContext.Context, cdsContext.SessionToken); err != nil {
		return err
	}

	return writeToml(writer, tomlConf)
}

func writeToml(writer io.Writer, cdsConfigFile *CDSConfigFile) error {
	values := make(map[string]interface{}, len(cdsConfigFile.Contexts)+1)
	values["current"] = cdsConfigFile.Current

	for _, c := range cdsConfigFile.Contexts {
		cv := make(map[string]string, 4)
		cv["Host"] = c.Host
		if c.InsecureSkipVerifyTLS {
			cv["InsecureSkipVerifyTLS"] = strconv.FormatBool(c.InsecureSkipVerifyTLS)
		}
		if c.Verbose {
			cv["Verbose"] = strconv.FormatBool(c.Verbose)
		}
		// if keychainEnabled, we don't put the token in the file
		if !keychainEnabled {
			cv["Token"] = c.SessionToken
		}
		cv["User"] = c.User
		values[c.Context] = cv
	}

	t, err := toml.TreeFromMap(values)
	if err != nil {
		return fmt.Errorf("error while decoding file content: %v", err)
	}

	_, err = t.WriteTo(writer)
	return err
}

func read(reader io.Reader) (*CDSConfigFile, error) {
	tree, err := toml.LoadReader(reader)
	if err != nil {
		return nil, fmt.Errorf("error while decoding config file: %v", err)
	}

	m := tree.ToMap()
	tomlConf := &CDSConfigFile{}

	for i := range m {
		if strings.ToLower(i) == "current" {
			tomlConf.Current = getStringValue(m[i])
		} else {
			if tomlConf.Contexts == nil {
				tomlConf.Contexts = map[string]CDSContext{}
			}
			c, ok := tree.Get(i).(*toml.Tree)
			if !ok { // if it's not a toml.Tree in config file, ignore it
				continue
			}
			cdsContext := CDSContext{Context: i}
			for k, v := range c.ToMap() {
				switch strings.ToLower(k) {
				case "host":
					cdsContext.Host = getStringValue(v.(string))
				case "insecureskipverifytls":
					cdsContext.InsecureSkipVerifyTLS = getBoolValue(v)
				case "verbose":
					cdsContext.Verbose = getBoolValue(v)
				case "user":
					cdsContext.User = getStringValue(v.(string))
				case "sessionToken":
					cdsContext.SessionToken = getStringValue(v.(string))
				}
			}
			tomlConf.Contexts[i] = cdsContext
		}
	}
	return tomlConf, nil
}

func getBoolValue(in interface{}) bool {
	if v, ok := in.(bool); ok {
		return v
	} else if v, ok := in.(string); ok {
		return v == sdk.TrueString
	}
	return false
}

func getStringValue(in interface{}) string {
	if v, ok := in.(string); ok {
		return v
	}
	return ""
}

// getContext read cdsrc file and return a CDSContext matching the contextName
// if token is in keychain, get the token from it
func getContext(tomlConf *CDSConfigFile, contextName string) (*CDSContext, error) {
	var cdsContext CDSContext
	for _, c := range tomlConf.Contexts {
		if c.Context == contextName {
			cdsContext = c
			break
		}
	}

	if keychainEnabled {
		token, err := cdsContext.getToken(contextName)
		if err != nil {
			return nil, err
		}
		cdsContext.SessionToken = token
	}
	return &cdsContext, nil
}
