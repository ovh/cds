module github.com/ovh/cds/contrib/grpcplugins/action/kafka-publish

replace github.com/ovh/cds => ../../../../

replace github.com/go-gorp/gorp => github.com/yesnault/gorp v2.0.1-0.20200325154225-2dc6d8c2da37+incompatible

go 1.16

require (
	github.com/Shopify/sarama v1.27.0
	github.com/bgentry/speakeasy v0.1.0
	github.com/fsamin/go-shredder v0.0.0-20180118184739-b2488aedb5be
	github.com/golang/protobuf v1.3.2
	github.com/ovh/cds v0.0.0-00010101000000-000000000000
	github.com/phayes/permbits v0.0.0-20190612203442-39d7c581d2ee
	github.com/stretchr/testify v1.6.1
	gopkg.in/urfave/cli.v1 v1.20.0
)
