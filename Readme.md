# Go, Kubernetes, Twitter and Kafka
## Overview

This repository is a sample project to work on the `twitter api` and `kafka` in `kubernetes` environment

* Stream tweet then push into kafka topic
* Deploy to kubernetes

## Containerization

```shell
docker build -t samutup/sam-twitter:1.0.0 --no-cache -f DockerFile .
```

## Helm Install

```shell
helm upgrade --install sam-twitter sam-twitter --debug
```