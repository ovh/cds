package sdk_test

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/ovh/cds/sdk"

	"github.com/stretchr/testify/require"
)

func TestBlur(t *testing.T) {
	b, err := sdk.NewBlur([]string{
		"1234567890",
		"1234567890abcdef",
		"&é'(§è!çà",
		`"1234567890`,
		"12345",
		"123456",
	})
	require.NoError(t, err)

	require.Equal(t, "12345", b.String("12345"), "Secret size < secret min length")
	require.Equal(t, sdk.PasswordPlaceholder, b.String("1234567890abcdef"))
	require.Equal(t, sdk.PasswordPlaceholder, b.String("&é'(§è!çà"))
	require.Equal(t, sdk.PasswordPlaceholder, b.String(`&é'"'"'(§è!çà`))
	require.Equal(t, sdk.PasswordPlaceholder, b.String(url.QueryEscape("&é'(§è!çà")))
	require.Equal(t, sdk.PasswordPlaceholder, b.String(base64.StdEncoding.EncodeToString([]byte("&é'(§è!çà"))))

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
