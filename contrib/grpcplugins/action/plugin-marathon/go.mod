module github.com/ovh/cds/contrib/grpcplugins/action/marathon

replace github.com/ovh/cds => ../../../../

go 1.16

require (
	github.com/gambol99/go-marathon v0.7.1
	github.com/golang/protobuf v1.5.0
	github.com/ovh/cds v0.0.0-00010101000000-000000000000
	github.com/ovh/cds/sdk/interpolate v0.0.0-20191126072910-b8d81d038865
	github.com/stretchr/testify v1.6.1
	github.com/xeipuuv/gojsonschema v1.2.0
)
