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

	id := e.JobRunID // job v2 as code
	if id == "" {
		id = strconv.FormatInt(e.ID, 10) // job v1
	}

	_, err := s.esClient.IndexDocWithExternalVersion(ctx, s.Cfg.ElasticSearch.IndexJobSummary, id, jobSummaryVersion(*e), e)
	if errors.Is(err, errVersionConflict) {
		log.Info(ctx, "skipping job summary %s: the index already holds a later state of the job", id)
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "unable to index document %+v", e)
	}
	return nil
}

// jobSummaryVersionBase keeps every version above the internal version of the documents indexed
// without one, which counts their writes.
const jobSummaryVersionBase = 1_000_000

// jobSummaryVersion ranks the state of the job: queued, started, ended. The events of a job can be
// consumed out of order, as they are spread over the partitions of the topic, and a late event must
// not take its document back to an earlier state.
func jobSummaryVersion(e sdk.EventJobSummary) int64 {
	switch {
	case e.Ended != nil:
		return jobSummaryVersionBase + 3
	case e.Started != nil:
		return jobSummaryVersionBase + 2
	default:
		return jobSummaryVersionBase + 1
	}
}
