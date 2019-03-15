package localauthentication

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"

	"github.com/ovh/cds/sdk"
)

type userLocalAuthentication struct {
	sdk.UserLocalAuthentication
	EncryptedPassword []byte `db:"encrypted_password" json:"-"`
}

func init() {
	gorpmapping.Register(gorpmapping.New(userLocalAuthentication{}, "user_local_authentication", false, "user_id"))
}
