package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/Shopify/sarama"

	"github.com/ovh/cds/contrib/plugins/plugin-kafka-publish/kafkapublisher"
)

//Wait for ACK to CDS through kafka. Entrypoint is the actionID from the context file. After a fimeout (seconds) it will return an error
func ackFromKafka(kafka, topic, group, key string, timeout time.Duration, actionID int64) (*kafkapublisher.Ack, error) {
	//Create a new client
	var config = sarama.NewConfig()
	// Set key as the client id for authentication
	config.ClientID = key
	client, err := sarama.NewClient([]string{kafka}, config)
	if err != nil {
		return nil, err
	}

	// Create an offsetManager
	offsetManager, err := sarama.NewOffsetManagerFromClient(group, client)
	if err != nil {
		return nil, err
	}

	// Create a client
	consumer, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		return nil, err
	}

	Logf("Waiting ACK on %s on topic %s...\n", kafka, topic)

	// Create the message chan, that will receive the queue
	messagesChan := make(chan []byte)
	// Create the error chan, that will receive the queue
	errorsChan := make(chan error)
	//Create the timout chan, that will receive data after a timeout
	timeoutChan := make(chan bool)

	// read the number of partition for the given topic
	partitions, err := consumer.Partitions(topic)
	if err != nil {
		return nil, err
	}

	// Create a consumer for each partition
	if len(partitions) > 1 {
		return nil, fmt.Errorf("Multiple partition not supported")
	}
	p := partitions[0]
	partitionOffsetManager, err := offsetManager.ManagePartition(topic, p)
	if err != nil {
		return nil, err
	}
	defer partitionOffsetManager.AsyncClose()

	// Start a consumer at next offset
	offset, _ := partitionOffsetManager.NextOffset()
	partitionConsumer, err := consumer.ConsumePartition(topic, p, offset)
	if err != nil {
		return nil, err
	}
	defer partitionConsumer.AsyncClose()

	// Wait for timeout
	go func() {
		time.Sleep(timeout)
		timeoutChan <- true
	}()

	// Asynchronously handle message
	go consumptionHandler(partitionConsumer, topic, partitionOffsetManager, messagesChan, errorsChan)

	// Trap SIGINT to trigger a shutdown.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	// Main routine, we will exit on error||timeout||ack
	for {
		select {
		case msg := <-messagesChan:
			ack := &kafkapublisher.Ack{}
			if err := json.Unmarshal(msg, ack); err != nil {
				return nil, err
			}
			if ack.Context.ActionID != actionID {
				continue
			}
			//Yep we receive the right ack !
			return ack, nil
		case err := <-errorsChan:
			return nil, err
		case <-signals:
			return nil, fmt.Errorf("Interrupted")
		case <-timeoutChan:
			return nil, fmt.Errorf("Timeout exceeded")
		}
	}
}
