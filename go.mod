module github.com/ovh/cds

go 1.16

require (
	contrib.go.opencensus.io/exporter/jaeger v0.1.0
	contrib.go.opencensus.io/exporter/prometheus v0.1.0
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Azure/go-autorest v11.1.1+incompatible // indirect
	github.com/Microsoft/go-winio v0.4.7 // indirect
	github.com/Netflix/go-expect v0.0.0-20180928190340-9d1f4485533b // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/Shopify/sarama v1.27.0
	github.com/alecthomas/jsonschema v0.0.0-20200123075451-43663a393755
	github.com/andygrunwald/go-gerrit v0.0.0-20181207071854-19ef3e9332a4
	github.com/aws/aws-sdk-go v1.19.11
	github.com/blang/semver v3.5.1+incompatible
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869
	github.com/buger/goterm v0.0.0-20170918171949-d443b9114f9c
	github.com/chzyer/logex v1.1.10 // indirect
	github.com/chzyer/readline v0.0.0-20171208011716-f6d7a1f6fbf3
	github.com/chzyer/test v0.0.0-20180213035817-a1ea475d72b1 // indirect
	github.com/confluentinc/bincover v0.2.0
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/docker/distribution v2.7.0-rc.0+incompatible // indirect
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.3.0
	github.com/docker/go-units v0.3.2 // indirect
	github.com/donovanhide/eventsource v0.0.0-20170630084216-b8f31a59085e // indirect
	github.com/dsnet/compress v0.0.0-20171208185109-cc9eb1d7ad76 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/eapache/go-resiliency v1.2.0
	github.com/fatih/color v1.7.0
	github.com/fsamin/go-dump v1.0.9
	github.com/fsamin/go-repo v0.1.8
	github.com/fsamin/go-shredder v0.0.0-20180118184739-b2488aedb5be
	github.com/fujiwara/shapeio v0.0.0-20170602072123-c073257dd745
	github.com/gambol99/go-marathon v0.0.0-20170922093320-ec4a50170df7
	github.com/gin-contrib/sse v0.0.0-20190301062529-5545eab6dad3 // indirect
	github.com/gin-gonic/gin v1.3.0
	github.com/go-gorp/gorp v2.0.0+incompatible
	github.com/go-redis/redis v6.15.2+incompatible
	github.com/go-sql-driver/mysql v1.4.1 // indirect
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.3.2
	github.com/googleapis/gnostic v0.1.0 // indirect
	github.com/gophercloud/gophercloud v0.0.0-20190504011306-6f9faf57fddc
	github.com/gorhill/cronexpr v0.0.0-20161205141322-d520615e531a
	github.com/gorilla/handlers v0.0.0-20160816184729-a5775781a543
	github.com/gorilla/mux v1.6.2
	github.com/gorilla/websocket v1.4.2
	github.com/gregjones/httpcache v0.0.0-20190212212710-3befbb6ad0cc // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.9.6 // indirect
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/hashicorp/vault/api v1.0.4
	github.com/hinshun/vt10x v0.0.0-20180809195222-d55458df857c // indirect
	github.com/iancoleman/orderedmap v0.0.0-20190318233801-ac98e3ecb4b0
	github.com/imdario/mergo v0.0.0-20180119215619-163f41321a19 // indirect
	github.com/inconshreveable/go-update v0.0.0-20160112193335-8152e7eb6ccf
	github.com/itsjamie/gin-cors v0.0.0-20160420130702-97b4a9da7933
	github.com/jordan-wright/email v4.0.1-0.20200917010138-e1c00e156980+incompatible
	github.com/kardianos/osext v0.0.0-20170510131534-ae77be60afb1
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/keybase/go-crypto v0.0.0-20181127160227-255a5089e85a
	github.com/keybase/go-keychain v0.0.0-20190828020956-aa639f275ae1
	github.com/keybase/go.dbus v0.0.0-20190710215703-a33a09c8a604
	github.com/kr/pty v1.1.8 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lib/pq v1.9.0
	github.com/mailru/easyjson v0.0.0-20171120080333-32fa128f234d // indirect
	github.com/maruel/panicparse v1.3.0
	github.com/mattn/go-runewidth v0.0.1 // indirect
	github.com/mattn/go-sqlite3 v1.9.0
	github.com/mattn/go-zglob v0.0.1
	github.com/mcuadros/go-defaults v0.0.0-20161116231230-e1c978be3307
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/mitchellh/hashstructure v0.0.0-20170609045927-2bca23e0e452
	github.com/mitchellh/mapstructure v1.1.2
	github.com/mndrix/tap-go v0.0.0-20170113192335-56cca451570b // indirect
	github.com/mum4k/termdash v0.10.0
	github.com/nbutton23/zxcvbn-go v0.0.0-20180912185939-ae427f1e4c1d
	github.com/ncw/swift v1.0.52
	github.com/nsf/termbox-go v0.0.0-20190817171036-93860e161317 // indirect
	github.com/nwaples/rardecode v1.0.0 // indirect
	github.com/olekukonko/tablewriter v0.0.0-20160621093029-daf2955e742c
	github.com/olivere/elastic v6.2.17+incompatible // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/ovh/cds/sdk/interpolate v0.0.0-20190319104452-71125b036b25
	github.com/ovh/configstore v0.3.3-0.20200701085609-a539fcf61db5
	github.com/ovh/symmecrypt v0.5.1
	github.com/ovh/venom v0.25.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pborman/uuid v0.0.0-20170612153648-e790cca94e6c
	github.com/pelletier/go-toml v1.8.0
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/browser v0.0.0-20170505125900-c90ca0c84f15
	github.com/pkg/errors v0.8.1
	github.com/poy/onpar v0.0.0-20190519213022-ee068f8ea4d1 // indirect
	github.com/pquerna/cachecontrol v0.0.0-20200819021114-67c6ae64274f // indirect
	github.com/prometheus/client_golang v1.1.0 // indirect
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4 // indirect
	github.com/rasky/go-xdr v0.0.0-20170124162913-1a41d1a06c93 // indirect
	github.com/rockbears/log v0.3.0
	github.com/rubenv/sql-migrate v0.0.0-20160620083229-6f4757563362
	github.com/satori/go.uuid v1.2.0
	github.com/sguiheux/go-coverage v0.0.0-20190710153556-287b082a7197
	github.com/shirou/gopsutil v0.0.0-20170406131756-e49a95f3d5f8
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/afero v1.2.2
	github.com/spf13/cast v1.3.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/viper v1.7.0
	github.com/streadway/amqp v0.0.0-20180528204448-e5adc2ada8b8
	github.com/stretchr/testify v1.6.1
	github.com/studio-b12/gowebdav v0.0.0-20200303150724-9380631c29a1
	github.com/tevino/abool v0.0.0-20170917061928-9b9efcf221b5
	github.com/ugorji/go v1.1.7 // indirect
	github.com/ulikunitz/xz v0.5.4 // indirect
	github.com/urfave/cli v1.20.0
	github.com/vmware/go-nfs-client v0.0.0-20190605212624-d43b92724c1b
	github.com/vmware/govmomi v0.23.0
	github.com/whilp/git-urls v0.0.0-20160530060445-31bac0d230fa
	github.com/xanzy/go-gitlab v0.15.0
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	github.com/yesnault/go-toml v0.0.0-20191205182532-f5ef6cee7945
	github.com/yuin/gluare v0.0.0-20170607022532-d7c94f1a80ed
	github.com/yuin/gopher-lua v0.0.0-20170901023928-8c2befcd3908
	github.com/ziutek/mymysql v1.5.4 // indirect
	go.opencensus.io v0.22.0
	golang.org/x/crypto v0.0.0-20200709230013-948cd5f35899
	golang.org/x/net v0.0.0-20200528225125-3c3fba18258b
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sys v0.0.0-20210124154548-22da62e12c0c
	golang.org/x/text v0.3.2
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4
	google.golang.org/grpc v1.23.0
	gopkg.in/AlecAivazis/survey.v1 v1.7.1
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/validator.v8 v8.18.2 // indirect
	gopkg.in/gorp.v1 v1.7.1 // indirect
	gopkg.in/h2non/gock.v1 v1.0.14
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ldap.v2 v2.5.1
	gopkg.in/olivere/elastic.v6 v6.2.17
	gopkg.in/spacemonkeygo/httpsig.v0 v0.0.0-20170228231032-6732593ec966
	gopkg.in/square/go-jose.v2 v2.3.1
	gopkg.in/yaml.v2 v2.3.0
	gotest.tools v2.1.0+incompatible // indirect
	k8s.io/api v0.0.0-20181204000039-89a74a8d264d
	k8s.io/apimachinery v0.0.0-20190223094358-dcb391cde5ca
	k8s.io/client-go v10.0.0+incompatible
	k8s.io/klog v0.2.0 // indirect
	sigs.k8s.io/yaml v1.1.0 // indirect
)

replace github.com/alecthomas/jsonschema => github.com/sguiheux/jsonschema v0.2.0

replace github.com/go-gorp/gorp => github.com/yesnault/gorp v2.0.1-0.20200325154225-2dc6d8c2da37+incompatible

replace github.com/docker/docker => github.com/docker/engine v0.0.0-20180816081446-320063a2ad06

replace github.com/ovh/cds/sdk/interpolate => ./sdk/interpolate

replace github.com/keybase/go-crypto => github.com/Alkorin/crypto v0.0.0-20190802123352-5ea49ae5e604

replace github.com/ovh/cds/tools/smtpmock => ./tools/smtpmock

replace github.com/keybase/go-keychain => github.com/yesnault/go-keychain v0.0.0-20190829085436-f78f7ae28786

replace github.com/coreos/etcd v3.3.10+incompatible => github.com/coreos/etcd v3.3.13+incompatible

replace github.com/jordan-wright/email => github.com/yesnault/email v0.0.0-20201006155628-d88bfe11e7f1
