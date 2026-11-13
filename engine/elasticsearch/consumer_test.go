package elasticsearch

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/ovh/cds/engine/elasticsearch/mock_elasticsearch"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/engine/test/config"
	"github.com/rockbears/log"
	"go.uber.org/mock/gomock"

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
	mockESClient.EXPECT().IndexDocWithExternalVersion(gomock.Any(), "IndexJobSummary", gomock.Any(), gomock.Any(), gomock.Any()).MinTimes(1)

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

func Test_processEventJobSummary(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	now := time.Now()

	tests := []struct {
		name    string
		event   sdk.EventJobSummary
		id      string
		version int64
	}{
		{name: "v1 queued", event: sdk.EventJobSummary{ID: 42, Created: &now}, id: "42", version: jobSummaryVersionBase + 1},
		{name: "v1 started", event: sdk.EventJobSummary{ID: 42, Created: &now, Started: &now}, id: "42", version: jobSummaryVersionBase + 2},
		{name: "v1 ended", event: sdk.EventJobSummary{ID: 42, Created: &now, Started: &now, Ended: &now}, id: "42", version: jobSummaryVersionBase + 3},
		{name: "v2 queued", event: sdk.EventJobSummary{JobRunID: "run-job-uuid", Created: &now}, id: "run-job-uuid", version: jobSummaryVersionBase + 1},
		{name: "v2 ended", event: sdk.EventJobSummary{JobRunID: "run-job-uuid", Created: &now, Started: &now, Ended: &now}, id: "run-job-uuid", version: jobSummaryVersionBase + 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockESClient := mock_elasticsearch.NewMockESClient(ctrl)
			s := Service{esClient: mockESClient}
			s.Cfg.ElasticSearch.IndexJobSummary = "IndexJobSummary"

			mockESClient.EXPECT().IndexDocWithExternalVersion(gomock.Any(), "IndexJobSummary", tt.id, tt.version, gomock.Any())
			require.NoError(t, s.processEventJobSummary(&tt.event))
		})
	}

	t.Run("an event older than the stored document is skipped", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockESClient := mock_elasticsearch.NewMockESClient(ctrl)
		s := Service{esClient: mockESClient}

		mockESClient.EXPECT().IndexDocWithExternalVersion(gomock.Any(), gomock.Any(), "42", int64(jobSummaryVersionBase+2), gomock.Any()).Return(nil, errVersionConflict)
		require.NoError(t, s.processEventJobSummary(&sdk.EventJobSummary{ID: 42, Created: &now, Started: &now}))
	})

	t.Run("other indexing errors are returned", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockESClient := mock_elasticsearch.NewMockESClient(ctrl)
		s := Service{esClient: mockESClient}

		mockESClient.EXPECT().IndexDocWithExternalVersion(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("unreachable"))
		require.Error(t, s.processEventJobSummary(&sdk.EventJobSummary{ID: 42, Created: &now}))
	})
}
