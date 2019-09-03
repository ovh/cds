// +build nokeychain freebsd openbsd 386

package internal

var keychainEnabled = false

//storeTokens store tokens into keychain
func storeTokens(contextName string, tokens ContextTokens) error {
	//nothing to do here, token is already in cdsrc file
	return nil
}

//getContextTokens rerieves a CDS Context from CDSContext
func (c CDSContext) getContextTokens(contextName string) (*ContextTokens, error) {
	return &ContextTokens{Session: c.Session, Token: c.Token}, nil
}
