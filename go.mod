module github.com/ovh/cds

go 1.22

require (
	code.gitea.io/sdk/gitea v0.15.1-0.20220530220844-359c771ce3d2
	contrib.go.opencensus.io/exporter/jaeger v0.2.1
	contrib.go.opencensus.io/exporter/prometheus v0.4.2
	github.com/Masterminds/semver/v3 v3.2.1
	github.com/RackSec/srslog v0.0.0-20180709174129-a4725f04ec91
	github.com/Shopify/sarama v1.36.0
	github.com/andygrunwald/go-gerrit v0.0.0-20181207071854-19ef3e9332a4
	github.com/antlr/antlr4/runtime/Go/antlr/v4 v4.0.0-20230321174746-8dcc6526cfb1
	github.com/aws/aws-sdk-go v1.43.21
	github.com/blang/semver v3.5.1+incompatible
	github.com/buger/goterm v0.0.0-20170918171949-d443b9114f9c
	github.com/bugsnag/osext v0.0.0-20130617224835-0dd3f918b21b
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e
	github.com/confluentinc/bincover v0.2.0
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/docker/cli v25.0.1+incompatible
	github.com/docker/distribution v2.8.3+incompatible
	github.com/docker/docker v25.0.5+incompatible
	github.com/docker/go-connections v0.5.0
	github.com/docker/go-units v0.5.0
	github.com/eapache/go-resiliency v1.3.0
	github.com/fatih/color v1.16.0
	github.com/fsamin/go-dump v1.0.9
	github.com/fsamin/go-repo v0.4.1
	github.com/fsamin/go-shredder v0.0.0-20180118184739-b2488aedb5be
	github.com/fujiwara/shapeio v0.0.0-20170602072123-c073257dd745
	github.com/go-gorp/gorp v2.0.0+incompatible
	github.com/go-redis/redis v6.15.2+incompatible
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.3
	github.com/gophercloud/gophercloud v0.1.0
	github.com/gorhill/cronexpr v0.0.0-20161205141322-d520615e531a
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.5.0
	github.com/hashicorp/vault/api v1.0.4
	github.com/iancoleman/orderedmap v0.3.0
	github.com/inconshreveable/go-update v0.0.0-20160112193335-8152e7eb6ccf
	github.com/jfrog/build-info-go v1.9.23
	github.com/jfrog/jfrog-client-go v1.37.1
	github.com/jordan-wright/email v4.0.1-0.20200917010138-e1c00e156980+incompatible
	github.com/kardianos/osext v0.0.0-20170510131534-ae77be60afb1
	github.com/keybase/go-crypto v0.0.0-20181127160227-255a5089e85a
	github.com/keybase/go-keychain v0.0.0-20190828020956-aa639f275ae1
	github.com/keybase/go.dbus v0.0.0-20190710215703-a33a09c8a604
	github.com/lib/pq v1.10.9
	github.com/maruel/panicparse/v2 v2.2.2
	github.com/mattn/go-sqlite3 v1.14.16
	github.com/mattn/go-zglob v0.0.4
	github.com/mcuadros/go-defaults v0.0.0-20161116231230-e1c978be3307
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d
	github.com/mholt/archiver/v3 v3.5.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/hashstructure v0.0.0-20170609045927-2bca23e0e452
	github.com/mitchellh/mapstructure v1.5.0
	github.com/moby/moby v25.0.3+incompatible
	github.com/mum4k/termdash v0.10.0
	github.com/nbutton23/zxcvbn-go v0.0.0-20180912185939-ae427f1e4c1d
	github.com/ncw/swift v1.0.52
	github.com/olekukonko/tablewriter v0.0.5
	github.com/olivere/elastic/v7 v7.0.32
	github.com/ovh/cds/sdk/interpolate v0.0.0-20190319104452-71125b036b25
	github.com/ovh/configstore v0.3.3-0.20200701085609-a539fcf61db5
	github.com/ovh/symmecrypt v0.5.1
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pborman/uuid v1.2.0
	github.com/pelletier/go-toml v1.9.5
	github.com/pkg/browser v0.0.0-20170505125900-c90ca0c84f15
	github.com/pkg/errors v0.9.1
	github.com/rockbears/log v0.10.2
	github.com/rockbears/yaml v0.4.0
	github.com/rubenv/sql-migrate v1.5.2
	github.com/sguiheux/go-coverage v0.0.0-20190710153556-287b082a7197
	github.com/sguiheux/jsonschema v0.0.0-20240314085137-97ecc280683c
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/afero v1.11.0
	github.com/spf13/cast v1.6.0
	github.com/spf13/cobra v1.8.0
	github.com/spf13/viper v1.18.2
	github.com/srerickson/checksum v0.10.0
	github.com/streadway/amqp v0.0.0-20180528204448-e5adc2ada8b8
	github.com/stretchr/testify v1.8.4
	github.com/studio-b12/gowebdav v0.0.0-20200303150724-9380631c29a1
	github.com/tevino/abool v0.0.0-20170917061928-9b9efcf221b5
	github.com/urfave/cli v1.22.12
	github.com/vmware/go-nfs-client v0.0.0-20190605212624-d43b92724c1b
	github.com/vmware/govmomi v0.23.0
	github.com/xanzy/go-gitlab v0.15.0
	github.com/xeipuuv/gojsonschema v1.2.0
	github.com/yesnault/go-toml v0.0.0-20191205182532-f5ef6cee7945
	github.com/yuin/gluare v0.0.0-20170607022532-d7c94f1a80ed
	github.com/yuin/gopher-lua v0.0.0-20170901023928-8c2befcd3908
	go.opencensus.io v0.24.0
	golang.org/x/crypto v0.21.0
	golang.org/x/net v0.23.0
	golang.org/x/oauth2 v0.15.0
	golang.org/x/sys v0.20.0
	golang.org/x/text v0.14.0
	golang.org/x/time v0.5.0
	google.golang.org/grpc v1.59.0
	google.golang.org/protobuf v1.33.0
	gopkg.in/AlecAivazis/survey.v1 v1.7.1
	gopkg.in/h2non/gock.v1 v1.1.2
	gopkg.in/ldap.v2 v2.5.1
	gopkg.in/spacemonkeygo/httpsig.v0 v0.0.0-20170228231032-6732593ec966
	gopkg.in/square/go-jose.v2 v2.5.1
	gopkg.in/yaml.v2 v2.4.0
	helm.sh/helm/v3 v3.14.2
	k8s.io/api v0.29.1
	k8s.io/apimachinery v0.29.1
	k8s.io/client-go v0.29.1
	k8s.io/helm v2.17.0+incompatible
	layeh.com/gopher-json v0.0.0-20201124131017-552bb3c4c3bf
	sigs.k8s.io/yaml v1.3.0
)

require (
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/distribution/reference v0.5.0 // indirect
	github.com/jfrog/archiver/v3 v3.6.0 // indirect
	github.com/nats-io/nats.go v1.31.0 // indirect
	github.com/nats-io/nkeys v0.4.6 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pelletier/go-toml/v2 v2.1.0 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	go.etcd.io/etcd/client/v3 v3.5.10 // indirect
)

require (
	cloud.google.com/go v0.110.10 // indirect
	cloud.google.com/go/compute v1.23.3 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cloud.google.com/go/firestore v1.14.0 // indirect
	cloud.google.com/go/longrunning v0.5.4 // indirect
	dario.cat/mergo v1.0.0 // indirect
	github.com/AdaLogics/go-fuzz-headers v0.0.0-20230811130428-ced1acdcaa24 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/BurntSushi/toml v1.3.2 // indirect
	github.com/CycloneDX/cyclonedx-go v0.8.0 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/Microsoft/hcsshim v0.11.4 // indirect
	github.com/Netflix/go-expect v0.0.0-20180928190340-9d1f4485533b // indirect
	github.com/ProtonMail/go-crypto v1.0.0 // indirect
	github.com/SSSaaS/sssa-golang v0.0.0-20170502204618-d37d7782d752 // indirect
	github.com/andybalholm/brotli v1.1.0 // indirect
	github.com/aokoli/goutils v1.1.1 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/chartmuseum/helm-push v0.10.4
	github.com/cloudflare/circl v1.3.7 // indirect
	github.com/containerd/containerd v1.7.11 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.3 // indirect
	github.com/cyphar/filepath-securejoin v0.2.4 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/docker/docker-credential-helpers v0.7.0 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/dsnet/compress v0.0.2-0.20210315054119-f66993602bf5 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20180814174437-776d5712da21 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/evanphx/json-patch v5.7.0+incompatible // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/forPelevin/gomoji v1.1.8 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.5.0 // indirect
	github.com/go-git/go-git/v5 v5.11.0 // indirect
	github.com/go-gorp/gorp/v3 v3.1.0 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-logr/logr v1.3.0 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/google/uuid v1.4.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.12.0 // indirect
	github.com/gookit/color v1.5.4 // indirect
	github.com/gregjones/httpcache v0.0.0-20180305231024-9cad4c3443a7 // indirect
	github.com/h2non/parth v0.0.0-20190131123155-b4df798d6542 // indirect
	github.com/hashicorp/consul/api v1.25.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.7 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.1.6 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/go-version v1.5.0 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/serf v0.10.1 // indirect
	github.com/hashicorp/vault/sdk v0.8.1 // indirect
	github.com/hinshun/vt10x v0.0.0-20180809195222-d55458df857c // indirect
	github.com/hpcloud/tail v1.0.0 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.3 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/jfrog/gofrog v1.6.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/klauspost/cpuid/v2 v2.2.3 // indirect
	github.com/klauspost/pgzip v1.2.6 // indirect
	github.com/kr/pty v1.1.8 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/miscreant/miscreant.go v0.0.0-20200214223636-26d376326b75 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/locker v1.0.1 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nsf/termbox-go v0.0.0-20190817171036-93860e161317 // indirect
	github.com/nwaples/rardecode v1.1.3 // indirect
	github.com/onsi/ginkgo v1.10.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc6 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/pquerna/cachecontrol v0.0.0-20200819021114-67c6ae64274f // indirect
	github.com/prometheus/client_golang v1.16.0 // indirect
	github.com/prometheus/client_model v0.4.0 // indirect
	github.com/prometheus/common v0.44.0 // indirect
	github.com/prometheus/procfs v0.10.1 // indirect
	github.com/prometheus/statsd_exporter v0.22.7 // indirect
	github.com/rasky/go-xdr v0.0.0-20170124162913-1a41d1a06c93 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/rivo/uniseg v0.4.3 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/sagikazarmark/crypt v0.17.0 // indirect
	github.com/sergi/go-diff v1.3.1 // indirect
	github.com/skeema/knownhosts v1.2.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/uber/jaeger-client-go v2.25.0+incompatible // indirect
	github.com/ulikunitz/xz v0.5.11 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	github.com/xlab/treeprint v1.2.0 // indirect
	github.com/xo/terminfo v0.0.0-20210125001918-ca9a967f8778 // indirect
	github.com/ziutek/mymysql v1.5.4 // indirect
	go.etcd.io/etcd/api/v3 v3.5.10 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.10 // indirect
	go.etcd.io/etcd/client/v2 v2.305.10 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.45.0 // indirect
	go.opentelemetry.io/otel v1.19.0 // indirect
	go.opentelemetry.io/otel/metric v1.19.0 // indirect
	go.opentelemetry.io/otel/trace v1.19.0 // indirect
	go.starlark.net v0.0.0-20230525235612-a134d8f9ddca // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/exp v0.0.0-20240213143201-ec583247a57a // indirect
	golang.org/x/mod v0.16.0 // indirect
	golang.org/x/sync v0.6.0 // indirect
	golang.org/x/term v0.18.0 // indirect
	golang.org/x/tools v0.19.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/api v0.153.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20231106174013-bbf56f31fb17 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20231106174013-bbf56f31fb17 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231120223509-83a465c0220f // indirect
	gopkg.in/asn1-ber.v1 v1.0.0-20181015200546-f715ec2f112d // indirect
	gopkg.in/fsnotify.v1 v1.4.7 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.29.0 // indirect
	k8s.io/cli-runtime v0.29.0 // indirect
	k8s.io/klog/v2 v2.110.1 // indirect
	k8s.io/kube-openapi v0.0.0-20231010175941-2dd684a91f00 // indirect
	k8s.io/utils v0.0.0-20230726121419-3b25d923346b // indirect
	oras.land/oras-go v1.2.5 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/kustomize/api v0.13.5-0.20230601165947-6ce0bf390ce3 // indirect
	sigs.k8s.io/kustomize/kyaml v0.14.3-0.20230601165947-6ce0bf390ce3 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
)

replace gopkg.in/yaml.v2 v2.4.0 => gopkg.in/yaml.v2 v2.3.0

replace github.com/vmware/go-nfs-client => github.com/sguiheux/go-nfs-client v0.0.0-20210311091651-4f075a6103cc

replace github.com/go-gorp/gorp => github.com/yesnault/gorp v2.0.1-0.20200325154225-2dc6d8c2da37+incompatible

replace github.com/ovh/cds/sdk/interpolate => ./sdk/interpolate

replace github.com/keybase/go-crypto => github.com/Alkorin/crypto v0.0.0-20190802123352-5ea49ae5e604

replace github.com/ovh/cds/tools/smtpmock => ./tools/smtpmock

replace github.com/keybase/go-keychain => github.com/yesnault/go-keychain v0.0.0-20190829085436-f78f7ae28786

replace github.com/coreos/etcd v3.3.10+incompatible => github.com/coreos/etcd v3.3.13+incompatible

replace github.com/jordan-wright/email => github.com/yesnault/email v0.0.0-20201006155628-d88bfe11e7f1
