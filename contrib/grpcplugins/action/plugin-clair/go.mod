module github.com/ovh/cds/contrib/grpcplugins/action/clair

go 1.17

replace github.com/ovh/cds => ../../../../

replace github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200528182317-b47e74255811+incompatible

replace github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.4.2

replace github.com/codegangsta/cli v1.22.2 => github.com/urfave/cli v1.22.2

replace github.com/prometheus/client_golang v1.1.0 => github.com/prometheus/client_golang v0.9.4

replace github.com/opencontainers/runc v0.1.1 => github.com/opencontainers/runc v1.0.0-rc9

require (
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.13.1
	github.com/golang/protobuf v1.5.0
	github.com/jgsqware/xnet v0.0.0-20170203143001-13630f0737d2
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/opencontainers/go-digest v1.0.0
	github.com/ovh/cds v0.0.0-00010101000000-000000000000
	github.com/prometheus/client_golang v1.1.0
	github.com/quay/clair/v2 v2.1.4
	github.com/spf13/viper v1.7.0
	golang.org/x/net v0.0.0-20210224082022-3d97a244fca7
)

require (
	contrib.go.opencensus.io/exporter/jaeger v0.1.0 // indirect
	contrib.go.opencensus.io/exporter/prometheus v0.1.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Microsoft/go-winio v0.4.15-0.20190919025122-fc70bd9a86b5 // indirect
	github.com/Microsoft/hcsshim v0.8.9 // indirect
	github.com/andybalholm/brotli v1.0.0 // indirect
	github.com/aokoli/goutils v1.1.0 // indirect
	github.com/apache/thrift v0.12.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/containerd/containerd v1.3.4 // indirect
	github.com/containerd/continuity v0.0.0-20200413184840-d3ef23f19fbb // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/docker/go-connections v0.3.0 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/eapache/go-resiliency v1.2.0 // indirect
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/fatih/color v1.7.0 // indirect
	github.com/fernet/fernet-go v0.0.0-20151007213151-1b2437bc582b // indirect
	github.com/frankban/quicktest v1.10.0 // indirect
	github.com/fsamin/go-dump v1.0.9 // indirect
	github.com/fsnotify/fsnotify v1.4.7 // indirect
	github.com/go-gorp/gorp v2.0.0+incompatible // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/gookit/color v1.4.2 // indirect
	github.com/gorilla/context v1.1.1 // indirect
	github.com/gorilla/mux v1.6.2 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jfrog/gofrog v1.0.6 // indirect
	github.com/jfrog/jfrog-client-go v0.22.3 // indirect
	github.com/julienschmidt/httprouter v1.2.0 // indirect
	github.com/kevinburke/ssh_config v0.0.0-20180830205328-81db2a75821e // indirect
	github.com/klauspost/compress v1.10.10 // indirect
	github.com/klauspost/pgzip v1.2.4 // indirect
	github.com/magiconair/properties v1.8.1 // indirect
	github.com/maruel/panicparse/v2 v2.2.0 // indirect
	github.com/mattn/go-colorable v0.1.11 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-shellwords v1.0.10 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/mholt/archiver/v3 v3.5.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/hashstructure v0.0.0-20170609045927-2bca23e0e452 // indirect
	github.com/mitchellh/mapstructure v1.1.2 // indirect
	github.com/mndrix/tap-go v0.0.0-20170113192335-56cca451570b // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/nwaples/rardecode v1.1.0 // indirect
	github.com/onsi/ginkgo v1.11.0 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/opencontainers/runc v0.1.1 // indirect
	github.com/opencontainers/runtime-spec v1.0.2 // indirect
	github.com/opencontainers/selinux v1.5.2 // indirect
	github.com/ovh/cds/sdk/interpolate v0.0.0-20190319104452-71125b036b25 // indirect
	github.com/ovh/venom v0.25.0 // indirect
	github.com/pborman/uuid v0.0.0-20170612153648-e790cca94e6c // indirect
	github.com/pelletier/go-buffruneio v0.2.0 // indirect
	github.com/pelletier/go-toml v1.8.0 // indirect
	github.com/pierrec/lz4 v2.5.2+incompatible // indirect
	github.com/pierrec/lz4/v4 v4.0.3 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4 // indirect
	github.com/prometheus/common v0.6.0 // indirect
	github.com/prometheus/procfs v0.0.3 // indirect
	github.com/rockbears/log v0.4.0 // indirect
	github.com/sergi/go-diff v1.0.0 // indirect
	github.com/sguiheux/go-coverage v0.0.0-20190710153556-287b082a7197 // indirect
	github.com/sirupsen/logrus v1.7.0 // indirect
	github.com/smartystreets/assertions v0.0.0-20180927180507-b2de0cb4f26d // indirect
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cast v1.3.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/src-d/gcfg v1.4.0 // indirect
	github.com/stretchr/testify v1.6.1 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	github.com/ulikunitz/xz v0.5.8 // indirect
	github.com/vbatts/tar-split v0.11.1 // indirect
	github.com/xanzy/ssh-agent v0.2.0 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	github.com/xo/terminfo v0.0.0-20210125001918-ca9a967f8778 // indirect
	go.opencensus.io v0.22.3 // indirect
	go.uber.org/atomic v1.6.0 // indirect
	go.uber.org/multierr v1.5.0 // indirect
	go.uber.org/zap v1.16.0 // indirect
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83 // indirect
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9 // indirect
	golang.org/x/sys v0.0.0-20211015200801-69063c4bb744 // indirect
	golang.org/x/term v0.0.0-20210220032956-6a3ed077a48d // indirect
	golang.org/x/text v0.3.4 // indirect
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba // indirect
	google.golang.org/api v0.20.0 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/grpc v1.27.1 // indirect
	google.golang.org/protobuf v1.26.0 // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/ini.v1 v1.51.0 // indirect
	gopkg.in/src-d/go-billy.v4 v4.3.0 // indirect
	gopkg.in/src-d/go-git.v4 v4.8.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200601152816-913338de1bd2 // indirect
)
