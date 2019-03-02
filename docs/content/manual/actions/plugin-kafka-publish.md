+++
title = "plugin-kafka-publish"

+++

This action helps you to send data through Kafka across every network.

You are able to send a custom "message" file and all the artifacts you want: there is no file size limit. To improve security, you can encrypt the files content with a GPG Key. From the consumer side, you will need to decrypt files content with you GPG private key and your passphrase.

This action is a CDS Plugin packaged as a single binary file you can download and use to listen and consume data coming from CDS through Kafka. CDS can also wait for an acknowledgement coming from the consumer side. To send the acknowledgement, you can again use the plugin binary. For more details, see readme file of the plugin.

How to use: https://github.com/ovh/cds/tree/master/contrib/grpcplugins/action/kafka-publish


## Parameters

* **message**: Kafka Message
* **kafkaAddresses**: Kafka Addresses
* **topic**: Kafka Topic
* **publicKey**: GPG Public Key (ASCII armored format)
* **key**: Key used by AES Encryption. It have to be the same value as --key on plugin binary
* **waitForAckTopic**: Wait for ACK topic
* **waitForAckTimeout**: Ack timeout (seconds). Used only if "waitForAck" is true.
* **kafkaUser**: Kafka user
* **kafkaPassword**: Kafka password
* **kafkaGroup**: Kafka Consumer Group (used for acknowledgment)
* **artifacts**: Artifacts list (comma separated)"
* **waitForAck**: Wait for ACK



