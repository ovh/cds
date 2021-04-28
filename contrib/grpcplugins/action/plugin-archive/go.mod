module github.com/ovh/cds/contrib/grpcplugins/action/archive

replace github.com/ovh/cds => ../../../../

go 1.16

require (
	github.com/golang/protobuf v1.5.0
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/ovh/cds v0.0.0-00010101000000-000000000000
	github.com/pkg/errors v0.9.1 // indirect
	github.com/stretchr/testify v1.6.1
)
