package elasticsearch

import (
	"context"
	"strconv"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/event"
	"github.com/pkg/errors"
	"github.com/rockbears/log"
)

func (s *Service) consumeKafka(ctx context.Context, cfg event.KafkaConsumerConfig) error {
	if s.Cfg.ElasticSearch.IndexJobSummary == "" {
		return errors.New("unable to start kafka consumer. missing index configuration")
	}

	log.Info(ctx, "starting kafka consumer")
	errorFunc := func(format string, args ...interface{}) {
		log.Error(ctx, format, args...)
	}
	return event.ConsumeKafka(ctx, cfg, sdk.EventJobSummary{}, s.processEventJobSummary, errorFunc)
}

func (s *Service) processEventJobSummary(i interface{}) error {
	log.Info(context.Background(), "processing document %+v", i)
	ctx := context.Background()

	e, ok := i.(*sdk.EventJobSummary)
	if !ok {
		return errors.Errorf("unsupported type %T", i)
	}

	_, err := esClient.Index().Index(s.Cfg.ElasticSearch.IndexJobSummary).Type("cds_job").Id(strconv.FormatInt(e.ID, 10)).BodyJson(e).Do(ctx)
	if err != nil {
		return errors.Wrapf(err, "unable to index document %+v", e)
	}
	return nil
}
