module github.com/ovh/cds/contrib/grpcplugins/action/kafka-publish

replace github.com/ovh/cds => ../../../../

replace github.com/go-gorp/gorp => github.com/yesnault/gorp v2.0.1-0.20200325154225-2dc6d8c2da37+incompatible

go 1.17

require (
	github.com/Shopify/sarama v1.27.0
	github.com/bgentry/speakeasy v0.1.0
	github.com/fsamin/go-shredder v0.0.0-20180118184739-b2488aedb5be
	github.com/golang/protobuf v1.5.0
	github.com/ovh/cds v0.0.0-00010101000000-000000000000
	github.com/phayes/permbits v0.0.0-20190612203442-39d7c581d2ee
	github.com/stretchr/testify v1.6.1
	gopkg.in/urfave/cli.v1 v1.20.0
)

require (
	contrib.go.opencensus.io/exporter/jaeger v0.1.0 // indirect
	contrib.go.opencensus.io/exporter/prometheus v0.1.0 // indirect
	github.com/SSSaaS/sssa-golang v0.0.0-20170502204618-d37d7782d752 // indirect
	github.com/andybalholm/brotli v1.0.0 // indirect
	github.com/aokoli/goutils v1.1.0 // indirect
	github.com/apache/thrift v0.12.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/eapache/go-resiliency v1.2.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20180814174437-776d5712da21 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/fatih/color v1.7.0 // indirect
	github.com/fsamin/go-dump v1.0.9 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-gorp/gorp v2.0.0+incompatible // indirect
	github.com/go-redis/redis v6.15.2+incompatible // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/gookit/color v1.4.2 // indirect
	github.com/gorilla/context v1.1.1 // indirect
	github.com/gorilla/mux v1.6.2 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jcmturner/gofork v1.0.0 // indirect
	github.com/jfrog/gofrog v1.0.6 // indirect
	github.com/jfrog/jfrog-client-go v0.22.3 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kevinburke/ssh_config v0.0.0-20180830205328-81db2a75821e // indirect
	github.com/klauspost/compress v1.10.10 // indirect
	github.com/klauspost/pgzip v1.2.4 // indirect
	github.com/lib/pq v1.9.0 // indirect
	github.com/maruel/panicparse/v2 v2.2.0 // indirect
	github.com/mattn/go-colorable v0.1.11 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.1 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/mholt/archiver/v3 v3.5.0 // indirect
	github.com/miscreant/miscreant.go v0.0.0-20200214223636-26d376326b75 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/hashstructure v0.0.0-20170609045927-2bca23e0e452 // indirect
	github.com/mitchellh/mapstructure v1.1.2 // indirect
	github.com/mndrix/tap-go v0.0.0-20170113192335-56cca451570b // indirect
	github.com/nwaples/rardecode v1.1.0 // indirect
	github.com/olekukonko/tablewriter v0.0.0-20160621093029-daf2955e742c // indirect
	github.com/onsi/ginkgo v1.11.0 // indirect
	github.com/onsi/gomega v1.7.0 // indirect
	github.com/ovh/cds/sdk/interpolate v0.0.0-20190319104452-71125b036b25 // indirect
	github.com/ovh/configstore v0.3.3-0.20200701085609-a539fcf61db5 // indirect
	github.com/ovh/symmecrypt v0.5.1 // indirect
	github.com/ovh/venom v0.25.0 // indirect
	github.com/pborman/uuid v0.0.0-20170612153648-e790cca94e6c // indirect
	github.com/pelletier/go-buffruneio v0.2.0 // indirect
	github.com/pierrec/lz4 v2.5.2+incompatible // indirect
	github.com/pierrec/lz4/v4 v4.0.3 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.1.0 // indirect
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4 // indirect
	github.com/prometheus/common v0.6.0 // indirect
	github.com/prometheus/procfs v0.0.3 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0 // indirect
	github.com/rockbears/log v0.4.0 // indirect
	github.com/rubenv/sql-migrate v0.0.0-20160620083229-6f4757563362 // indirect
	github.com/sergi/go-diff v1.0.0 // indirect
	github.com/sguiheux/go-coverage v0.0.0-20190710153556-287b082a7197 // indirect
	github.com/sirupsen/logrus v1.7.0 // indirect
	github.com/smartystreets/assertions v0.0.0-20180927180507-b2de0cb4f26d // indirect
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cobra v1.1.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/src-d/gcfg v1.4.0 // indirect
	github.com/ulikunitz/xz v0.5.8 // indirect
	github.com/xanzy/ssh-agent v0.2.0 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	github.com/xo/terminfo v0.0.0-20210125001918-ca9a967f8778 // indirect
	go.opencensus.io v0.22.3 // indirect
	go.uber.org/atomic v1.6.0 // indirect
	go.uber.org/multierr v1.5.0 // indirect
	go.uber.org/zap v1.16.0 // indirect
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83 // indirect
	golang.org/x/net v0.0.0-20210224082022-3d97a244fca7 // indirect
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9 // indirect
	golang.org/x/sys v0.0.0-20211015200801-69063c4bb744 // indirect
	golang.org/x/term v0.0.0-20210220032956-6a3ed077a48d // indirect
	golang.org/x/text v0.3.4 // indirect
	google.golang.org/api v0.20.0 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/grpc v1.27.1 // indirect
	google.golang.org/protobuf v1.26.0 // indirect
	gopkg.in/AlecAivazis/survey.v1 v1.7.1 // indirect
	gopkg.in/gorp.v1 v1.7.1 // indirect
	gopkg.in/jcmturner/aescts.v1 v1.0.1 // indirect
	gopkg.in/jcmturner/dnsutils.v1 v1.0.1 // indirect
	gopkg.in/jcmturner/gokrb5.v7 v7.5.0 // indirect
	gopkg.in/jcmturner/rpc.v1 v1.1.0 // indirect
	gopkg.in/spacemonkeygo/httpsig.v0 v0.0.0-20170228231032-6732593ec966 // indirect
	gopkg.in/square/go-jose.v2 v2.3.1 // indirect
	gopkg.in/src-d/go-billy.v4 v4.3.0 // indirect
	gopkg.in/src-d/go-git.v4 v4.8.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200601152816-913338de1bd2 // indirect
)
