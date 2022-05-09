# Go, Kubernetes, Twitter and Kafka
## Overview

This repository is a sample project to work on the `twitter api` and `kafka` in `kubernetes` environment

* Stream tweet then push into kafka topic
* Deploy to kubernetes

## Containerization

```shell
docker build -t samutup/sam-twitter:1.0.0 --no-cache -f DockerFile .
```
## Helm chart structure
```folder
|-- Chart.yaml
|-- charts
|-- templates
|   |-- NOTES.txt
|   |-- _helpers.tpl
|   |-- configmap.yaml
|   |-- deployment.yaml
|   |-- hpa.yaml
|   |-- ingress.yaml
|   |-- service.yaml
|   |-- serviceaccount.yaml
|   |-- tests
|   |   `-- test-connection.yaml
|   `-- tweetist.yaml
`-- values.yaml
```
## Helm Install

```shell
helm upgrade --install sam-twitter sam-twitter --debug
```

# Kafka Setup

## Docker Compose
Thanks to Bitnami, [Bitnami](https://hub.docker.com/r/bitnami/kafka/)
The intention is to have a kafka with persistence and an initial setup with some topics created
In docker-compose, we define the necessary services to run the kafka cluster something like,
```yaml
services:
  zookeeper:
    image: docker.io/bitnami/zookeeper:3.8
    ports:
      - "2181:2181"
    volumes:
      - "/Users/putumas/zookeeper-persistence:/bitnami/zookeeper"
    environment:
      - ALLOW_ANONYMOUS_LOGIN=yes
      - ZOOKEEPER_CLIENT_PORT=2181
      - ZOOKEEPER_TICK_TIME=2000
  kafka:
    image: docker.io/bitnami/kafka:3.1
    ports:
      - "9092:9092"
      - "9093:9093"
    volumes:
      - "/Users/putumas/kafka-persistence:/bitnami/kafka"
    environment:
      - KAFKA_CFG_ZOOKEEPER_CONNECT=zookeeper:2181
      - ALLOW_PLAINTEXT_LISTENER=yes
      - KAFKA_BROKER_ID=1
      - KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP=CLIENT:PLAINTEXT,EXTERNAL:PLAINTEXT
      - KAFKA_CFG_LISTENERS=CLIENT://:9092,EXTERNAL://:9093
      - KAFKA_CFG_ADVERTISED_LISTENERS=CLIENT://kafka:9092,EXTERNAL://localhost:9093
      - KAFKA_CFG_INTER_BROKER_LISTENER_NAME=CLIENT
    depends_on:
      - zookeeper

volumes:
  zookeeper_data:
    driver: local
  kafka_data:
    driver: local
```



