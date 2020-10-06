package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/Shopify/sarama"
	"github.com/fsamin/go-dump"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var nbKafkaConsumers int64

func (s *Service) saveKafkaExecution(t *sdk.Task, error string, nbError int64) {
	exec := &sdk.TaskExecution{
		Timestamp: time.Now().UnixNano(),
		Type:      t.Type,
		UUID:      t.UUID,
		Config:    t.Config,
		Status:    TaskExecutionDone,
		LastError: error,
		NbErrors:  nbError,
	}
	s.Dao.SaveTaskExecution(exec)
}

func (s *Service) startKafkaHook(ctx context.Context, t *sdk.Task) error {
	var kafkaIntegration, kafkaUser, kafkaVersion, projectKey, topic string
	for k, v := range t.Config {
		switch k {
		case sdk.HookModelIntegration:
			kafkaIntegration = v.Value
		case sdk.KafkaHookModelTopic:
			topic = v.Value
		case sdk.HookConfigProject:
			projectKey = v.Value
		}
	}
	pf, err := s.Client.ProjectIntegrationGet(projectKey, kafkaIntegration, true)
	if err != nil {
		_ = s.stopTask(ctx, t)
		return sdk.WrapError(err, "Cannot get kafka configuration for %s/%s", projectKey, kafkaIntegration)
	}

	var password, broker string
	for k, v := range pf.Config {
		switch k {
		case "password":
			password = v.Value
		case "broker url":
			broker = v.Value
		case "username":
			kafkaUser = v.Value
		case "version":
			kafkaVersion = v.Value
		}
	}

	var config = sarama.NewConfig()
	config.Net.TLS.Enable = true
	config.Net.SASL.Enable = true
	config.Net.SASL.User = kafkaUser
	config.Net.SASL.Password = password
	config.ClientID = kafkaUser
	config.Consumer.Return.Errors = true

	if kafkaVersion != "" {
		kafkaVersion, err := sarama.ParseKafkaVersion(kafkaVersion)
		if err != nil {
			return fmt.Errorf("error parsing Kafka version %v err:%s", kafkaVersion, err)
		}
		config.Version = kafkaVersion
	} else {
		config.Version = sarama.V0_10_0_1
	}

	var group = fmt.Sprintf("%s.%s", kafkaUser, t.UUID)
	consumerGroup, err := sarama.NewConsumerGroup([]string{broker}, group, config)
	if err != nil {
		_ = s.stopTask(ctx, t)
		return fmt.Errorf("startKafkaHook>Error creating consumer: (%s %s %s %s): %v", broker, consumerGroup, topic, kafkaUser, err)
	}

	// Track errors
	go func() {
		for err := range consumerGroup.Errors() {
			s.saveKafkaExecution(t, err.Error(), 1)
		}
	}()

	h := handler{
		task: t,
		dao:  &s.Dao,
	}

	go func() {
		atomic.AddInt64(&nbKafkaConsumers, 1)
		defer atomic.AddInt64(&nbKafkaConsumers, -1)
		for {
			if err := consumerGroup.Consume(ctx, []string{topic}, &h); err != nil {
				log.Error(ctx, "error on consume:%s", err)
			}
		}
	}()
	<-h.ready // Await till the consumer has been set up

	return nil
}

// handler represents a Sarama consumer group consumer
type handler struct {
	ready chan bool
	task  *sdk.Task
	dao   *dao
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (h *handler) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(h.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (h *handler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (h *handler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		exec := sdk.TaskExecution{
			Status:    TaskExecutionScheduled,
			Config:    h.task.Config,
			Type:      TypeKafka,
			UUID:      h.task.UUID,
			Timestamp: time.Now().UnixNano(),
			Kafka:     &sdk.KafkaTaskExecution{Message: message.Value},
		}
		h.dao.SaveTaskExecution(&exec)
		session.MarkMessage(message, "delivered")
	}
	return nil
}

func (s *Service) doKafkaTaskExecution(t *sdk.TaskExecution) (*sdk.WorkflowNodeRunHookEvent, error) {
	log.Debug("Hooks> Processing kafka %s %s", t.UUID, t.Type)

	// Prepare a struct to send to CDS API
	h := sdk.WorkflowNodeRunHookEvent{
		WorkflowNodeHookUUID: t.UUID,
		Payload:              map[string]string{},
	}

	var bodyJSON interface{}

	//Try to parse the body as an array
	bodyJSONArray := []interface{}{}
	if err := json.Unmarshal(t.Kafka.Message, &bodyJSONArray); err != nil {
		//Try to parse the body as a map
		bodyJSONMap := map[string]interface{}{}
		if err2 := json.Unmarshal(t.Kafka.Message, &bodyJSONMap); err2 == nil {
			bodyJSON = bodyJSONMap
		}
	} else {
		bodyJSON = bodyJSONArray
	}

	//Go Dump
	e := dump.NewDefaultEncoder()
	e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
	e.ExtraFields.DetailedMap = false
	e.ExtraFields.DetailedStruct = false
	e.ExtraFields.DeepJSON = true
	e.ExtraFields.Len = false
	e.ExtraFields.Type = false
	m, err := e.ToStringMap(bodyJSON)
	if err != nil {
		return nil, sdk.WrapError(err, "Unable to dump body %s", t.WebHook.RequestBody)
	}
	h.Payload = m
	h.Payload["payload"] = string(t.Kafka.Message)

	return &h, nil
}
