module github.com/ovh/cds/contrib/grpcplugins/action/venom

replace github.com/ovh/cds => ../../../../

go 1.13

require (
	github.com/golang/protobuf v1.3.2
	github.com/ovh/cds v0.0.0-00010101000000-000000000000
	github.com/ovh/venom v0.26.0
	github.com/stretchr/testify v1.4.0
	gopkg.in/yaml.v2 v2.2.7
)
