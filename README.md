# mesh-crawler

## Overview

The mesh crawler is an HTTP server which o in that implements 4 endpoints. Its main purpose is to test and tryout mesh functionalities and behaviours.
The implemented endpoints are the following:

* /metrics - exposes prometheus metrics
* /health - healthcheck
* /identify - identifies the deployment with name and zone
* /crawl - is the main endpoint that crawls through the mesh

## Building Docker

In order to build the docker image yourself, please use the following command:

* For multiarch:

```bash
docker buildx build --platform linux/amd64,linux/arm64 -t <repository/tag-name>:<tag> .
```

* For single arch (same arch as your OS):

```bash
docker build -t <repository/tag-name>:<tag> .
```

## Building binary

In order to build the binary yourself, please use the following command:

* For multiarch:

```bash
GOARCH=amd64 GOOS=linux go build -o server
GOARCH=arm64 GOOS=linux go build -o server_arm
```

* For single arch (same arch as your OS):

```bash
GOARCH=amd64 GOOS=linux go build -o server
```

## Deployment

In the deployment section there is a simple kubernetes deployment that can be used for reference.

## Usage

In order to run the application locally, you need to set the following environment variables:

* SERVICE_NAME
* MESH_ZONE
* PORT (default port is 8080)

Run with docker:

```bash
docker run -i -t -p 8080 mesh-crawler-local:latest sh
```

Run with with(out) binary:

```bash
go run main.go
./server
```

In the example section there are 2 files that represent an example payload that can be send.
