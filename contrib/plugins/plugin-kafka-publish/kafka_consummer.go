package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"strconv"

	"github.com/Shopify/sarama"

	"github.com/ovh/cds/contrib/plugins/plugin-kafka-publish/kafkapublisher"
)

func consumeFromKafka(kafka, topic, group, key string) error {
	//Create a new client
	var config = sarama.NewConfig()
	// Set key as the client id for authentication
	config.ClientID = key
	client, err := sarama.NewClient([]string{kafka}, config)
	if err != nil {
		return err
	}

	// Create an offsetManager
	offsetManager, err := sarama.NewOffsetManagerFromClient(group, client)
	if err != nil {
		return err
	}

	// Create a client
	consumer, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		return err
	}

	fmt.Printf("Listening Kafka %s on topic %s...\n", kafka, topic)

	// Create the message chan, that will receive the queue
	messagesChan := make(chan []byte)
	// Create the error chan, that will receive the queue
	errorsChan := make(chan error)

	// read the number of partition for the given topic
	partitions, err := consumer.Partitions(topic)
	if err != nil {
		return err
	}

	// Create a consumer for each partition
	if len(partitions) > 1 {
		return fmt.Errorf("Multiple partition not supported")
	}
	p := partitions[0]
	partitionOffsetManager, err := offsetManager.ManagePartition(topic, p)
	if err != nil {
		return err
	}
	defer partitionOffsetManager.AsyncClose()

	// Start a consumer at next offset
	offset, _ := partitionOffsetManager.NextOffset()
	partitionConsumer, err := consumer.ConsumePartition(topic, p, offset)
	if err != nil {
		return err
	}
	defer partitionConsumer.AsyncClose()

	// Asynchronously handle message
	go consumptionHandler(partitionConsumer, topic, partitionOffsetManager, messagesChan, errorsChan)

	// Trap SIGINT to trigger a shutdown.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	files := map[string]*kafkapublisher.File{}
	chunks := map[string]*kafkapublisher.Chunks{}
	contexts := map[int64]*kafkapublisher.Context{}

	for {
		select {
		case msg := <-messagesChan:
			//If we receive a "Context" Message
			if c, ok := kafkapublisher.GetContext(msg); ok {
				if contexts[c.ActionID] != nil {
					fmt.Printf("Context reinitialized : %s\n", msg)
					os.RemoveAll(contexts[c.ActionID].Directory)
				} else {
					fmt.Printf("New Context received : %s\n", msg)
				}
				contexts[c.ActionID] = c
				continue
			}

			//If we recieve a "Chunk" Message
			if kafkapublisher.IsChunk(msg) {
				f, c, err := kafkapublisher.ReadBytes(msg)
				if err != nil {
					fmt.Printf("Unable to read bytes : %s\n", err)
					continue
				}
				if files[f.ID] == nil {
					files[f.ID] = f
				}
				if chunks[f.ID] == nil {
					chunks[f.ID] = &kafkapublisher.Chunks{}
				}
				cs := *chunks[f.ID]
				cs = append(cs, *c)
				chunks[f.ID] = &cs

				//Try to match a context
				var ctx *kafkapublisher.Context
				if c.ContextID != nil {
					ctx = contexts[*c.ContextID]
				}

				//If we received all chunks for a file, let save it on disk
				if cs.IsFileComplete(f) {
					if err := fileHandler(ctx, f, chunks[f.ID]); err != nil {
						fmt.Printf("Error: %s\n", err)
						continue
					}
					//File has been processed, remove data from memory
					delete(files, f.ID)
					delete(chunks, f.ID)
				}

				if ctx != nil && ctx.Closed {
					//File has been processed, remove data from memory
					delete(contexts, *c.ContextID)
					fmt.Printf("Context %d successfully closed\n", *c.ContextID)
				}
				continue
			}

			//We received a plain test, just display it
			fmt.Printf("% x\n", msg)

		case err := <-errorsChan:
			fmt.Printf("%s\n", err)
			return err
		case <-signals:
			return nil
		}
	}
}

// ConsumptionHandler pipes the handled messages and push them to a chan
func consumptionHandler(pc sarama.PartitionConsumer, topic string, po sarama.PartitionOffsetManager, messagesChan chan<- []byte, errorsChan chan<- error) {
	for {
		select {
		case msg := <-pc.Messages():
			// Write message consumed in the sub channel
			messagesChan <- msg.Value
			po.MarkOffset(msg.Offset+1, topic)
		case err := <-pc.Errors():
			fmt.Println(err)
			errorsChan <- err
		case offsetErr := <-po.Errors():
			fmt.Println(offsetErr)
			errorsChan <- offsetErr
		}
	}
}

var (
	pgpPassphrase []byte
	pgpPrivateKey []byte
)

//This manages a file composed of chunks within a context or not
func fileHandler(ctx *kafkapublisher.Context, f *kafkapublisher.File, chunks *kafkapublisher.Chunks) error {
	//Reassemble the chunks onto the file
	if err := chunks.Reassemble(f); err != nil {
		return err
	}

	//If there is a private key, decrypt the file content
	if len(pgpPrivateKey) != 0 {
		if err := f.DecryptContent(pgpPrivateKey, pgpPassphrase); err != nil {
			return err
		}
	}

	//No context
	if ctx == nil {
		fmt.Printf("Received file %s\n", f.Name)
		if err := ioutil.WriteFile(f.Name, f.Content.Bytes(), os.FileMode(0644)); err != nil {
			return err
		}

		return nil
	}

	//Context is not nil
	var found bool
	for _, name := range ctx.Files {
		if name == f.Name {
			found = true
			break
		}
	}

	//The file doesn't match with the context
	if !found {
		return fmt.Errorf("File %s is not expected in context %d", f.Name, ctx.ActionID)
	}

	//Mkdir the directory
	if err := os.MkdirAll(ctx.Directory, os.FileMode(0755)); err != nil {
		return err
	}

	//Write the file
	filename := path.Join(ctx.Directory, f.Name)
	fmt.Printf("Received file %s in context %d (%s)\n", f.Name, ctx.ActionID, filename)
	if err := ioutil.WriteFile(filename, f.Content.Bytes(), os.FileMode(0644)); err != nil {
		return err
	}

	//Mark the file as received in the context
	ctx.ReceivedFiles[f.Name] = true

	//Write the Context file
	if ctx.IsComplete() {
		name := "cds-action-" + strconv.Itoa(int(ctx.ActionID)) + ".json"
		buff, err := json.Marshal(ctx)
		if err != nil {
			return err
		}
		if err := ioutil.WriteFile(name, buff, os.FileMode(0644)); err != nil {
			return err
		}
		ctx.Closed = true
	}

	return nil
}
