package corpsso

import (
	"bytes"
	"encoding/base64"
	"testing"
	"time"

	jose "gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"

	"github.com/ovh/cds/sdk/gpg"
)

const (
	MasterKey = `-----BEGIN PGP PUBLIC KEY BLOCK-----

mDMEXF1XRhYJKwYBBAHaRw8BAQdABEHVkfddwOIEFd7V0hsGrudgRuOlnV4/VSK6
YJGFag+0HnRlc3QtbG9ja2VyIDx0ZXN0QGxvbGNhdC5ob3N0PoiQBBMWCAA4FiEE
BN0dlUe5Vi8hx0ZsWXCoyV8Z2eQFAlxdV0YCGwMFCwkIBwIGFQoJCAsCBBYCAwEC
HgECF4AACgkQWXCoyV8Z2eQt5gEAycwThBk4CzuQ8XtPvLA/kml3Jkclgw6ACGsP
YOrnz+gA/2XOjnhYOA6S3sn9g4UMVtON8TofBMTTSqCdgrghu3kFuDgEXF1XRhIK
KwYBBAGXVQEFAQEHQGlq7X9fCeXKxlmcWgT+fFJyS1MlL2uwKQteXl8yIadwAwEI
B4h4BBgWCAAgFiEEBN0dlUe5Vi8hx0ZsWXCoyV8Z2eQFAlxdV0YCGwwACgkQWXCo
yV8Z2eR4rgD/cPn9TStAoXc4Pa+sKgAFmG3NVCNln8FtkH5cQ1g0ouUA/AzcLTL4
VQHT6ArvDWzJKKrh2PepZ5PVMS/Hwh/GDH4J
=n1Ws
-----END PGP PUBLIC KEY BLOCK-----`

	AuthKey = `-----BEGIN PGP PRIVATE KEY BLOCK-----

lFgEXGE6vBYJKwYBBAHaRw8BAQdAWHzsCcqZgcWlcYQdgC+8ZjtBhyuNpiuECnUM
Ya98JlEAAP9LldoVz0MGzqpcy+1B4IzVaEH8rTjlXYqvv8mjWAliORIgtCF0ZXN0
LWxvY2tlci0yIDx0ZXN0MkBsb2xjYXQuaG9zdD6IkAQTFggAOBYhBMViilPFEBsK
55iNjrxDRZAQNUl5BQJcYTq8AhsDBQsJCAcCBhUKCQgLAgQWAgMBAh4BAheAAAoJ
ELxDRZAQNUl5BjEA/26x13zHoICnflMQItCsxM4Ab07li0keyilfwyZ6nm73AQD3
xAuyEnt1hTo1srOafYun/LXNNGxoVsJIfMk7eVn4Boh1BBAWCAAdFiEEBN0dlUe5
Vi8hx0ZsWXCoyV8Z2eQFAlxhTRoACgkQWXCoyV8Z2eTVPwEA5n727+N1bDAm4jJY
HEWL9QOA7MH2+tZKhLXpgdiZ/oMA/25j8nvrdVkxrbQV9ODLomC2Q5whsq5ucj4/
SQUuBb4IiQIzBBABCAAdFiEExDJ6aWjwDDMFny4YxVpSg4XsU8wFAlxhZVcACgkQ
xVpSg4XsU8yJ/xAAt5/X+0AQc0a3z2CB+65gRgV4Fnb35cV59uQFNGEgSRqoVv0s
z1ehIneVtHKDv07eZ143BPjQSl1G9nqZs/4TLOOxfFLpAngRdUoZa2Y8z8YHx/gD
LBwSwrlnPR2/WjX/a/Spukv8hgWNCI/mUs2AOUzPkiyyzBubNRH1JGsONNE7nt4w
PDw6gPlX6DvTDBsxqZienG1EOatYy6NDK3b06ShCc/lYMaQi4yg6AbkTvh9IHPE1
RbBf6hPZVEI7Dwni1RLHs+6fqLrLRUHNdcOg2OUvuaZ9zFU8DWhV3M5H2+1w+0Tv
fI1tThRdgZNCwlveGkoApOK435G95bovFaWo78C+LwuKj6+s9SU/Wt5X6ycGWCJj
VGffpqFjk/GZN6jW8Bp/+4GhYymU+GSC6lkzbA07MbGnQAwd29/QgxaQbjOMEwza
DOYm6vXR8LiNNVOFnZ6ChhzwNxpaHb45zJvZ95FW546JmuCT70AyLSjArQQnpzTa
t+p1xwWlYN+LCYedPZ4JiUioSl9Cz6D/Z9rBhKBPDPVrqHE90t78/91AbQ8el/DB
uOW6mhUAyM2JHdu8oRFKp1PcOBN1FZ4nRK+SOsSqVGLGCQGTUjT4HXrZJRzwGTIV
M7FuMsden1WuaDw5hA7R+0F0P7iDiyhxDVmndntzVgWEERV/SSniuTqCD7acXQRc
YTq8EgorBgEEAZdVAQUBAQdAU1MwdRy9nzPQqWumOn8yW6hn1D3/NoNKLipzLVdp
SjADAQgHAAD/cL/rPYub3SuwSOhmYIr+SySWyM7xz8Eg9YMElj1nsQASVYh4BBgW
CAAgFiEExWKKU8UQGwrnmI2OvENFkBA1SXkFAlxhOrwCGwwACgkQvENFkBA1SXmJ
/gD/SBH6/tULYlpxqD0dO7D4wmHzWKPRtBIPHDWHtWKFAXoA/AjbE7M6aZBHWFAL
dg/94O8U5bC2T8a9CsA/q8eGuucP
=fl70
-----END PGP PRIVATE KEY BLOCK-----`
)

func TestIsSignedBy(t *testing.T) {
	// Key signin key
	MasterKey, err := gpg.NewPublicKeyFromPem(MasterKey)
	if err != nil {
		t.Errorf("unable to parse key: %v", err)
	}

	t.Log("master key: ", MasterKey.KeyFingerprint())

	privKey, err := gpg.NewPrivateKeyFromPem(AuthKey, "")
	if err != nil {
		t.Errorf("unable to parse key: %v", err)
	}

	t.Log("auth key: ", privKey.KeyId())

	if privKey.IsSignedBy(MasterKey) == nil {
		t.Error("key is not signed by master key")
	}
}

func TestGPGLib(t *testing.T) {
	// Key signin key
	MasterKey, err := gpg.NewPublicKeyFromPem(MasterKey)
	if err != nil {
		t.Errorf("unable to parse key: %v", err)
	}

	t.Log("master key: ", MasterKey.KeyFingerprint())

	privKey, err := gpg.NewPrivateKeyFromPem(AuthKey, "")
	if err != nil {
		t.Errorf("unable to parse key: %v", err)
	}

	t.Log("auth key: ", privKey.KeyId())

	if privKey.IsSignedBy(MasterKey) == nil {
		t.Error("key is not signed by master key")
	}
	data, _ := privKey.Serialize()

	sig, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.EdDSA, Key: jose.JSONWebKey{Key: privKey.GetKey(), KeyID: privKey.KeyId()}},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("key", data))
	if err != nil {
		t.Errorf("unable to create signer: %v", err)
	}

	cl := jwt.Claims{
		IssuedAt: jwt.NewNumericDate(time.Now()),
	}

	var token = struct{ Foo string }{Foo: "bar"}

	raw, err := jwt.Signed(sig).Claims(cl).Claims(token).CompactSerialize()
	if err != nil {
		t.Errorf("unable to sign jwt: %v", err)
	}

	t.Log("raw token", raw)

	jws, err := jose.ParseSigned(raw)
	if err != nil {
		t.Errorf("unable to parse jwt: %v", err)
	}

	rawSsoKeyContent := jws.Signatures[0].Protected.ExtraHeaders[jose.HeaderKey("key")].(string)

	t.Log("raw key", rawSsoKeyContent)

	rawSsoKey, err := base64.StdEncoding.DecodeString(rawSsoKeyContent)
	if err != nil {
		t.Errorf("unable to decode rawSsoKey: %v", err)
	}

	t.Log("sso key", string(rawSsoKey))

	ssoKey, err := gpg.NewPublicKeyFromData(bytes.NewReader(rawSsoKey))
	if err != nil {
		t.Errorf("unable to parse public key: %v", err)
	}

	ssoKeySig := ssoKey.IsSignedBy(MasterKey)
	if ssoKeySig == nil {
		t.Errorf("ssoKey is not signed by KSK")
	}

}
