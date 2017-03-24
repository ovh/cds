package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/bgentry/speakeasy"
	"github.com/fsamin/go-shredder"
	"github.com/phayes/permbits"
	"gopkg.in/urfave/cli.v1"

	"github.com/ovh/cds/contrib/plugins/plugin-kafka-publish/kafkapublisher"
)

//This shows the help
func helpAction(c *cli.Context) error {
	args := c.Args()
	if args.Present() {
		return cli.ShowCommandHelp(c, args.First())
	}

	cli.ShowAppHelp(c)
	return nil
}

//This shows information about the plugin
func infoAction(c *cli.Context) error {
	m := KafkaPlugin{}
	p := m.Parameters()

	fmt.Printf(" - Name:\t%s\n", m.Name())
	fmt.Printf(" - Author:\t%s\n", m.Author())
	fmt.Printf(" - Description:\t%s\n", m.Description())
	fmt.Println(" - Parameters:")
	for _, n := range p.Names() {
		fmt.Printf("\t - %s: %s\n", n, p.GetDescription(n))
	}
	return nil
}

//This will listen kafka topic forever and manage all incomming context, file and chunks
func listenAction(c *cli.Context) error {
	args := []string{c.Args().First()}
	args = append(args, c.Args().Tail()...)
	if len(args) != 5 {
		cli.ShowCommandHelp(c, "listen")
		return cli.NewExitError("Invalid usage", 10)
	}

	//Get arguments from commandline
	kafka := c.Args().Get(0)
	topic := c.Args().Get(1)
	group := c.Args().Get(2)
	user := c.Args().Get(3)
	password := c.Args().Get(4)

	//If provided, read the pgp private key file, and ask for the password
	pgpPrivKey := c.String("pgp-decrypt")
	var pgpPrivateKey, pgpPassphrase []byte
	if pgpPrivKey != "" {
		var err error
		pgpPrivateKey, err = ioutil.ReadFile(pgpPrivKey)
		if err != nil {
			return cli.NewExitError(err.Error(), 11)
		}
		password, err := speakeasy.Ask("Please enter your passphrase: ")
		if err != nil {
			return cli.NewExitError(err.Error(), 12)
		}
		pgpPassphrase = []byte(password)
	}

	//If provided, exec the script
	execScript := c.String("exec")
	if execScript != "" {
		if _, err := os.Stat(execScript); os.IsNotExist(err) {
			return cli.NewExitError(err.Error(), 14)
		}
		permissions, err := permbits.Stat(execScript)
		if err != nil {
			return cli.NewExitError(err.Error(), 15)
		}
		if !permissions.UserExecute() && !permissions.GroupExecute() && !permissions.OtherExecute() {
			return cli.NewExitError("exec script is not executable", 16)
		}
	}

	//Goroutine for kafka listening
	if err := consumeFromKafka(kafka, topic, group, user, password, pgpPrivateKey, pgpPassphrase, execScript); err != nil {
		return cli.NewExitError(err.Error(), 13)
	}

	return nil
}

//This will send a ack to CDS through Kafka. Entrypoint is the json context file
func ackAction(c *cli.Context) error {
	args := []string{c.Args().First()}
	args = append(args, c.Args().Tail()...)
	if len(args) != 6 {
		cli.ShowCommandHelp(c, "ack")
		return cli.NewExitError("Invalid usage", 40)
	}

	//Get arguments from commandline
	kafka := c.Args().Get(0)
	topic := c.Args().Get(1)
	user := c.Args().Get(2)
	password := c.Args().Get(3)
	contextFile := c.Args().Get(4)
	result := c.Args().Get(5)

	//Connect to kafka
	producer, err := initKafkaProducer(kafka, user, password)
	if err != nil {
		return cli.NewExitError(err.Error(), 44)
	}

	if result != "OK" && result != "KO" {
		cli.ShowCommandHelp(c, "ack")
		return cli.NewExitError("Invalid usage", 45)
	}

	//Read logs file
	var logsBody []byte
	if logFile := c.String("log"); logFile != "" {
		var err error
		logsBody, err = ioutil.ReadFile(logFile)
		if err != nil {
			return cli.NewExitError(err.Error(), 41)
		}
		if len(logsBody) > 700*1024 {
			return cli.NewExitError("Log file too large. Limit is up to 700 ko", 41)
		}
	}

	//Read the context json file
	contextBody, err := ioutil.ReadFile(contextFile)
	if err != nil {
		return cli.NewExitError(err.Error(), 42)
	}

	//Parste the context file
	ctx := &kafkapublisher.Context{}
	if err := json.Unmarshal(contextBody, ctx); err != nil {
		return cli.NewExitError(err.Error(), 43)
	}

	//Send artifacts
	if artifacts := c.StringSlice("artifact"); len(artifacts) > 0 {
		//Artifacts are send with AES encryption
		aes, err := getAESEncryptionOptions(password)
		if err != nil {
			return cli.NewExitError(err.Error(), 65)
		}
		var opts = &shredder.Opts{
			ChunkSize:     512 * 1024,
			AESEncryption: aes,
		}

		for _, a := range artifacts {
			chunks, err := shredder.ShredFile(a, fmt.Sprintf("%d", ctx.ActionID), opts)
			if err != nil {
				return cli.NewExitError(err.Error(), 66)
			}
			datas, err := kafkapublisher.KafkaMessages(chunks)
			if err != nil {
				return cli.NewExitError(err.Error(), 67)
			}
			if _, _, err := sendDataOnKafka(producer, topic, datas); err != nil {
				return cli.NewExitError(err.Error(), 68)
			}
		}
	}

	//Prepare the ack object wich will be send to kafka
	ack := kafkapublisher.Ack{
		Context: *ctx,
		Result:  result,
		Log:     logsBody,
	}

	//Marshal it to byte array
	ackBody, err := json.Marshal(ack)
	if err != nil {
		return cli.NewExitError(err.Error(), 46)
	}

	//Send it on kafka
	if _, _, err := sendDataOnKafka(producer, topic, [][]byte{ackBody}); err != nil {
		return cli.NewExitError(err.Error(), 47)
	}

	return nil
}

func getAESEncryptionOptions(password string) (*shredder.AESEncryption, error) {
	aeskey := []byte(password)
	if len(aeskey) > 32 {
		aeskey = aeskey[:32]
	} else {
		for len(aeskey) != 32 {
			aeskey = append(aeskey, '\x00')
		}
	}
	return &shredder.AESEncryption{Key: aeskey}, nil
}
