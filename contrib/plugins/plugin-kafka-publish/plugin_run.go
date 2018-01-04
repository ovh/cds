package main

import (
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

	"github.com/fsamin/go-shredder"
	"github.com/ovh/cds/contrib/plugins/plugin-kafka-publish/kafkapublisher"
	"github.com/ovh/cds/sdk/plugin"
)

var (
	version = "0.3"
	job     plugin.IJob
)

//Run execute the action
func (m KafkaPlugin) Run(j plugin.IJob) plugin.Result {
	job = j
	kafka := job.Arguments().Get("kafkaAddresses")
	user := job.Arguments().Get("kafkaUser")
	password := job.Arguments().Get("kafkaPassword")
	group := job.Arguments().Get("kafkaGroup")
	topic := job.Arguments().Get("topic")

	if user == "" || password == "" || kafka == "" || topic == "" {
		Logf("Kafka is not configured : %+v", job.Arguments().Data)
		return plugin.Fail
	}

	waitForAckString := job.Arguments().Get("waitForAck")
	var ackTopic string
	var timeout int
	if waitForAckString == "true" {
		ackTopic = job.Arguments().Get("waitForAckTopic")
		timeoutStr := job.Arguments().Get("waitForAckTimeout")

		timeout, _ = strconv.Atoi(timeoutStr)
		if ackTopic == "" && timeout == 0 {
			Logf("Error: ackTopic and waitForAckTimeout parameters are mandatory")
			return plugin.Fail
		}

	}

	message := job.Arguments().Get("message")
	messageFile, err := tmplMessage(job, []byte(message))
	if err != nil {
		return plugin.Fail
	}
	files := []string{messageFile}

	//Check if every file exist
	artifactsList := job.Arguments().Get("artifacts")
	if strings.TrimSpace(artifactsList) != "" {

		var artifacts []string
		//If the parameter contains a comma, consider it as a list; else glob it
		if strings.Contains(artifactsList, ",") {
			artifacts := strings.Split(artifactsList, ",")
			for _, f := range artifacts {
				if _, err := os.Stat(f); os.IsNotExist(err) {
					Logf("%s : no such file", f)
					return plugin.Fail
				}
			}
		} else {
			filesPath, err := filepath.Glob(artifactsList)
			if err != nil {
				Logf("Unable to parse files %s: %s", artifactsList, err)
				return plugin.Fail
			}
			artifacts = filesPath
		}

		files = append(files, artifacts...)
	}

	//Send the context message
	ctx := kafkapublisher.NewContext(job.ID(), files)

	producer, err := initKafkaProducer(kafka, user, password)
	if err != nil {
		Logf("Unable to connect to kafka : %s", err)
		return plugin.Fail
	}

	btes, err := json.Marshal(ctx)
	if err != nil {
		Logf("Error : %s", err)
		return plugin.Fail
	}

	if _, _, err := sendDataOnKafka(producer, topic, [][]byte{btes}); err != nil {
		Logf("Unable to send on kafka : %s", err)
		return plugin.Fail
	}

	pubKey := job.Arguments().Get("publicKey")

	//Send all the files
	for _, f := range files {
		aes, err := getAESEncryptionOptions(password)
		if err != nil {
			Logf("Unable to shred file %s : %s", f, err)
			return plugin.Fail
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

		chunks, err := shredder.ShredFile(f, fmt.Sprintf("%d", job.ID()), opts)
		if err != nil {
			Logf("Unable to shred file %s : %s", f, err)
			return plugin.Fail
		}

		datas, err := kafkapublisher.KafkaMessages(chunks)
		if err != nil {
			Logf("Unable to compute chunks for file %s : %s", f, err)
			return plugin.Fail
		}
		if _, _, err := sendDataOnKafka(producer, topic, datas); err != nil {
			Logf("Unable to send chunks through kafka : %s", err)
			return plugin.Fail
		}
	}

	Logf("Data sent to topic %s, action id: %d : %v", topic, job.ID(), files)

	//Don't wait for ack
	if waitForAckString != "true" {
		return plugin.Success
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
	ack, err := ackFromKafka(kafka, ackTopic, group, user, password, time.Duration(timeout)*time.Second, job.ID())
	if err != nil {
		Logf("Failed to get ack on topic %s: %s", ackTopic, err)
		return plugin.Fail
	}

	//Check the ack
	Logf("Got ACK from %s : %s", ackTopic, ack.Result)
	if len(ack.Log) > 0 {
		Logf(string(ack.Log))
	}
	if ack.Result == "OK" {
		return plugin.Success
	}

	return plugin.Fail
}

func tmplMessage(j plugin.IJob, buff []byte) (string, error) {
	fileContent := string(buff)
	data := map[string]string{}
	for k, v := range j.Arguments().Data {
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
