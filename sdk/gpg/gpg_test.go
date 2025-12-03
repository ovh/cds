package gpg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetKeyIdFromSignature(t *testing.T) {

	tests := []struct {
		name          string
		signature     string
		expectedError bool
		errorContains string
	}{
		{
			name:          "empty signature",
			signature:     "",
			expectedError: true,
			errorContains: "unable to decode signature",
		},
		{
			name:          "invalid armored signature",
			signature:     "invalid signature data",
			expectedError: true,
			errorContains: "unable to decode signature",
		},
		{
			name:          "malformed PGP signature",
			signature:     "-----BEGIN PGP SIGNATURE-----\ninvalid\n-----END PGP SIGNATURE-----",
			expectedError: true,
			errorContains: "unable to decode signature",
		},
		{
			name:          "not a signature block",
			signature:     "-----BEGIN PGP PUBLIC KEY BLOCK-----\ndata\n-----END PGP PUBLIC KEY BLOCK-----",
			expectedError: true,
			errorContains: "unable to decode signature",
		},
		{
			name:          "signature with no armor headers",
			signature:     "some random text without PGP markers",
			expectedError: true,
			errorContains: "unable to decode signature",
		},
		{
			name:          "signature with wrong armor type",
			signature:     "-----BEGIN PGP MESSAGE-----\nAAA=\n-----END PGP MESSAGE-----",
			expectedError: true,
			errorContains: "unable to decode signature",
		},
		{
			name:          "signature with invalid base64",
			signature:     "-----BEGIN PGP SIGNATURE-----\n@@@invalid@@@\n-----END PGP SIGNATURE-----",
			expectedError: true,
			errorContains: "unable to decode signature",
		},
		{
			name:          "truncated signature",
			signature:     "-----BEGIN PGP SIGNATURE-----\niQEz\n-----END PGP SIGNATURE-----",
			expectedError: true,
			errorContains: "unable to decode signature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyId, err := GetKeyIdFromSignature(tt.signature)

			if tt.expectedError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Empty(t, keyId)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, keyId)
				assert.Len(t, keyId, 16) // Should be 16 hex characters
				assert.Regexp(t, "^[0-9A-F]{16}$", keyId)
			}
		})
	}
}

func TestGetKeyIdFromSignature_ValidSignature(t *testing.T) {
	// This test would need a real GPG signature to work properly
	// For now, we test the structure and error handling
	t.Skip("Skipping test that requires real GPG signature data")

	// Example of how to test with real signature:
	// realSignature := "-----BEGIN PGP SIGNATURE-----\n...\n-----END PGP SIGNATURE-----"
	// keyId, err := GetKeyIdFromSignature(realSignature)
	// require.NoError(t, err)
	// assert.Regexp(t, "^[0-9A-F]{16}$", keyId)
}
