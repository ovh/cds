package internal

import (
	"io"
	"strconv"
	"strings"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	toml "github.com/pelletier/go-toml"
)

// CDSContext represents a CDS Context store in .cds/cdsrc file or in keychain
type CDSContext struct {
	Context               string `json:"context" cli:"context"`
	Host                  string `json:"host" cli:"host"`
	InsecureSkipVerifyTLS bool   `json:"-" cli:"-"`
	Verbose               bool   `json:"-" cli:"-"`
	Session               string `json:"-" cli:"-"` // Session Token
	Token                 string `json:"-" cli:"-"` // BuiltinConsumerAuthenticationToken
}

// ContextTokens contains a Session Token and a Sign in token
// this struct is use to store secret in keychain
type ContextTokens struct {
	Session string `json:"session" cli:"-"` // Session Token
	Token   string `json:"token" cli:"-"`   // BuiltinConsumerAuthenticationToken
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
		return "", cli.NewError("no current context")
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
		return nil, cli.NewError("no current context")
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
		return cli.NewError("context %s does not exist", contextName)
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
func StoreContext(reader io.Reader, writer io.Writer, cdsctx CDSContext) error {
	tomlConf, err := read(reader)
	if err != nil {
		return err
	}

	var found bool
	for i := range tomlConf.Contexts {
		if tomlConf.Contexts[i].Context == cdsctx.Context {
			tomlConf.Contexts[i] = cdsctx
			found = true
			break
		}
	}
	tomlConf.Current = cdsctx.Context
	if tomlConf.Contexts == nil {
		tomlConf.Contexts = make(map[string]CDSContext, 1)
	}
	if !found {
		tomlConf.Contexts[cdsctx.Context] = cdsctx
	}

	tokens := ContextTokens{Session: cdsctx.Session, Token: cdsctx.Token}
	if err := storeTokens(cdsctx.Context, tokens); err != nil {
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
			cv["Session"] = c.Session
			cv["Token"] = c.Token
		}
		values[c.Context] = cv
	}

	t, err := toml.TreeFromMap(values)
	if err != nil {
		return cli.WrapError(err, "error while decoding file content")
	}

	_, err = t.WriteTo(writer)
	return err
}

func read(reader io.Reader) (*CDSConfigFile, error) {
	tree, err := toml.LoadReader(reader)
	if err != nil {
		return nil, cli.WrapError(err, "error while decoding config file")
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
				case "session":
					cdsContext.Session = getStringValue(v.(string))
				case "token":
					cdsContext.Token = getStringValue(v.(string))
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
		tokens, err := cdsContext.getTokens(contextName)
		if err != nil {
			return nil, err
		}
		cdsContext.Session = tokens.Session
		cdsContext.Token = tokens.Token
	}
	return &cdsContext, nil
}
