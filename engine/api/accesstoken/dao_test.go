package accesstoken

import (
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

// This is a test key, do not use it in real life
var TestKey = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAj3YCi33CaIiWfhsYz3lxOGjDSxxtA+LM4dDjIFe3Xq+gntcg
1WKFoAsnHFgC3sOoZKSeIjuBIsGXvfOzOs10EdlU388bAOP51NmsGLtVwBSpYQkQ
FGb1QricuZy6BZB0JiBM9raz5ikszG3m52opS3pibw19ZyvUSSjHAiXEaJpML0m/
YiKowrf2bO2cFbSATCDEhK5pDhzllRhLOkST/VH3QSrKL0xydKNjGmmJDlpM2xKT
7Vbb2DkMPl4kVnYf/XveojS0GSbsQaIS17WEMayP4ch9g27E5GMp0+IZ7w9Dq/ai
7T+hMqlkFfajB97zTqHFRD4hMITckjpPlPx8WwIDAQABAoIBABpC8xJP8i+qmUn6
cd9BDu3Rl7Z/PKGSegj4cStxgzrNEa0iGnuVbnqur/krT1MyI/hQfjYsCGaxY7K9
Etk31QCTdUsHIZ1XHlvNgQiB+p+P6LW/r/bcJheRrfb4bsEoAWsdTJl5NpNyhCXk
FHnWYDrV64ECyisBxfmiglOtUDgJht1IKgIp9vULWJPQ7/PYRc7R7kpfiSGTPgmP
LV/20edWfBsbxPR/2rL5azpL3YIkJgNDRrnieDHzuOJ86FzICWq8gLhta9j5FARG
0PSs0Myy9ucfAu+lVi4S5/GsyfEiljXznGyQxFwR9EZp+BZEBvdFtkIod48d0DQy
t7xmrEECgYEA8KU+9O1pC/B8/61ZLLMEgq7EQ1qUDZ5cNSIJoTf9vvCQD+hCbtwY
Wgq+MIYR0dNn8MxAmwsZeAFfu9USJNiDKzc7yYSQ4OJXHXk33895UllsZdpaj5cc
hGkxnr8JMdWLIsmeCF8F9mIQywV+QLmjPQVW8VBYFY4+0dbrfmtIYSECgYEAmJ1U
6klHtEWv+Msc8Yjg/d5oPQuBy9ilRv97g5ilaHQ4aMDvsiV1HCxER0NA85jjCP+/
ulYwoLWgV+WObbEGeg+B929oHRSFp/XTvEWhoOxAAICMrVwQ6qX5yPOAKtPKmkop
m6PbzM+QrIRw0cYXEZEVG3Cme8x+sHKQ54CAIfsCgYBGzig3Ar+8zpbI1+V8HHRA
S1HeC4GyfBzfWVOCByp3CusocwtQ+RuFKtIJDvmhRlW36TE9LUfiIm1bo/bBtp7p
kUfbJFFIifBd8LO6+53T2BHn6hZpV2oBn74E2mrHKfDVXINOLT9g3jvYsJYUT0qz
gqWxPRWdygu7zEPgH4rdYQKBgBzvDy9P71k9MQyhLX6ZbdaTuP2B1fzYuRUJ0Nf1
M77m8d7iXU9QDLDnr5Y3KPRGEx0cp7PjLVr6tEiVy/f97PVtRT2tEHca8fATCi6S
oP8Ka2Ps+z7OyqJCD2ZKzAzSlIHF97d7TGu7Gnmqrl0HCk6ZTAAkzluAPLClN9W8
Jg7LAoGAFxXOBXuGB+Lsbgioka0vM1mGYWEKjobPcQRkMq37b6GdkhMl2A5fH4C+
uhOrSSJ8cK0UO9ET6DV6V5MuQoEAMVYt8v39fxOnrH7sX2OwTqXOqK7b27vfcY+g
G6f1bOI7lNhA4uAqZICcXO8cxwEa8xoeuPFT2I0R8tzAD5GhIto=
-----END RSA PRIVATE KEY-----`)

func TestInsertForUser(t *testing.T) {
	Init("cds_test", []byte("secretkey"))
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	usr1, _ := assets.InsertLambdaUser(db)
	grp1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	exp := time.Now().Add(5 * time.Minute)
	token, _, err := New(*usr1, []sdk.Group{*grp1}, "cds_test", "cds test", &exp)
	test.NoError(t, err)

	test.NoError(t, Insert(db, &token))
}

func TestUpdate(t *testing.T) {

}
