// +build nokeychain

package internal

var keychainEnabled = false

//storeToken store a context into keychain
func storeToken(contextName, token string) error {
	//nothing to do here, token is already in cdsrc file
	return nil
}

//getToken rerieves a CDS Context from CDSContext
func (c CDSContext) getToken(contextName string) (string, error) {
	return c.Token, nil
}
