package worker

//Initialize init the package
func Initialize() error {
	go Heartbeat()
	go ModelCapabilititiesCacheLoader(5)
	return nil
}
