package kafka

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	cluster "github.com/bsm/sarama-cluster"
	"github.com/mitchellh/mapstructure"
	"github.com/ovh/venom"
	"github.com/ovh/venom/executors"
)

// Name of executor
const Name = "kafka"

// New returns a new Executor
func New() venom.Executor {
	return &Executor{}
}

//Message represents the object sended or received from kafka
type Message struct {
	Topic string
	Value string
}

//MessageJSON represents the object sended or received from kafka
type MessageJSON struct {
	Topic string
	Value interface{}
}

// Executor represents a Test Exec
type Executor struct {
	Addrs              []string `json:"addrs,omitempty" yaml:"addrs,omitempty"`
	WithTLS            bool     `json:"with_tls,omitempty" yaml:"withTLS,omitempty"`
	WithSASL           bool     `json:"with_sasl,omitempty" yaml:"withSASL,omitempty"`
	WithSASLHandshaked bool     `json:"with_sasl_handshaked,omitempty" yaml:"withSASLHandshaked,omitempty"`
	User               string   `json:"user,omitempty" yaml:"user,omitempty"`
	Password           string   `json:"password,omitempty" yaml:"password,omitempty"`

	//ClientType must be "consumer" or "producer"
	ClientType string `json:"client_type,omitempty" yaml:"clientType,omitempty"`

	//Used when ClientType is consumer
	GroupID string   `json:"group_id,omitempty" yaml:"groupID,omitempty"`
	Topics  []string `json:"topics,omitempty" yaml:"topics,omitempty"`
	//Represents the timeout for reading messages. In Milliseconds. Default 5000
	Timeout int64 `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	//Represents the limit of message will be read. After limit, consumer stop read message
	MessageLimit int `json:"message_limit,omitempty" yaml:"messageLimit,omitempty"`
	//InitialOffset represents the initial offset for the consumer. Possible value : newest, oldest. default: newest
	InitialOffset string `json:"initial_offset,omitempty" yaml:"initialOffset,omitempty"`
	//MarkOffset allows to mark offset when consuming message
	MarkOffset bool `json:"mark_offset,omitempty" yaml:"markOffset,omitempty"`

	//Used when ClientType is producer
	//Messages represents the message sended by producer
	Messages []Message `json:"messages,omitempty" yaml:"messages,omitempty"`

	//MessagesFile represents the messages into the file sended by producer (messages field would be ignored)
	MessagesFile string `json:"messages_file,omitempty" yaml:"messages_file,omitempty"`
}

// Result represents a step result.
type Result struct {
	Executor     Executor      `json:"executor,omitempty" yaml:"executor,omitempty"`
	TimeSeconds  float64       `json:"timeSeconds,omitempty" yaml:"timeSeconds,omitempty"`
	TimeHuman    string        `json:"timeHuman,omitempty" yaml:"timeHuman,omitempty"`
	Messages     []Message     `json:"messages,omitempty" yaml:"messages,omitempty"`
	MessagesJSON []interface{} `json:"messagesJSON,omitempty" yaml:"messagesJSON,omitempty"`
	Err          string        `json:"error" yaml:"error"`
}

// ZeroValueResult return an empty implemtation of this executor result
func (Executor) ZeroValueResult() venom.ExecutorResult {
	r, _ := executors.Dump(Result{})
	return r
}

// GetDefaultAssertions return default assertions for type exec
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []string{"result.err ShouldBeEmpty"}}
}

// Run execute TestStep of type exec
func (Executor) Run(testCaseContext venom.TestCaseContext, l venom.Logger, step venom.TestStep, workdir string) (venom.ExecutorResult, error) {
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}
	start := time.Now()

	result := Result{Executor: e}

	if e.Timeout == 0 {
		e.Timeout = 5000
	}
	if e.ClientType == "producer" {
		err := e.produceMessages(workdir)
		if err != nil {
			result.Err = err.Error()
		}
	} else if e.ClientType == "consumer" {
		var err error
		result.Messages, result.MessagesJSON, err = e.consumeMessages(l)
		if err != nil {
			result.Err = err.Error()
		}
	} else {
		return nil, fmt.Errorf("type must be a consumer or a producer")
	}

	elapsed := time.Since(start)
	result.TimeSeconds = elapsed.Seconds()
	result.TimeHuman = elapsed.String()
	result.Executor.Password = "****hidden****" // do not output password

	return executors.Dump(result)
}

func (e Executor) produceMessages(workdir string) error {

	if len(e.Messages) == 0 && e.MessagesFile == "" {
		return fmt.Errorf("At least messages or messagesFile property must be setted")
	}

	producerCfg := sarama.NewConfig()
	producerCfg.Net.TLS.Enable = e.WithTLS // Enable TLS anyway
	producerCfg.Net.SASL.Enable = e.WithSASL
	producerCfg.Net.SASL.User = e.User
	producerCfg.Net.SASL.Password = e.Password
	producerCfg.Producer.RequiredAcks = sarama.WaitForLocal
	producerCfg.Producer.Retry.Max = 10
	producerCfg.Net.DialTimeout = 5 * time.Second
	producerCfg.Producer.Return.Successes = true
	producerCfg.Producer.Return.Errors = true
	sp, err := sarama.NewSyncProducer(e.Addrs, producerCfg)
	if err != nil {
		return err
	}
	defer sp.Close()

	messages := []*sarama.ProducerMessage{}

	if e.MessagesFile != "" {
		path := filepath.Join(workdir, string(e.MessagesFile))
		if _, err = os.Stat(path); err == nil {
			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			messages := []Message{}
			err = json.Unmarshal(content, &messages)
			if err != nil {
				return err
			}
			e.Messages = messages
		} else {
			return err
		}
	}

	for i := range e.Messages {
		message := e.Messages[i]
		messages = append(messages, &sarama.ProducerMessage{
			Topic: message.Topic,
			Value: sarama.ByteEncoder([]byte(message.Value)),
		})
	}
	return sp.SendMessages(messages)
}

func (e Executor) consumeMessages(l venom.Logger) ([]Message, []interface{}, error) {
	if len(e.Topics) == 0 {
		return nil, nil, fmt.Errorf("You must provide topics")
	}

	consumerConfig := cluster.NewConfig()
	consumerConfig.Net.TLS.Enable = e.WithTLS
	consumerConfig.Net.SASL.Enable = e.WithSASL
	consumerConfig.Net.SASL.User = e.User
	consumerConfig.Net.SASL.Password = e.Password

	if strings.TrimSpace(e.InitialOffset) == "oldest" {
		consumerConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	}

	consumer, err := cluster.NewConsumer(e.Addrs, e.GroupID, e.Topics, consumerConfig)
	if err != nil {
		return nil, nil, err
	}
	defer consumer.Close()

	timeout := time.Duration(e.Timeout) * time.Millisecond

	messages := []Message{}
	messagesJSON := []interface{}{}

reading:
	for {
		select {
		case message := <-consumer.Messages():
			messages = append(messages, Message{
				Topic: message.Topic,
				Value: string(message.Value),
			})
			messageJSONArray := []MessageJSON{}
			if err := json.Unmarshal(message.Value, &messageJSONArray); err != nil {
				messageJSONMap := map[string]interface{}{}
				if err2 := json.Unmarshal(message.Value, &messageJSONMap); err2 == nil {
					messagesJSON = append(messagesJSON, MessageJSON{
						Topic: message.Topic,
						Value: messageJSONMap,
					})
				} else {
					messagesJSON = append(messagesJSON, MessageJSON{
						Topic: message.Topic,
						Value: string(message.Value),
					})
				}
			} else {
				messagesJSON = append(messagesJSON, MessageJSON{
					Topic: message.Topic,
					Value: messageJSONArray,
				})
			}
			if e.MarkOffset {
				consumer.MarkOffset(message, "")
			}
			if e.MessageLimit > 0 && len(messages) >= e.MessageLimit {
				break reading
			}
		case <-time.After(timeout):
			l.Infof("Timeout reached")
			break reading
		}
	}

	return messages, messagesJSON, nil

}
