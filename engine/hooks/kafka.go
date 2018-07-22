package hooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	"github.com/fsamin/go-dump"
	"gopkg.in/bsm/sarama-cluster.v2"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) saveKafkaExecution(t *sdk.Task, error string, nbError int64) {
	exec := &sdk.TaskExecution{
		Timestamp:           time.Now().UnixNano(),
		Type:                t.Type,
		UUID:                t.UUID,
		Config:              t.Config,
		Status:              TaskExecutionDone,
		LastError:           error,
		NbErrors:            nbError,
		ProcessingTimestamp: time.Now().UnixNano(),
	}
	s.Dao.SaveTaskExecution(exec)
}

func (s *Service) startKafkaHook(t *sdk.Task) error {
	var kafkaPlatform, kafkaUser, projectKey, topic string
	for k, v := range t.Config {
		switch k {
		case sdk.HookModelPlatform:
			kafkaPlatform = v.Value
		case sdk.KafkaHookModelTopic:
			topic = v.Value
		case sdk.HookConfigProject:
			projectKey = v.Value
		}
	}
	pf, err := s.Client.ProjectPlatformGet(projectKey, kafkaPlatform, true)
	if err != nil {
		s.stopTask(t)
		return sdk.WrapError(err, "startTask> Cannot get kafka configuration for %s/%s", projectKey, kafkaPlatform)
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
		}

	}

	var config = sarama.NewConfig()
	config.Net.TLS.Enable = true
	config.Net.SASL.Enable = true
	config.Net.SASL.User = kafkaUser
	config.Net.SASL.Password = password
	config.Version = sarama.V0_10_0_1

	config.ClientID = kafkaUser

	clusterConfig := cluster.NewConfig()
	clusterConfig.Config = *config
	clusterConfig.Consumer.Return.Errors = true

	var consumerGroup = fmt.Sprintf("%s.%s", kafkaUser, t.UUID)
	var errConsumer error
	consumer, errConsumer := cluster.NewConsumer(
		strings.Split(broker, ","),
		consumerGroup,
		[]string{topic},
		clusterConfig)

	if errConsumer != nil {
		s.stopTask(t)
		return fmt.Errorf("startKafkaHook>Error creating consumer: (%s %s %s %s): %v", broker, consumerGroup, topic, kafkaUser, errConsumer)
	}

	vConsumer := t.Config[sdk.KafkaHookModelConsumerGroup]
	vConsumer.Value = consumerGroup
	t.Config[sdk.KafkaHookModelConsumerGroup] = vConsumer
	s.saveKafkaExecution(t, "", 0)

	// Consume errors
	go func() {
		for err := range consumer.Errors() {
			s.saveKafkaExecution(t, err.Error(), 1)
		}
	}()

	// consume message
	go func() {
		for msg := range consumer.Messages() {
			exec := sdk.TaskExecution{
				ProcessingTimestamp: time.Now().UnixNano(),
				Status:              TaskExecutionDoing,
				Config:              t.Config,
				Type:                TypeKafka,
				UUID:                t.UUID,
				Timestamp:           time.Now().UnixNano(),
				Kafka:               &sdk.KafkaTaskExecution{Message: msg.Value},
			}
			s.Dao.SaveTaskExecution(&exec)
			s.Dao.EnqueueTaskExecution(&exec)
			consumer.MarkOffset(msg, "delivered")
		}
	}()

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
	e := dump.NewDefaultEncoder(new(bytes.Buffer))
	e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
	e.ExtraFields.DetailedMap = false
	e.ExtraFields.DetailedStruct = false
	e.ExtraFields.DeepJSON = true
	e.ExtraFields.Len = false
	e.ExtraFields.Type = false
	m, err := e.ToStringMap(bodyJSON)
	if err != nil {
		return nil, sdk.WrapError(err, "Hooks.doKafkaTaskExecution> Unable to dump body %s", t.WebHook.RequestBody)
	}
	h.Payload = m

	return &h, nil
}
