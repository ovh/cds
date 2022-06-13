module github.com/ovh/cds

go 1.18

require (
	code.gitea.io/sdk/gitea v0.15.1
	contrib.go.opencensus.io/exporter/jaeger v0.1.0
	contrib.go.opencensus.io/exporter/prometheus v0.1.0
	github.com/Shopify/sarama v1.30.0
	github.com/alecthomas/jsonschema v0.0.0-20200123075451-43663a393755
	github.com/andygrunwald/go-gerrit v0.0.0-20181207071854-19ef3e9332a4
	github.com/aws/aws-sdk-go v1.19.11
	github.com/blang/semver v3.5.1+incompatible
	github.com/buger/goterm v0.0.0-20170918171949-d443b9114f9c
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e
	github.com/confluentinc/bincover v0.2.0
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/docker/distribution v2.8.0+incompatible
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.3.0
	github.com/eapache/go-resiliency v1.2.0
	github.com/fatih/color v1.13.0
	github.com/fsamin/go-dump v1.0.9
	github.com/fsamin/go-repo v0.1.10
	github.com/fsamin/go-shredder v0.0.0-20180118184739-b2488aedb5be
	github.com/fujiwara/shapeio v0.0.0-20170602072123-c073257dd745
	github.com/gambol99/go-marathon v0.0.0-20170922093320-ec4a50170df7
	github.com/ghodss/yaml v1.0.0
	github.com/go-gorp/gorp v2.0.0+incompatible
	github.com/go-redis/redis v6.15.2+incompatible
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/gophercloud/gophercloud v0.0.0-20190504011306-6f9faf57fddc
	github.com/gorhill/cronexpr v0.0.0-20161205141322-d520615e531a
	github.com/gorilla/handlers v0.0.0-20160816184729-a5775781a543
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/vault/api v1.0.4
	github.com/iancoleman/orderedmap v0.0.0-20190318233801-ac98e3ecb4b0
	github.com/inconshreveable/go-update v0.0.0-20160112193335-8152e7eb6ccf
	github.com/jfrog/jfrog-client-go v1.5.1
	github.com/jordan-wright/email v4.0.1-0.20200917010138-e1c00e156980+incompatible
	github.com/kardianos/osext v0.0.0-20170510131534-ae77be60afb1
	github.com/keybase/go-crypto v0.0.0-20181127160227-255a5089e85a
	github.com/keybase/go-keychain v0.0.0-20190828020956-aa639f275ae1
	github.com/keybase/go.dbus v0.0.0-20190710215703-a33a09c8a604
	github.com/lib/pq v1.9.0
	github.com/maruel/panicparse/v2 v2.2.2
	github.com/mattn/go-sqlite3 v1.9.0
	github.com/mattn/go-zglob v0.0.1
	github.com/mcuadros/go-defaults v0.0.0-20161116231230-e1c978be3307
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/mitchellh/hashstructure v0.0.0-20170609045927-2bca23e0e452
	github.com/mitchellh/mapstructure v1.4.3
	github.com/mum4k/termdash v0.10.0
	github.com/nbutton23/zxcvbn-go v0.0.0-20180912185939-ae427f1e4c1d
	github.com/ncw/swift v1.0.52
	github.com/olekukonko/tablewriter v0.0.0-20160621093029-daf2955e742c
	github.com/ovh/cds/sdk/interpolate v0.0.0-20190319104452-71125b036b25
	github.com/ovh/configstore v0.3.3-0.20200701085609-a539fcf61db5
	github.com/ovh/symmecrypt v0.5.1
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pborman/uuid v0.0.0-20170612153648-e790cca94e6c
	github.com/pelletier/go-toml v1.9.4
	github.com/pkg/browser v0.0.0-20170505125900-c90ca0c84f15
	github.com/pkg/errors v0.9.1
	github.com/rockbears/log v0.6.0
	github.com/rubenv/sql-migrate v0.0.0-20160620083229-6f4757563362
	github.com/satori/go.uuid v1.2.0
	github.com/sguiheux/go-coverage v0.0.0-20190710153556-287b082a7197
	github.com/shirou/gopsutil v0.0.0-20170406131756-e49a95f3d5f8
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/afero v1.8.1
	github.com/spf13/cast v1.4.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.10.1
	github.com/streadway/amqp v0.0.0-20180528204448-e5adc2ada8b8
	github.com/stretchr/testify v1.7.0
	github.com/studio-b12/gowebdav v0.0.0-20200303150724-9380631c29a1
	github.com/tevino/abool v0.0.0-20170917061928-9b9efcf221b5
	github.com/urfave/cli v1.20.0
	github.com/vmware/go-nfs-client v0.0.0-20190605212624-d43b92724c1b
	github.com/vmware/govmomi v0.23.0
	github.com/whilp/git-urls v0.0.0-20160530060445-31bac0d230fa
	github.com/xanzy/go-gitlab v0.15.0
	github.com/yesnault/go-toml v0.0.0-20191205182532-f5ef6cee7945
	github.com/yuin/gluare v0.0.0-20170607022532-d7c94f1a80ed
	github.com/yuin/gopher-lua v0.0.0-20170901023928-8c2befcd3908
	go.opencensus.io v0.23.0
	golang.org/x/crypto v0.0.0-20211108221036-ceb1ce70b4fa
	golang.org/x/net v0.0.0-20210917221730-978cfadd31cf
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
	golang.org/x/sys v0.0.0-20220408201424-a24fb2fb8a0f
	golang.org/x/text v0.3.7
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba
	google.golang.org/grpc v1.43.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/AlecAivazis/survey.v1 v1.7.1
	gopkg.in/h2non/gock.v1 v1.0.14
	gopkg.in/ldap.v2 v2.5.1
	gopkg.in/olivere/elastic.v6 v6.2.17
	gopkg.in/spacemonkeygo/httpsig.v0 v0.0.0-20170228231032-6732593ec966
	gopkg.in/square/go-jose.v2 v2.3.1
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
	layeh.com/gopher-json v0.0.0-20201124131017-552bb3c4c3bf
)

require (
	cloud.google.com/go v0.99.0 // indirect
	cloud.google.com/go/firestore v1.6.1 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.12 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.5 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/logger v0.2.0 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/Microsoft/go-winio v0.4.16 // indirect
	github.com/Netflix/go-expect v0.0.0-20180928190340-9d1f4485533b // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20210428141323-04723f9f07d7 // indirect
	github.com/SSSaaS/sssa-golang v0.0.0-20170502204618-d37d7782d752 // indirect
	github.com/acomagu/bufpipe v1.0.3 // indirect
	github.com/andybalholm/brotli v1.0.1 // indirect
	github.com/aokoli/goutils v1.1.0 // indirect
	github.com/apache/thrift v0.12.0 // indirect
	github.com/armon/go-metrics v0.3.10 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/census-instrumentation/opencensus-proto v0.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/cncf/udpa/go v0.0.0-20210930031921-04548b0d99d4 // indirect
	github.com/cncf/xds/go v0.0.0-20211130200136-a8f946100490 // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/go-units v0.3.2 // indirect
	github.com/donovanhide/eventsource v0.0.0-20170630084216-b8f31a59085e // indirect
	github.com/dsnet/compress v0.0.2-0.20210315054119-f66993602bf5 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20180814174437-776d5712da21 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/envoyproxy/go-control-plane v0.10.1 // indirect
	github.com/envoyproxy/protoc-gen-validate v0.6.2 // indirect
	github.com/form3tech-oss/jwt-go v3.2.2+incompatible // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/go-git/gcfg v1.5.0 // indirect
	github.com/go-git/go-billy/v5 v5.3.1 // indirect
	github.com/go-git/go-git/v5 v5.4.2 // indirect
	github.com/go-logr/logr v0.4.0 // indirect
	github.com/go-sql-driver/mysql v1.4.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.1.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/googleapis/gax-go/v2 v2.1.1 // indirect
	github.com/googleapis/gnostic v0.4.1 // indirect
	github.com/gookit/color v1.4.2 // indirect
	github.com/h2non/parth v0.0.0-20190131123155-b4df798d6542 // indirect
	github.com/hashicorp/consul/api v1.12.0 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.0.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-multierror v1.1.0 // indirect
	github.com/hashicorp/go-retryablehttp v0.5.4 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/go-version v1.2.1 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/serf v0.9.6 // indirect
	github.com/hashicorp/vault/sdk v0.1.13 // indirect
	github.com/hinshun/vt10x v0.0.0-20180809195222-d55458df857c // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.0.0 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.2 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/jfrog/gofrog v1.0.6 // indirect
	github.com/jmespath/go-jmespath v0.0.0-20180206201540-c2b33e8439af // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kevinburke/ssh_config v0.0.0-20201106050909-4977a11b4351 // indirect
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/klauspost/pgzip v1.2.5 // indirect
	github.com/kr/pty v1.1.8 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/mailru/easyjson v0.0.0-20190626092158-b2ccc519800e // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.1 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mholt/archiver/v3 v3.5.1-0.20210618180617-81fac4ba96e4 // indirect
	github.com/miscreant/miscreant.go v0.0.0-20200214223636-26d376326b75 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/nsf/termbox-go v0.0.0-20190817171036-93860e161317 // indirect
	github.com/nwaples/rardecode v1.1.0 // indirect
	github.com/olivere/elastic v6.2.17+incompatible // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/pierrec/lz4/v4 v4.1.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/poy/onpar v0.0.0-20190519213022-ee068f8ea4d1 // indirect
	github.com/pquerna/cachecontrol v0.0.0-20200819021114-67c6ae64274f // indirect
	github.com/prometheus/client_golang v1.4.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.9.1 // indirect
	github.com/prometheus/procfs v0.0.8 // indirect
	github.com/rasky/go-xdr v0.0.0-20170124162913-1a41d1a06c93 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/sagikazarmark/crypt v0.4.0 // indirect
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	github.com/ulikunitz/xz v0.5.9 // indirect
	github.com/xanzy/ssh-agent v0.3.0 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	github.com/xo/terminfo v0.0.0-20210125001918-ca9a967f8778 // indirect
	github.com/ziutek/mymysql v1.5.4 // indirect
	go.etcd.io/etcd/api/v3 v3.5.1 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.1 // indirect
	go.etcd.io/etcd/client/v2 v2.305.1 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.17.0 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/term v0.0.0-20210220032956-6a3ed077a48d // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/api v0.63.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20211208223120-3a66f561d7aa // indirect
	gopkg.in/asn1-ber.v1 v1.0.0-20181015200546-f715ec2f112d // indirect
	gopkg.in/gorp.v1 v1.7.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.66.4 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	gotest.tools v2.1.0+incompatible // indirect
	k8s.io/klog/v2 v2.8.0 // indirect
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.1.0 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace gopkg.in/yaml.v2 v2.4.0 => gopkg.in/yaml.v2 v2.3.0

replace github.com/vmware/go-nfs-client => github.com/sguiheux/go-nfs-client v0.0.0-20210311091651-4f075a6103cc

replace github.com/alecthomas/jsonschema => github.com/sguiheux/jsonschema v0.2.0

replace github.com/go-gorp/gorp => github.com/yesnault/gorp v2.0.1-0.20200325154225-2dc6d8c2da37+incompatible

replace github.com/docker/docker => github.com/docker/engine v0.0.0-20180816081446-320063a2ad06

replace github.com/ovh/cds/sdk/interpolate => ./sdk/interpolate

replace github.com/keybase/go-crypto => github.com/Alkorin/crypto v0.0.0-20190802123352-5ea49ae5e604

replace github.com/ovh/cds/tools/smtpmock => ./tools/smtpmock

replace github.com/keybase/go-keychain => github.com/yesnault/go-keychain v0.0.0-20190829085436-f78f7ae28786

replace github.com/coreos/etcd v3.3.10+incompatible => github.com/coreos/etcd v3.3.13+incompatible

replace github.com/jordan-wright/email => github.com/yesnault/email v0.0.0-20201006155628-d88bfe11e7f1
