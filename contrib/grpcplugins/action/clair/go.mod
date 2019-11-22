module github.com/ovh/cds/contrib/grpcplugins/action/clair

replace github.com/ovh/cds => ../../../../

replace github.com/docker/docker => github.com/docker/engine v0.0.0-20180816081446-320063a2ad06

go 1.13

require (
	github.com/artyom/untar v1.0.0
	github.com/coreos/clair v2.1.0+incompatible
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.13.1
	github.com/golang/protobuf v1.3.2
	github.com/jgsqware/xnet v0.0.0-20170203143001-13630f0737d2
	github.com/opencontainers/go-digest v1.0.0-rc1
	github.com/ovh/cds v0.0.0-00010101000000-000000000000
	github.com/spf13/viper v1.5.0
	golang.org/x/net v0.0.0-20191125084936-ffdde1057850
)
