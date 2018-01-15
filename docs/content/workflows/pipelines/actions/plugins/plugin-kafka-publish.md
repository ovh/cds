+++
title = "plugin-kafka-publish"

+++

This action helps you to send data through Kafka across every network.

	You are able to send a custom "message" file and all the artifacts you want: there is no file size limit. To improve security, you can encrypt the files content with a GPG Key. From the consumer side, you will need to decrypt files content with you GPG private key and your passphrase.

	This action is a CDS Plugin packaged as a single binary file you can download and use to listen and consume data coming from CDS through Kafka. CDS can also wait for an acknowledgement coming from the consumer side. To send the acknowledgement, you can again use the plugin binary. For more details, see readme file of the plugin.

## Parameters

* **artifacts**: Artifacts list (comma separated)
* **kafkaAddresses**: Kafka Addresses
* **kafkaGroup**: Kafka Consumer Group (used for acknowledgment)
* **kafkaPassword**: Kafka Password
* **kafkaUser**: Kafka User
* **message**: Kafka Message
* **publicKey**: GPG Public Key (ASCII armored format)
* **topic**: Kafka Topic
* **waitForAck**: Wait for Ack
* **waitForAckTimeout**: Ack timeout (seconds). Used only if "waitForAck" is true.
* **waitForAckTopic**: Kafka Topic. Used only if "waitForAck" is true.


## More

More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/plugins/plugin-kafka-publish/README.md)

