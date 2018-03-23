# CDS to ES

## Configuration file

```toml

[Debug]
  log_level = "info"

[ElasticSearch]
  domain = "your-es"
  index = "your-index-cds"
  password = "your-token"
  port = "9200"
  protocol = "https"
  username = "your-username"

[Kafka]
  brokers = "your-broker:9093"
  group = "your-group"
  password = "your-kafka-password"
  topic = "your-topic"
  user = "your-kafka-user"

[Http]
  port = 9085

```

## Run it

```bash
cds2es -f config.yml
```