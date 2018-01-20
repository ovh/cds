package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// GeneratePassword return clear password (to user only), salt and hashed password
// salt and hashed password are stored in db
func TestGeneratePassword(t *testing.T) {
	password, err := generatePassword()
	assert.Nil(t, err, "should be nil")
	salt, err := GenerateSalt()
	assert.Nil(t, err, "should be nil")
	passwordHashed := HashPassword(password, salt)
	passwordHashed2 := HashPassword(password, salt)
	assert.Equal(t, passwordHashed, passwordHashed2, "should be same")
}
