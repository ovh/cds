module github.com/ovh/cds/contrib/grpcplugins/action/clair

go 1.16

replace github.com/ovh/cds => ../../../../

replace github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200528182317-b47e74255811+incompatible

replace github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.4.2

replace github.com/codegangsta/cli v1.22.2 => github.com/urfave/cli v1.22.2

replace github.com/prometheus/client_golang v1.1.0 => github.com/prometheus/client_golang v0.9.4

replace github.com/opencontainers/runc v0.1.1 => github.com/opencontainers/runc v1.0.0-rc9

require (
	github.com/Microsoft/hcsshim v0.8.9 // indirect
	github.com/containerd/containerd v1.3.4 // indirect
	github.com/containerd/continuity v0.0.0-20200413184840-d3ef23f19fbb // indirect
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.13.1
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/golang/protobuf v1.5.0
	github.com/jgsqware/xnet v0.0.0-20170203143001-13630f0737d2
	github.com/mattn/go-shellwords v1.0.10 // indirect
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/runc v0.1.1 // indirect
	github.com/opencontainers/runtime-spec v1.0.2 // indirect
	github.com/opencontainers/selinux v1.5.2 // indirect
	github.com/ovh/cds v0.0.0-00010101000000-000000000000
	github.com/prometheus/client_golang v1.1.0
	github.com/quay/clair/v2 v2.1.4
	github.com/spf13/viper v1.7.0
	github.com/vbatts/tar-split v0.11.1 // indirect
	golang.org/x/net v0.0.0-20210224082022-3d97a244fca7
)
