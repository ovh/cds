module github.com/ovh/cds/contrib/grpcplugins/action/kafka-publish

replace github.com/ovh/cds => ../../../../

go 1.13

require (
	github.com/Shopify/sarama v1.24.1
	github.com/bgentry/speakeasy v0.1.0
	github.com/facebookgo/ensure v0.0.0-20160127193407-b4ab57deab51 // indirect
	github.com/facebookgo/freeport v0.0.0-20150612182905-d4adf43b75b9 // indirect
	github.com/facebookgo/stack v0.0.0-20160209184415-751773369052 // indirect
	github.com/facebookgo/subset v0.0.0-20150612182917-8dac2c3c4870 // indirect
	github.com/fsamin/go-shredder v0.0.0-20180118184739-b2488aedb5be
	github.com/golang/protobuf v1.3.2
	github.com/klauspost/cpuid v1.2.1 // indirect
	github.com/ovh/cds v0.0.0-00010101000000-000000000000
	github.com/phayes/permbits v0.0.0-20190612203442-39d7c581d2ee
	github.com/stretchr/testify v1.5.1
	gopkg.in/urfave/cli.v1 v1.20.0
)
