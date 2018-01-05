package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/Shopify/sarama"
	"github.com/fsamin/go-shredder"

	"github.com/ovh/cds/contrib/plugins/plugin-kafka-publish/kafkapublisher"
)

//Wait for ACK to CDS through kafka. Entrypoint is the actionID from the context file. After a fimeout (seconds) it will return an error
func ackFromKafka(kafka, topic, group, user, password string, timeout time.Duration, actionID int64) (*kafkapublisher.Ack, error) {
	//Create a new client
	var config = sarama.NewConfig()
	config.Net.TLS.Enable = true
	config.Net.SASL.Enable = true
	config.Net.SASL.User = user
	config.Net.SASL.Password = password
	config.ClientID = user
	config.Producer.Return.Successes = true

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
	defer partitionOffsetManager.Close()

	// Start a consumer at next offset
	offset, _ := partitionOffsetManager.NextOffset()
	partitionConsumer, err := consumer.ConsumePartition(topic, p, offset)
	if err != nil {
		return nil, err
	}
	defer partitionConsumer.Close()

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

	//This chunks list is for artifacts
	chunks := shredder.Chunks{}

	// Main routine, we will exit on error||timeout||ack
	for {
		select {
		case msg := <-messagesChan:
			//If we recieve a "Chunk" Message
			if kafkapublisher.IsChunk(msg) {
				c, err := kafkapublisher.ReadBytes(msg)
				if err != nil {
					fmt.Printf("Unable to read bytes : %s\n", err)
					continue
				}
				fmt.Println("Chunk received")
				chunks = append(chunks, *c)

				allChunks := shredder.Filter(chunks)
				cs := allChunks[c.Ctx.UUID]

				//If we received all chunks for a file, let save it on disk
				if cs.Completed() {
					aes, err := getAESEncryptionOptions(password)
					if err != nil {
						fmt.Printf("Error: %s\n", err)
						continue
					}
					var opts = &shredder.Opts{
						ChunkSize:     512 * 1024,
						AESEncryption: aes,
					}

					content, err := shredder.Reassemble(cs, opts)
					if err != nil {
						fmt.Printf("Error: %s\n", err)
						continue
					}

					filename, data, err := content.File()
					if err != nil {
						fmt.Printf("Error: %s\n", err)
						continue
					}

					fmt.Printf("Receiving file : %s\n", filename)

					if err := fileHandler(nil, filename, data); err != nil {
						fmt.Printf("Error: %s\n", err)
						continue
					}
					//File has been processed, remove data from memory
					chunks.Delete(*c)
				}
				continue
			} else {
				ack := &kafkapublisher.Ack{}
				if err := json.Unmarshal(msg, ack); err != nil {
					fmt.Printf("Unable to parse ack: %s\n", err)
					continue
				}
				if ack.Context.ActionID != actionID {
					continue
				}
				//Yep we receive the right ack !
				return ack, nil
			}
		case <-signals:
			return nil, fmt.Errorf("Interrupted")
		case <-timeoutChan:
			return nil, fmt.Errorf("Timeout exceeded")
		}
	}
}
