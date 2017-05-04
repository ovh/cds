+++
title = "plugin-kafka-publish"

[menu.main]
parent = "actions-plugins"
identifier = "plugin-kafka-publish"

+++

This action helps you to send data through Kafka across every network.

	You are able to send a custom "message" file and all the artifacts you want: there is no file size limit. To improve security, you can encrypt the files content with a GPG Key. From the consummer side, you will need to decrypt files content with you GPG private key and your passphrase.

	This action is a CDS Plugin packaged as a single binary file you can download and use to listen and consume data coming from CDS through Kafka. CDS can also wait for an acknowledgement coming from the consummer side. To send the acknowledgement, you can again use the plugin binary. For more details, see readme file of the plugin.

## Parameters

* **message**: Kafka Message
* **kafkaUser**: Kafka User
* **kafkaGroup**: Kafka Consumer Group (used for acknowledgment)
* **topic**: Kafka Topic
* **artifacts**: Artifacts list (comma separated)
* **publicKey**: GPG Public Key (ASCII armored format)
* **waitForAckTopic**: Kafka Topic. Used only if "waitForAck" is true.
* **waitForAckTimeout**: Ack timeout (seconds). Used only if "waitForAck" is true.
* **kafkaPassword**: Kafka Password
* **kafkaAddresses**: Kafka Addresses
* **waitForAck**: Wait for Ack


## More

More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/plugins/plugin-kafka-publish/README.md)

