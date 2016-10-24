package secret

//Status return status for secret backend
func Status() string {
	if Client == nil {
		return "Secret Backend not initialized"
	}
	return Client.Name()
}
