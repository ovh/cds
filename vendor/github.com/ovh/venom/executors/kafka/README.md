# Venom - Executor Kafka

Step for execute a Kakka

## Input
In your yaml file, you can use:

```yaml

  - addrs mandatory
  - with_tls optional
  - with_sasl optional
  - with_sasl_handshaked optional
  - user optional
  - password optional

  - client_type mandator : producer or consumer

  # for consumer client type:
  - group_id mandatory
  - topics mandatory
  - timeout optional
  - message_limit optional
  - initial_offset optional
  - mark_offset optional

  # for producer client type:
  - messages
  - messages_file

```

Example :

```yaml

name: My Kafka testsuite
version: "2"
testcases:
- name: Kafka test
  description: Test kafka
  steps:
  - type: kafka
    clientType: producer
    withSASL: true
    withTLS: true
    user: "{{.kafkaUser}}"
    password: "{{.kafkaPwd}}"
    addrs:
      - "{{.kafkaHost}}:{{kafkaPort}}"
    messages:
    - topic: test-topic
      value: '{"hello":"bar"}'
  - type: kafka
    clientType: consumer
    withTLS: true
    withSASL: true
    user: "{{.kafkaUser}}"
    password: "{{.kafkaPwd}}"
    markOffset: true
    initialOffset: oldest
    messageLimit: 1
    groupID: venom
    addrs:
      - "{{.kafkaHost}}:{{kafkaPort}}"
    topics:
      - test-topic
    assertions:
    - result.messagesjson.messagesjson0.value.hello ShouldEqual bar
    - result.messages.__len__ ShouldEqual 1

```