package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	shredder "github.com/fsamin/go-shredder"
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/contrib/grpcplugins/action/kafka-publish/kafkapublisher"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

/* Inside contrib/grpcplugins/action
$ make build kafka-publish
$ make publish kafka-publish
*/

type kafkaPublishActionPlugin struct {
	actionplugin.Common
}

func (actPlugin *kafkaPublishActionPlugin) Manifest(ctx context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:   "plugin-kafka-publish",
		Author: "François SAMIN <francois.samin@corp.ovh.com>",
		Description: `This action helps you to send data through Kafka across every network.

		You are able to send a custom "message" file and all the artifacts you want: there is no file size limit. To improve security, you can encrypt the files content with a GPG Key. From the consumer side, you will need to decrypt files content with you GPG private key and your passphrase.
	  
		This action is a CDS Plugin packaged as a single binary file you can download and use to listen and consume data coming from CDS through Kafka. CDS can also wait for an acknowledgement coming from the consumer side. To send the acknowledgement, you can again use the plugin binary. For more details, see readme file of the plugin.
		
		How to use: https://github.com/ovh/cds/tree/master/contrib/grpcplugins/action/kafka-publish
		`,
		Version: sdk.VERSION,
	}, nil
}

func (actPlugin *kafkaPublishActionPlugin) Run(ctxBack context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	kafka := q.GetOptions()["kafkaAddresses"]
	user := q.GetOptions()["kafkaUser"]
	password := q.GetOptions()["kafkaPassword"]
	group := q.GetOptions()["kafkaGroup"]
	topic := q.GetOptions()["topic"]
	key := q.GetOptions()["key"]
	if key == "" {
		key = password
	}

	if user == "" || password == "" || kafka == "" || topic == "" {
		return actionplugin.Fail("Kafka is not configured: missing kafkaUser, kafkaPassword, kafkaAddresses or kafkaTopic")
	}

	waitForAckString := q.GetOptions()["waitForAck"]
	var ackTopic string
	var timeout int
	if waitForAckString == "true" {
		ackTopic = q.GetOptions()["waitForAckTopic"]
		timeoutStr := q.GetOptions()["waitForAckTimeout"]

		timeout, _ = strconv.Atoi(timeoutStr)
		if ackTopic == "" && timeout == 0 {
			return actionplugin.Fail("Error: ackTopic and waitForAckTimeout parameters are mandatory")
		}
	}

	message := q.GetOptions()["message"]
	messageFile, err := tmplMessage(q, []byte(message))
	if err != nil {
		return actionplugin.Fail("Error on tmpMessage: %v", err)
	}
	files := []string{messageFile}

	//Check if every file exist
	artifactsList := q.GetOptions()["artifacts"]
	if strings.TrimSpace(artifactsList) != "" {
		var artifacts []string
		//If the parameter contains a comma, consider it as a list; else glob it
		if strings.Contains(artifactsList, ",") {
			artifacts := strings.Split(artifactsList, ",")
			for _, f := range artifacts {
				if _, err := os.Stat(f); os.IsNotExist(err) {
					return actionplugin.Fail("%s : no such file", f)
				}
			}
		} else {
			filesPath, err := filepath.Glob(artifactsList)
			if err != nil {
				return actionplugin.Fail("Unable to parse files %s: %v", artifactsList, err)
			}
			artifacts = filesPath
		}

		files = append(files, artifacts...)
	}

	//Send the context message
	ctx := kafkapublisher.NewContext(q.GetJobID(), files)

	producer, err := initKafkaProducer(kafka, user, password)
	if err != nil {
		return actionplugin.Fail("Unable to connect to kafka: %v", err)
	}

	btes, err := json.Marshal(ctx)
	if err != nil {
		return actionplugin.Fail("Error: %v", err)
	}

	if _, _, err := sendDataOnKafka(producer, topic, [][]byte{btes}); err != nil {
		return actionplugin.Fail("Unable to send on kafka: %v", err)
	}

	pubKey := q.GetOptions()["publicKey"]

	//Send all the files
	for _, f := range files {
		aes, err := getAESEncryptionOptions(key)
		if err != nil {
			return actionplugin.Fail("Unable to shred file %s: %s", f, err)
		}
		var opts = &shredder.Opts{
			ChunkSize:     512 * 1024,
			AESEncryption: aes,
		}

		//If provided use GPG encryption
		if pubKey != "" {
			opts.AESEncryption = nil
			opts.GPGEncryption = &shredder.GPGEncryption{
				PublicKey: []byte(pubKey),
			}
		}

		chunks, err := shredder.ShredFile(f, fmt.Sprintf("%d", q.GetJobID()), opts)
		if err != nil {
			return actionplugin.Fail("Unable to shred file %s : %s", f, err)
		}

		datas, err := kafkapublisher.KafkaMessages(chunks)
		if err != nil {
			return actionplugin.Fail("Unable to compute chunks for file %s: %v", f, err)
		}
		if _, _, err := sendDataOnKafka(producer, topic, datas); err != nil {
			return actionplugin.Fail("Unable to send chunks through kafka: %v", err)
		}
	}

	Logf("Data sent to topic %s, action %d : %v", topic, q.GetJobID(), files)

	//Don't wait for ack
	if waitForAckString != "true" {
		return &actionplugin.ActionResult{
			Status: sdk.StatusSuccess,
		}, nil
	}

	//Log every 5 sesonds
	ticker := time.NewTicker(time.Second * 5)
	stop := make(chan bool, 1)
	defer func() {
		stop <- true
		ticker.Stop()
	}()
	go func() {
		t0 := time.Now()
		for {
			select {
			case t := <-ticker.C:
				delta := math.Floor(t.Sub(t0).Seconds())
				Logf("[%d seconds] Please wait...\n", int(delta))
			case <-stop:
				return
			}
		}
	}()

	//Wait for ack
	ack, err := ackFromKafka(kafka, ackTopic, group, user, password, key, time.Duration(timeout)*time.Second, q.GetJobID())
	if err != nil {
		return actionplugin.Fail("Failed to get ack on topic %s: %v", ackTopic, err)
	}

	//Check the ack
	Logf("Got ACK from %s : %s", ackTopic, ack.Result)
	if len(ack.Log) > 0 {
		Logf(string(ack.Log))
	}
	if ack.Result == "OK" {
		return &actionplugin.ActionResult{
			Status: sdk.StatusSuccess,
		}, nil
	}

	Logf("Ack Received: %+v\n", ack)

	return actionplugin.Fail("Plugin failed with ACK.Result:%s", ack.Result)
}

func main() {
	app := initCli(func() {
		actPlugin := kafkaPublishActionPlugin{}
		if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
			panic(err)
		}
	})
	_ = app.Run(os.Args)
	return
}

func tmplMessage(q *actionplugin.ActionQuery, buff []byte) (string, error) {
	fileContent := string(buff)
	data := map[string]string{}
	for k, v := range q.GetOptions() {
		kb := strings.Replace(k, ".", "__", -1)
		data[kb] = v
		re := regexp.MustCompile("{{." + k + "(.*)}}")
		for {
			sm := re.FindStringSubmatch(fileContent)
			if len(sm) > 0 {
				fileContent = strings.Replace(fileContent, sm[0], "{{."+kb+sm[1]+"}}", -1)
			} else {
				break
			}
		}
	}

	funcMap := template.FuncMap{
		"title": strings.Title,
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
	}

	t, err := template.New("file").Funcs(funcMap).Parse(fileContent)
	if err != nil {
		Logf("Invalid template format: %v\n", err.Error())
		return "", err
	}

	out, err := os.Create("message")
	if err != nil {
		Logf("Error writing temporary file : %v\n", err.Error())
		return "", err
	}
	outPath := out.Name()
	if err := t.Execute(out, data); err != nil {
		Logf("Failed to execute template: %v\n", err.Error())
		return "", err
	}

	return outPath, nil
}
