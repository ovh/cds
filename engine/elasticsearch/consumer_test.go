package elasticsearch

import (
	"context"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/engine/elasticsearch/mock_elasticsearch"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/engine/test/config"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/event"
	"github.com/stretchr/testify/require"
)

func Test_consumeKafka(t *testing.T) {
	if !config.ConfExist(t, sdk.TypeElasticsearch) {
		t.SkipNow()
	}
	log.Factory = log.NewTestingWrapper(t)
	var s = Service{}
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()
	s.Common.GoRoutines = sdk.NewGoRoutines(ctx)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockESClient := mock_elasticsearch.NewMockESClient(ctrl)
	s.esClient = mockESClient
	mockESClient.EXPECT().IndexDoc(gomock.Any(), "IndexJobSummary", gomock.Any(), gomock.Any(), gomock.Any()).MinTimes(1)

	cfg := test.LoadTestingConf(t, sdk.TypeElasticsearch)
	if cfg["broker"] == "" {
		t.SkipNow()
	}

	s.Cfg.ElasticSearch.IndexJobSummary = "IndexJobSummary"
	var offset = sarama.OffsetOldest
	s.Cfg.EventBus.JobSummaryKafka = event.KafkaConsumerConfig{
		ConsumerGroup: cfg["consumerGroup"],
		InitialOffset: &offset,
		KafkaConfig: event.KafkaConfig{
			Enabled:         true,
			BrokerAddresses: cfg["broker"],
			User:            cfg["user"],
			Password:        cfg["password"],
			Topic:           cfg["topic"],
		},
	}
	require.NoError(t, s.consumeKafka(ctx, s.Cfg.EventBus.JobSummaryKafka))
	<-ctx.Done()
}
