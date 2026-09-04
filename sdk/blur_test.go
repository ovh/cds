package sdk_test

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"html"
	"net/url"
	"strings"
	"testing"

	"github.com/ovh/cds/sdk"

	"github.com/stretchr/testify/require"
)

func TestBlur(t *testing.T) {
	jsonScret := `{ "varjsoncomplex": { "foo": { "bar": { "aa": "**********", "fb": "**********", "fb2": [ "111", "fds" ], "fb3": [ { "tt": "aze" } ], "fb4": [ 1, 2, 3 ] } } } }`
	var mJson map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(strings.Replace(jsonScret, " ", "", -1)), &mJson))
	bts, err := json.Marshal(mJson)
	require.NoError(t, err)

	b, err := sdk.NewBlur([]string{
		"1234567890",
		"1234567890abcdef",
		"&é'(§è!çà",
		`"1234567890`,
		"12345",
		"123456",
		"123\n456",
		string(bts),
	})
	require.NoError(t, err)

	require.Equal(t, "12345", b.String("12345"), "Secret size < secret min length")
	require.Equal(t, sdk.PasswordPlaceholder, b.String("1234567890abcdef"))
	require.Equal(t, sdk.PasswordPlaceholder, b.String("&é'(§è!çà"))
	require.Equal(t, sdk.PasswordPlaceholder, b.String(`&é'"'"'(§è!çà`))
	require.Equal(t, sdk.PasswordPlaceholder, b.String(url.QueryEscape("&é'(§è!çà")))
	require.Equal(t, sdk.PasswordPlaceholder, b.String(base64.StdEncoding.EncodeToString([]byte("&é'(§è!çà"))))
	require.Equal(t, sdk.PasswordPlaceholder, b.String("123\\n456"))
	require.Equal(t, sdk.PasswordPlaceholder, b.String(jsonScret))

	buf, err := json.Marshal(`"1234567890`)
	require.NoError(t, err)
	require.Equal(t, "\""+sdk.PasswordPlaceholder+"\"", b.String(string(buf)))

	type report struct {
		String      string   `json:"string,omitempty"`
		StringSlice []string `json:"string_slice,omitempty"`
		Number      int      `json:"number,omitempty"`
	}
	r := report{
		String:      "&é'(§è!çà",
		StringSlice: []string{"1234567890", "&é'(§è!çà"},
	}
	require.NoError(t, b.Interface(&r))
	require.Equal(t, sdk.PasswordPlaceholder, r.String)
	require.Equal(t, sdk.PasswordPlaceholder, r.StringSlice[0])
	require.Equal(t, sdk.PasswordPlaceholder, r.StringSlice[1])

	expected, err := json.Marshal(report{
		String:      sdk.PasswordPlaceholder,
		StringSlice: []string{sdk.PasswordPlaceholder, sdk.PasswordPlaceholder},
	})
	require.NoError(t, err)
	source, err := json.Marshal(report{
		String:      "&é'(§è!çà",
		StringSlice: []string{`"1234567890`, "&é'(§è!çà"},
	})
	require.NoError(t, err)
	require.Equal(t, string(expected), b.String(string(source)))

	tests := sdk.JUnitTestsSuites{
		TestSuites: []sdk.JUnitTestSuite{{
			Total:   123456,
			Skipped: 5,
			TestCases: []sdk.JUnitTestCase{{
				Systemout: sdk.JUnitInnerResult{
					Value: "1234567890abcdef",
				},
			}},
		}},
	}
	require.NoError(t, b.Interface(&tests))
	require.Equal(t, sdk.PasswordPlaceholder, tests.TestSuites[0].TestCases[0].Systemout.Value)
	require.Equal(t, 0, tests.TestSuites[0].Total)
	require.Equal(t, 5, tests.TestSuites[0].Skipped)
}

func TestBlurBase64(t *testing.T) {
	// One secret for each length modulo 3 as the base64 padding depends on the input length
	secrets := []string{"the-api-key-aa", "the-api-key-abc", "the-api-key-ab"}

	for _, secret := range secrets {
		b, err := sdk.NewBlur([]string{secret})
		require.NoError(t, err)

		for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
			// Raw secret, as done by `printf '%s' "$secret" | base64 -w0`
			require.Equal(t, sdk.PasswordPlaceholder, b.String(enc.EncodeToString([]byte(secret))))
			// Secret with a trailing end of line, as done by `echo "$secret" | base64 -w0`
			require.Equal(t, sdk.PasswordPlaceholder, b.String(enc.EncodeToString([]byte(secret+"\n"))))
			require.Equal(t, sdk.PasswordPlaceholder, b.String(enc.EncodeToString([]byte(secret+"\r\n"))))
		}

		// Encoded secret in the middle of a log line
		require.Equal(t, "API KEY: "+sdk.PasswordPlaceholder+", but... "+sdk.PasswordPlaceholder,
			b.String("API KEY: "+secret+", but... "+base64.StdEncoding.EncodeToString([]byte(secret+"\n"))))
	}

	// A long secret encoded without -w0 is wrapped every 76 chars by the base64 tool, logs are
	// blurred line by line so every wrapped line must be blurred on its own
	longSecret := strings.Repeat("abcdefghij", 20)
	b, err := sdk.NewBlur([]string{longSecret})
	require.NoError(t, err)
	encoded := base64.StdEncoding.EncodeToString([]byte(longSecret + "\n"))
	for i := 0; i < len(encoded); i += 76 {
		require.Equal(t, sdk.PasswordPlaceholder, b.String(encoded[i:min(i+76, len(encoded))]))
	}
}

func TestBlurSecretEncodedInsideAnotherValue(t *testing.T) {
	// `base64("user:$password")`, the shape of a registry credential: the encoding of the password
	// alone is nowhere in that string, only a part of it survives.
	secret := "th3-r3g1stry-p4ssw0rd"
	b, err := sdk.NewBlur([]string{secret})
	require.NoError(t, err)

	// Every prefix length, so the secret starts on each of the three byte alignments, and a
	// suffix, which changes the last chars of the encoding
	for _, user := range []string{"", "a", "ab", "abc", "user", "user:", "a-longer-user:"} {
		for _, suffix := range []string{"", "\n", ":extra"} {
			blob := base64.StdEncoding.EncodeToString([]byte(user + secret + suffix))
			blurred := b.String(blob)
			require.Contains(t, blurred, sdk.PasswordPlaceholder,
				"secret encoded with prefix %q and suffix %q not blurred: %s", user, suffix, blurred)
			// What stays readable are the chars of the groups of three bytes the secret shares
			// with the prefix and the suffix, so at most two of its bytes on each side. The rest
			// of the blob, everything the secret encodes on its own, has to be gone.
			replaced := len(blob) - len(strings.Replace(blurred, sdk.PasswordPlaceholder, "", 1))
			require.GreaterOrEqual(t, replaced, len(secret),
				"too little blurred for prefix %q and suffix %q: %s", user, suffix, blurred)
		}
	}
}

func TestBlurEncodings(t *testing.T) {
	secret := "the api key &é'(§è!çà"
	b, err := sdk.NewBlur([]string{secret})
	require.NoError(t, err)

	require.Equal(t, sdk.PasswordPlaceholder, b.String(url.QueryEscape(secret)))
	require.Equal(t, sdk.PasswordPlaceholder, b.String(url.PathEscape(secret)))
	require.Equal(t, sdk.PasswordPlaceholder, b.String(html.EscapeString(secret)))
	require.Equal(t, sdk.PasswordPlaceholder, b.String(hex.EncodeToString([]byte(secret))))
	require.Equal(t, sdk.PasswordPlaceholder, b.String(strings.ToUpper(hex.EncodeToString([]byte(secret)))))
	// `echo "$secret" | xxd -p` encodes the end of line added by echo, only that byte remains
	require.Equal(t, sdk.PasswordPlaceholder+"0a", b.String(hex.EncodeToString([]byte(secret+"\n"))))
}

func TestBlurMarkupEscapes(t *testing.T) {
	secret := `p@ss w0rd&<sh3ll>'quoted'`
	b, err := sdk.NewBlur([]string{secret})
	require.NoError(t, err)

	// Go escapes the quotes as numeric references, a `sed` in a step script usually escapes only
	// the three structural chars, and an XML escaper uses the named references
	require.Equal(t, sdk.PasswordPlaceholder, b.String(html.EscapeString(secret)))
	require.Equal(t, sdk.PasswordPlaceholder, b.String(
		strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(secret)))
	require.Equal(t, sdk.PasswordPlaceholder, b.String(
		strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;").Replace(secret)))
}

func TestBlurSecretWithSurroundingBlankChars(t *testing.T) {
	b, err := sdk.NewBlur([]string{"  the-api-key\n"})
	require.NoError(t, err)

	require.Equal(t, sdk.PasswordPlaceholder, b.String("the-api-key"))
	require.Equal(t, sdk.PasswordPlaceholder, b.String(base64.StdEncoding.EncodeToString([]byte("the-api-key"))))
}

func TestBlurMultiLineSecret(t *testing.T) {
	// The shape of a key or of a credential file, without the header of one: a value looking like
	// a private key in the source of a test is what secret scanners are there to shout about
	secret := "AAAAAmultilinesecretAAAAA\nBBBBBmultilinesecretBBBBB\nCCCCCmultilinesecretCCCCC\n"
	b, err := sdk.NewBlur([]string{secret})
	require.NoError(t, err)

	// The worker blurs the logs line by line, the whole secret is never in a single log line
	for _, line := range strings.Split(strings.TrimSpace(secret), "\n") {
		require.Equal(t, sdk.PasswordPlaceholder, b.String(line))
	}
	require.Equal(t, sdk.PasswordPlaceholder, b.String(secret))

	// A multi-line secret given to a step through `env` reaches it as a single line, whatever else
	// it holds: the worker exports it through OneLineValue
	quoted := "line one \"quoted\"\nline two 'quoted'\nline three\n"
	b, err = sdk.NewBlur([]string{quoted})
	require.NoError(t, err)
	require.Equal(t, sdk.PasswordPlaceholder, b.String(sdk.OneLineValue(quoted)))
}
