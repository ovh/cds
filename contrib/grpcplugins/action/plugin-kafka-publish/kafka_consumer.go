package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strconv"

	"github.com/Shopify/sarama"
	"github.com/fsamin/go-shredder"

	"github.com/ovh/cds/contrib/grpcplugins/action/kafka-publish/kafkapublisher"
)

func consumeFromKafka(kafka, topic, group, user, password, key string, gpgPrivatekey, gpgPassphrase []byte, execScript string) error {
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
	defer partitionOffsetManager.Close()

	// Start a consumer at next offset
	offset, _ := partitionOffsetManager.NextOffset()
	partitionConsumer, err := consumer.ConsumePartition(topic, p, offset)
	if err != nil {
		return err
	}
	defer partitionConsumer.Close()

	// Asynchronously handle message
	go consumptionHandler(partitionConsumer, topic, partitionOffsetManager, messagesChan, errorsChan)

	// Trap SIGINT to trigger a shutdown.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	chunks := shredder.Chunks{}
	contexts := map[string]*kafkapublisher.Context{}

	for {
		select {
		case msg := <-messagesChan:
			//If we receive a "Context" Message
			if c, ok := kafkapublisher.GetContext(msg); ok {
				id := fmt.Sprintf("%d", c.ActionID)
				if contexts[id] != nil {
					fmt.Printf("Context reinitialized : %s\n", msg)
					os.RemoveAll(contexts[id].Directory)
				} else {
					fmt.Printf("New Context received : %s\n", msg)
				}
				contexts[id] = c
				continue
			}

			//If we receive a "Chunk" Message
			if kafkapublisher.IsChunk(msg) {
				c, err := kafkapublisher.ReadBytes(msg)
				if err != nil {
					fmt.Printf("Unable to read bytes : %v\n", err)
					continue
				}
				chunks = append(chunks, *c)

				actionID := c.Ctx.GetUUID()
				opts, errOpts := getShredderOpts(key, gpgPrivatekey, gpgPassphrase)
				if errOpts != nil {
					fmt.Printf("Error on getShredderOpts : %v\n", errOpts)
					continue
				}

				//Try to match a context
				ctx, ok := contexts[actionID]
				if !ok {
					fmt.Printf("Unknown CDS context : %s\n", c.Ctx.UUID)
				}

				allChunks := shredder.Filter(chunks)
				cs := allChunks[c.Ctx.UUID]

				//If we received all chunks for a file, let save it on disk
				if cs.Completed() {
					content, err := shredder.Reassemble(cs, opts)
					if err != nil {
						fmt.Printf("Error: %v\n", err)
						continue
					}

					filename, data, err := content.File()
					if err != nil {
						fmt.Printf("Error: %v\n", err)
						continue
					}

					if err := fileHandler(ctx, filename, data); err != nil {
						fmt.Printf("Error: %v\n", err)
						continue
					}
					//File has been processed, remove data from memory
					chunks.Delete(*c)
				}

				if ctx != nil && ctx.Closed {
					//File has been processed, remove data from memory
					delete(contexts, c.Ctx.UUID)
					fmt.Printf("Context %d successfully closed\n", ctx.ActionID)

					if execScript != "" {
						cmd := exec.Command(execScript, getCtxFileName(ctx))
						var stdOut = new(bytes.Buffer)
						var stdErr = new(bytes.Buffer)
						cmd.Stdout = stdOut
						cmd.Stderr = stdErr
						if err := cmd.Run(); err != nil {
							fmt.Printf("Error with Exec: %s : %s\n", execScript, err)
						}
						if len(stdOut.String()) > 0 {
							fmt.Println(stdOut.String())
						}
						if len(stdErr.String()) > 0 {
							fmt.Println(stdErr.String())
						}
					}
				}
				continue
			}

			//We received a plain test, just display it
			fmt.Printf("% x\n", msg)

		case err := <-errorsChan:
			fmt.Printf("%v\n", err)
			return err
		case <-signals:
			return nil
		}
	}
}

func getShredderOpts(key string, gpgPrivatekey, gpgPassphrase []byte) (*shredder.Opts, error) {
	//Default is AES encryption
	aes, err := getAESEncryptionOptions(key)
	if err != nil {
		return nil, err
	}
	var opts = &shredder.Opts{
		ChunkSize:     512 * 1024,
		AESEncryption: aes,
	}

	//If provided use GPG encryption
	if len(gpgPrivatekey) > 0 && len(gpgPassphrase) > 0 {
		opts.AESEncryption = nil
		opts.GPGEncryption = &shredder.GPGEncryption{
			PrivateKey: gpgPrivatekey,
			Passphrase: gpgPassphrase,
		}
	}
	return opts, nil
}

// ConsumptionHandler pipes the handled messages and push them to a chan
func consumptionHandler(pc sarama.PartitionConsumer, topic string, po sarama.PartitionOffsetManager, messagesChan chan<- []byte, errorsChan chan<- error) {
	for {
		select {
		case msg := <-pc.Messages():
			// Write message consumed in the sub channel
			if msg != nil {
				messagesChan <- msg.Value
				po.MarkOffset(msg.Offset+1, topic)
			}
		case err := <-pc.Errors():
			if err != nil {
				fmt.Println(err)
				errorsChan <- err
			}
		case offsetErr := <-po.Errors():
			if offsetErr != nil {
				fmt.Println(offsetErr)
				errorsChan <- offsetErr
			}
		}
	}
}

//This manages a file composed of chunks within a context or not
func fileHandler(ctx *kafkapublisher.Context, filename string, data []byte) error {
	//No context
	if ctx == nil {
		fmt.Printf("Received file %s\n", filename)
		if err := ioutil.WriteFile(filename, data, os.FileMode(0644)); err != nil {
			return err
		}
		return nil
	}

	//Context is not nil
	var found bool
	for _, name := range ctx.Files {
		if name == filename {
			found = true
			break
		}
	}

	//The file doesn't match with the context
	if !found {
		return fmt.Errorf("File %s is not expected in context %d", filename, ctx.ActionID)
	}

	//Mkdir the directory
	if err := os.MkdirAll(ctx.Directory, os.FileMode(0755)); err != nil {
		return err
	}

	//Write the file
	filepath := path.Join(ctx.Directory, filename)
	fmt.Printf("Received file %s in context %d => %s\n", filename, ctx.ActionID, filepath)
	if err := ioutil.WriteFile(filepath, data, os.FileMode(0644)); err != nil {
		return err
	}

	//Mark the file as received in the context
	ctx.ReceivedFiles[filename] = true

	//Write the Context file
	if ctx.IsComplete() {
		name := getCtxFileName(ctx)
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

func getCtxFileName(ctx *kafkapublisher.Context) string {
	return "cds-action-" + strconv.Itoa(int(ctx.ActionID)) + ".json"
}
