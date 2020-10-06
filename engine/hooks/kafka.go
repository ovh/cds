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
	var kafkaIntegration, projectKey, topic string
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

	var config = sarama.NewConfig()
	if _, ok := pf.Config["disableTLS"]; ok && pf.Config["disableTLS"].Value == "true" {
		config.Net.TLS.Enable = false
	} else {
		config.Net.TLS.Enable = true
	}
	if _, ok := pf.Config["disableSASL"]; ok && pf.Config["disableSASL"].Value == "true" {
		config.Net.SASL.Enable = false
	} else {
		config.Net.SASL.Enable = true
		config.Net.SASL.User = pf.Config["username"].Value
		config.Net.SASL.Password = pf.Config["password"].Value
	}
	if _, ok := pf.Config["user"]; ok && pf.Config["user"].Value != "" {
		config.ClientID = pf.Config["user"].Value
	} else {
		config.ClientID = "cds"
	}

	config.Consumer.Return.Errors = true
	if v, ok := pf.Config["version"]; ok && v.Value != "" {
		kafkaVersion, err := sarama.ParseKafkaVersion(pf.Config["version"].Value)
		if err != nil {
			return fmt.Errorf("error parsing Kafka version %v err:%s", kafkaVersion, err)
		}
		config.Version = kafkaVersion
	} else {
		config.Version = sarama.V0_10_2_0
	}

	var group = fmt.Sprintf("%s.%s", config.Net.SASL.User, t.UUID)
	consumerGroup, err := sarama.NewConsumerGroup([]string{pf.Config["broker url"].Value}, group, config)
	if err != nil {
		_ = s.stopTask(ctx, t)
		return fmt.Errorf("startKafkaHook>Error creating consumer: (%s %s %s %s): %v", pf.Config["broker url"].Value, consumerGroup, topic, config.Net.SASL.User, err)
	}

	// Track errors
	go func() {
		for err := range consumerGroup.Errors() {
			s.saveKafkaExecution(t, err.Error(), 1)
		}
	}()

	h := &handler{
		task: t,
		dao:  &s.Dao,
	}

	s.GoRoutines.Exec(context.Background(), "kafka-consume-"+topic, func(ctx context.Context) {
		atomic.AddInt64(&nbKafkaConsumers, 1)
		defer atomic.AddInt64(&nbKafkaConsumers, -1)
		for {
			if err := consumerGroup.Consume(ctx, []string{topic}, h); err != nil {
				log.Error(ctx, "error on consume:%s", err)
			}
		}
	})

	return nil
}

// handler represents a Sarama consumer group consumer
type handler struct {
	task *sdk.Task
	dao  *dao
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (h *handler) Setup(s sarama.ConsumerGroupSession) error {
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
