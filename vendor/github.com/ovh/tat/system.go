package tat

type Capabilities struct {
	UsernameFromEmail bool              `json:"username_from_email"`
	Hooks             []CapabilitieHook `json:"hooks"`
}

type CapabilitieHook struct {
	HookType    string `json:"type"`
	HookEnabled bool   `json:"enabled"`
}

// SystemCacheClean clean cache, only for tat admin
func (c *Client) SystemCacheClean() ([]byte, error) {
	return c.simpleGetAndGetBytes("/system/cache/clean")
}

// SystemCacheInfo returns cache information
func (c *Client) SystemCacheInfo() ([]byte, error) {
	return c.simpleGetAndGetBytes("/system/cache/info")
}
