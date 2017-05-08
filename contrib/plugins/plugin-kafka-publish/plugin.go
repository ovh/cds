package main

import "github.com/ovh/cds/sdk/plugin"

//KafkaPlugin is a plugin to send data through kafka
type KafkaPlugin struct {
	plugin.Common
}

//Name return plugin names. It must me the same as the binary file
func (m KafkaPlugin) Name() string {
	return "plugin-kafka-publish"
}

//Description explains the purpose of the plugin
func (m KafkaPlugin) Description() string {
	return `This action helps you to send data through Kafka across every network.

	You are able to send a custom "message" file and all the artifacts you want: there is no file size limit. To improve security, you can encrypt the files content with a GPG Key. From the consumer side, you will need to decrypt files content with you GPG private key and your passphrase.

	This action is a CDS Plugin packaged as a single binary file you can download and use to listen and consume data coming from CDS through Kafka. CDS can also wait for an acknowledgement coming from the consumer side. To send the acknowledgement, you can again use the plugin binary. For more details, see readme file of the plugin.`
}

//Author of the plugin
func (m KafkaPlugin) Author() string {
	return "François SAMIN <francois.samin@corp.ovh.com>"
}

//Parameters return parameters description
func (m KafkaPlugin) Parameters() plugin.Parameters {
	params := plugin.NewParameters()

	params.Add("message", plugin.TextParameter, "Kafka Message", `{
    "project" : "{{.cds.project}}",
    "application" : "{{.cds.application}}",
    "pipeline" : "{{.cds.pipeline}}",
    "version" : "{{.cds.version}}"
}`)
	params.Add("kafkaUser", plugin.StringParameter, "Kafka User", "{{.cds.proj.kafkaUser}}")
	params.Add("kafkaPassword", plugin.StringParameter, "Kafka Password", "{{.cds.proj.kafkaPassword}}")
	params.Add("kafkaGroup", plugin.StringParameter, "Kafka Consumer Group (used for acknowledgment)", "{{.cds.proj.kafkaGroup}}")
	params.Add("kafkaAddresses", plugin.StringParameter, "Kafka Addresses", "{{.cds.proj.kafkaAddresses}}")
	params.Add("topic", plugin.StringParameter, "Kafka Topic", "{{.cds.env.kafkaTopic}}")
	params.Add("artifacts", plugin.StringParameter, "Artifacts list (comma separated)", "")
	params.Add("publicKey", plugin.StringParameter, "GPG Public Key (ASCII armored format)", "{{.cds.proj.gpgPubAsc}}")
	params.Add("waitForAck", plugin.BooleanParameter, `Wait for Ack`, "true")
	params.Add("waitForAckTopic", plugin.StringParameter, `Kafka Topic. Used only if "waitForAck" is true.`, "{{.cds.env.kafkaAckTopic}}")
	params.Add("waitForAckTimeout", plugin.NumberParameter, `Ack timeout (seconds). Used only if "waitForAck" is true.`, "120")

	return params
}
