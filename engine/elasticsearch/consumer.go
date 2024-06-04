package elasticsearch

import (
	"context"
	"strconv"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/event"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/pkg/errors"
	"github.com/rockbears/log"
)

func (s *Service) consumeKafka(ctx context.Context, cfg event.KafkaConsumerConfig) error {
	if s.Cfg.ElasticSearch.IndexJobSummary == "" {
		return errors.New("unable to start kafka consumer. missing index configuration")
	}

	ctx = context.WithValue(ctx, cdslog.KafkaBroker, cfg.BrokerAddresses)
	ctx = context.WithValue(ctx, cdslog.KafkaTopic, cfg.Topic)

	log.Info(ctx, "starting kafka consumer {broker: %s, user:%s ,topic: %s, group: %s}", cfg.BrokerAddresses, cfg.User, cfg.Topic, cfg.ConsumerGroup)
	errorFunc := func(format string, args ...interface{}) {
		log.Error(ctx, format, args...)
	}
	return event.ConsumeKafka(ctx, s.GoRoutines, cfg, sdk.EventJobSummary{}, s.processEventJobSummary, errorFunc)
}

func (s *Service) processEventJobSummary(i interface{}) error {
	ctx := context.Background()
	ctx = context.WithValue(ctx, cdslog.KafkaBroker, s.Cfg.EventBus.JobSummaryKafka.BrokerAddresses)
	ctx = context.WithValue(ctx, cdslog.KafkaTopic, s.Cfg.EventBus.JobSummaryKafka.Topic)
	log.Info(ctx, "processing document %+v", i)

	e, ok := i.(*sdk.EventJobSummary)
	if !ok {
		return errors.Errorf("unsupported type %T", i)
	}

	// job v2 as code
	if e.JobRunID != "" {
		_, err := s.esClient.IndexDoc(ctx, s.Cfg.ElasticSearch.IndexJobSummary, "cds_job", e.JobRunID, e)
		if err != nil {
			return errors.Wrapf(err, "unable to index document ascode v2 %+v", e)
		}
		return nil
	}

	// job v1
	_, err := s.esClient.IndexDoc(ctx, s.Cfg.ElasticSearch.IndexJobSummary, "cds_job", strconv.FormatInt(e.ID, 10), e)
	if err != nil {
		return errors.Wrapf(err, "unable to index document %+v", e)
	}
	return nil
}
