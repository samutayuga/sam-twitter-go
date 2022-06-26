# Go, Kubernetes and Kafka

## Overview
This is the rest service to control the consumption of a kafka topic

* read `get /topic/start`
* stop `patch /topic/stop`
* resume `patch /topic/resume`
* pause `patch /topic/pause`

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
# Useful Command

`Create Topic`

```shell
./kafka-topics.sh --bootstrap-server=127.0.0.1:9093 --create --topic=DIFF-FIFO-ACC_062_OPER --partitions=1 --config retention.ms=-1
```

`Alter Topic`

```shell
./kafka-configs.sh --bootstrap-server 127.0.0.1:9093 --alter --topic CDC-CDC_OPER-FPL_OPER --add-config retention.ms=-1
```

`Check if a topic have a message`

```shell
./kafka-get-offsets.sh --bootstrap-server 127.0.0.1:9093 --topic CDC-CDC_OPER-FPL_OPER
```

`Consume the message`

```shell
./kafka-console-consumer.sh --bootstrap-server 127.0.0.1:9093 --from-beginning --topic RAW-FIXM42-OPER.v3 --property print.timestamp=true --property print.value=false --property print.offset=true
```

`describe the topic`

```shell
./kafka-topics.sh --bootstrap-server 127.0.0.1:9093 --topic RAW-FIXM42-OPER.v2 --describe
```

`Check  the consumer group`

```shell
./kafka-consumer-groups.sh --bootstrap-server 127.0.0.1:9093 --group=moge --describe
```

`Reset offset`

```shell
./kafka-consumer-groups.sh --bootstrap-server=127.0.0.1:9093 --group=moge --topic=RAW-FIXM42-OPER.v2 --execute --reset-offsets --to-earliest
```

# Add a container into an existing network
By default a container cannot access the other container. In order for that to get to work, they are needed to be within the same network.

`Check the network where a container belong`
```shell
docker network inspect -f '{{range .Containers}}{{.Name}} {{end}}' k8s_default
```

`Add a container into a network`

```shell
docker network connect k8s_default kafkarest
```

```shell
docker network inspect -f '{{range .Containers}}{{.Name}} {{end}}' [network]
```

# Minimize the docker image size
`Strategy`

> Use the multi stage docker build

* First stage is to build binary from the source codes

* Second stage is to get the binary from first stage then put it in a image built from relatively small image, such as `scratch`

`Materialize`

```shell
# syntax=docker/dockerfile:1
FROM golang:1.18-buster AS builder
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download


COPY *.go ./


RUN go build -o /kafkist_consumer
# Final Stage - Stage 2
FROM gcr.io/distroless/base-debian10 as baseImage
WORKDIR /app

COPY --from=builder /kafkist_consumer ./kafkist_consumer
COPY config.yaml ./config.yaml

ENV CONFIG_FILE=/app/config.yaml

ENTRYPOINT ["/app/kafkist_consumer"]
EXPOSE 8872
```

## go app binary

The first build stage is started from `FROM golang:1.18-buster AS builder`. The goal is to create the binary from `*.go` files.
The dependencies are resolved through the `go module`. It is the reason why there are steps to copy both `go.mod` and `go.sum`. 
The `RUN go mod download` is to download all the dependencies. The following step is copying the source codes from local 
to the docker context at current working directory. Remember `WORKDIR /app` just after the first line
Finally, `RUN go build -o /kafkist_consumer` is building the binary from source codes and output it as `/kafkist_consumer`
After this step, under the `/app` there will be `kafkist_consumer` binary file.

## go app deployment
The second stage is started from `FROM gcr.io/distroless/base-debian10 as baseImage`. The goal is to copy the binary created from first stage
into the `/app` directory of current folder. This step is invoked by the line, `COPY --from=builder /kafkist_consumer ./kafkist_consumer`
`COPY config.yaml ./config.yaml` is to copy the config.yaml from local working directory (developer machine) into 
`/app` directory
`ENV CONFIG_FILE=/app/config.yaml` is to create an environment variable required by the app
`ENTRYPOINT ["/app/kafkist_consumer"]` is to define the entry point for launching application
`EXPOSE 8872` is to expose port 8872 so that it is visible from outside of container.

## Build

```shell
docker build -t samutup/kafkarest:1.0.0 --no-cache . -f DockerFile
```
Once it is done check the size,

```shell
docker images
```
It will show,

```text
EPOSITORY                                  TAG           IMAGE ID       CREATED          SIZE
samutup/kafkarest                           1.0.0         8be48b3ad8f8   15 minutes ago   33.3MB
```
That is not bad, `33MB`

## Launch the app

It is achieved by executing the `docker run`. Or using the `docker-compose up -d` command.
Let's do it in `docker compose` way.

`Prepare docker compose configuration file`

```yaml
---
version: '2'
services:
  kafkarest:
    image: samutup/kafkarest:1.0.0
    hostname: kafkarest
    container_name: kafkarest
    ports:
      - "8872:8872"
```

`run`

```shell
docker-compose up -d
```
Successful message

```text
Creating kafkarest ... done
```

`tail the log`

```shell
docker-compose logs --follow
```
Successful message

```text
Attaching to kafkarest--follow
kafkarest    | 2022/06/26 07:06:00 kafkist_consumer.go:304: starting server at port 8872 
```

`stop`

```shell
docker-compose stop
```
`remove container`

```shell
docker-compose rm
```