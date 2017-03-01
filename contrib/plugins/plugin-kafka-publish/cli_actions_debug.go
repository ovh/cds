package main

import (
	"encoding/json"
	"io/ioutil"
	"strconv"

	"gopkg.in/urfave/cli.v1"

	"github.com/ovh/cds/contrib/plugins/plugin-kafka-publish/kafkapublisher"
)

func sendAction(c *cli.Context) error {
	args := []string{c.Args().First()}
	args = append(args, c.Args().Tail()...)
	if len(args) != 4 {
		cli.ShowCommandHelp(c, "send")
		return cli.NewExitError("Invalid usage", 20)
	}

	kafka := c.Args().Get(0)
	topic := c.Args().Get(1)
	key := c.Args().Get(2)
	file := c.Args().Get(3)

	producer, err := initKafkaProducer(kafka, key)
	if err != nil {
		return cli.NewExitError(err.Error(), 21)
	}

	f, err := kafkapublisher.OpenFile(file)
	if err != nil {
		return cli.NewExitError(err.Error(), 22)
	}

	if c.Int64("actionID") != 0 {
		i := c.Int64("actionID")
		f.ContextID = &i
	}

	pgpPubKey := c.String("pgp-encrypt")
	if pgpPubKey != "" {
		pubKey, err := ioutil.ReadFile(pgpPubKey)
		if err != nil {
			return cli.NewExitError(err.Error(), 23)
		}
		if err := f.EncryptContent(pubKey); err != nil {
			return cli.NewExitError(err.Error(), 24)
		}
	}

	if c.Bool("as-file") {
		if _, _, err := sendFileOnKafka(producer, topic, f); err != nil {
			return cli.NewExitError(err.Error(), 25)
		}
		return nil
	}

	if c.Bool("as-chunks") {
		chunks, err := f.KafkaMessages(512)
		if err != nil {
			return cli.NewExitError(err.Error(), 26)
		}
		if _, _, err := sendDataOnKafka(producer, topic, chunks); err != nil {
			return cli.NewExitError(err.Error(), 27)
		}
		return nil
	}

	cli.ShowCommandHelp(c, "send")
	return cli.NewExitError("Invalid usage choose option as-file or as-chunks", 20)
}

func sendContext(c *cli.Context) error {
	args := []string{c.Args().First()}
	args = append(args, c.Args().Tail()...)
	if len(args) <= 4 {
		cli.ShowCommandHelp(c, "send-context")
		return cli.NewExitError("Invalid usage", 30)
	}

	kafka := c.Args().Get(0)
	topic := c.Args().Get(1)
	key := c.Args().Get(2)
	actionID := c.Args().Get(3)
	iActionID, err := strconv.ParseInt(actionID, 10, 64)
	if err != nil {
		cli.ShowCommandHelp(c, "send-context")
		return cli.NewExitError(err.Error(), 30)
	}

	files := args[4:]

	ctx := kafkapublisher.NewContext(iActionID, files)

	producer, err := initKafkaProducer(kafka, key)
	if err != nil {
		return cli.NewExitError(err.Error(), 31)
	}

	btes, err := json.Marshal(ctx)
	if err != nil {
		return cli.NewExitError(err.Error(), 32)
	}

	if _, _, err := sendDataOnKafka(producer, topic, [][]byte{btes}); err != nil {
		return cli.NewExitError(err.Error(), 33)
	}

	return nil
}
