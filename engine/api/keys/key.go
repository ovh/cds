package keys

import "github.com/ovh/cds/sdk"

func GenerateKey(name string, t sdk.KeyType) (sdk.Key, error) {
	switch t {
	case sdk.KeyTypeSSH:
		return GenerateSSHKey(name)
	case sdk.KeyTypePGP:
		return GeneratePGPKeyPair(name)
	default:
		return sdk.Key{}, sdk.WrapError(sdk.ErrUnknownKeyType, "unknown key of type: %s", t)
	}
}
