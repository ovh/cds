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
		Author: "Alexandre JIN  <alexandre.jin@corp.ovh.com>",
		Description: `This action helps you generates a file using a template file and text/template golang package.

	Check documentation on text/template for more information https://golang.org/pkg/text/template.`,
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
		return fail("Kafka is not configured: missing kafkaUser, kafkaPassword, kafkaAddresses or kafkaTopic")
	}

	waitForAckString := q.GetOptions()["waitForAck"]
	var ackTopic string
	var timeout int
	if waitForAckString == "true" {
		ackTopic = q.GetOptions()["waitForAckTopic"]
		timeoutStr := q.GetOptions()["waitForAckTimeout"]

		timeout, _ = strconv.Atoi(timeoutStr)
		if ackTopic == "" && timeout == 0 {
			return fail("Error: ackTopic and waitForAckTimeout parameters are mandatory")
		}
	}

	message := q.GetOptions()["message"]
	messageFile, err := tmplMessage(q, []byte(message))
	if err != nil {
		return fail("Error on tmpMessage: %v", err)
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
					return fail("%s : no such file", f)
				}
			}
		} else {
			filesPath, err := filepath.Glob(artifactsList)
			if err != nil {
				return fail("Unable to parse files %s: %s", artifactsList, err)
			}
			artifacts = filesPath
		}

		files = append(files, artifacts...)
	}

	//Send the context message
	ctx := kafkapublisher.NewContext(q.GetJobID(), files)

	producer, err := initKafkaProducer(kafka, user, password)
	if err != nil {
		return fail("Unable to connect to kafka : %s", err)
	}

	btes, err := json.Marshal(ctx)
	if err != nil {
		return fail("Error : %s", err)
	}

	if _, _, err := sendDataOnKafka(producer, topic, [][]byte{btes}); err != nil {
		return fail("Unable to send on kafka : %s", err)
	}

	pubKey := q.GetOptions()["publicKey"]

	//Send all the files
	for _, f := range files {
		aes, err := getAESEncryptionOptions(key)
		if err != nil {
			return fail("Unable to shred file %s : %s", f, err)
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

		chunks, err := shredder.ShredFile(f, q.GetOptions()[""], opts)
		if err != nil {
			return fail("Unable to shred file %s : %s", f, err)
		}

		datas, err := kafkapublisher.KafkaMessages(chunks)
		if err != nil {
			return fail("Unable to compute chunks for file %s : %s", f, err)
		}
		if _, _, err := sendDataOnKafka(producer, topic, datas); err != nil {
			return fail("Unable to send chunks through kafka : %s", err)
		}
	}

	Logf("Data sent to topic %s, action %d : %v", topic, q.GetJobID(), files)

	//Don't wait for ack
	if waitForAckString != "true" {
		return &actionplugin.ActionResult{
			Status: sdk.StatusSuccess.String(),
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
		return fail("Failed to get ack on topic %s: %s", ackTopic, err)
	}

	//Check the ack
	Logf("Got ACK from %s : %s", ackTopic, ack.Result)
	if len(ack.Log) > 0 {
		Logf(string(ack.Log))
	}
	if ack.Result == "OK" {
		return &actionplugin.ActionResult{
			Status: sdk.StatusSuccess.String(),
		}, nil
	}

	Logf("Ack Received: %+v\n", ack)

	return &actionplugin.ActionResult{
		Status:  sdk.StatusSuccess.String(),
		Details: fmt.Sprintf("Plugin failed with ACK.Result:%s", ack.Result),
	}, nil
}

func (actPlugin *kafkaPublishActionPlugin) WorkerHTTPPort(ctx context.Context, q *actionplugin.WorkerHTTPPortQuery) (*empty.Empty, error) {
	actPlugin.HTTPPort = q.Port
	return &empty.Empty{}, nil
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

func fail(format string, args ...interface{}) (*actionplugin.ActionResult, error) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(msg)
	return &actionplugin.ActionResult{
		Details: msg,
		Status:  sdk.StatusFail.String(),
	}, nil
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
		Logf("Invalid template format: %s\n", err.Error())
		return "", err
	}

	out, err := os.Create("message")
	if err != nil {
		Logf("Error writing temporary file : %s\n", err.Error())
		return "", err
	}
	outPath := out.Name()
	if err := t.Execute(out, data); err != nil {
		Logf("Failed to execute template: %s\n", err.Error())
		return "", err
	}

	return outPath, nil
}
