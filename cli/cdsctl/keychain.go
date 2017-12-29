// +build !darwin !windows !linux

package main

func storeSecret(configFile, url, username, token string) error {
	return nil
}

func loadSecret(configFile, url string) (username, token string, err error) {
	return "", "", nil
}
