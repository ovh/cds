# CDS to HTTP

This µservice:
- consume a CDS Event Kafka topic
- for each event of type "EventNotif", POST to a HTTP Url the event content

## How to run it?

```
go build
./service --log-level=debug \
--event-kafka-broker-addresses=your-kafka-broker:9093 \
--event-kafka-version=0.10.2.0 \
--event-kafka-topic=cds-example.example-events \
--event-kafka-user=cds-example.reader \
--event-kafka-password=your-password \
--event-kafka-group=cds-example.reader.example-cds2http \
--event-remote-url=http://127.0.0.1:8080/cds/notifications
```
