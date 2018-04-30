# message-cannon
Consume messages from message systems (RabbitMQ, NATS, Kafka) and send to other applications.


[![Release](https://img.shields.io/github/release/leandro-lugaresi/message-cannon.svg?style=flat-square)](https://github.com/leandro-lugaresi/message-cannon/releases/latest)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)](LICENSE.md)
[![Build Status](https://travis-ci.org/leandro-lugaresi/message-cannon.svg?branch=master&style=flat-square)](https://travis-ci.org/leandro-lugaresi/message-cannon)
[![Coverage Status](https://img.shields.io/codecov/c/github/leandro-lugaresi/message-cannon/master.svg?style=flat-square)](https://codecov.io/gh/leandro-lugaresi/message-cannon)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/leandro-lugaresi/message-cannon)
[![Go Report Card](https://goreportcard.com/badge/github.com/leandro-lugaresi/message-cannon?style=flat-square)](https://goreportcard.com/report/github.com/leandro-lugaresi/message-cannon)
[![Say Thanks!](https://img.shields.io/badge/Say%20Thanks-!-1EAEDB.svg)](https://saythanks.io/to/leandro-lugaresi)
[![Powered By: GoReleaser](https://img.shields.io/badge/powered%20by-goreleaser-green.svg?style=flat-square)](https://github.com/goreleaser)

---

# Motivation
If you already tried to use some long running code with PHP you probably notice some problems like:
- Doctrine connection closed;
- Entities outdated;
- A large amount of ram used by consumers sleeping;
- The rabbitMQ connection is dead but the consumer still running (supervisor think it's alive).

message-cannon is a binary used to solve this problem we faced in PHP projects.
The idea is to run the consumers in a go binary and send the messages to callbacks using `runners`.

# Installation

## Docker
The simple way to run message-cannon is to run the docker image for it. This is an example of docker-compose running the message-cannon:
```
  cannon:
    image: leandrolugaresi/message-cannon:latest
    volumes:
      - ./app/config/cannon.yaml:/.cannon.yaml
    command: launch --config=.cannon.yaml
    links:
      - rabbitmq
      - my-app
```

## Manual
You can get the binaries and use it. Just go to [releases](https://github.com/leandro-lugaresi/message-cannon/releases) and download the newest binary for your SO (deb, rpm, snap are also available)

# Usage

## CLI Commands

We have only one command: `message-cannon launch` will open one config file and start all the consumers availlable.

## Runners

### Command

This is the first runner developed and it will open an executable(PHP, python, ruby, bash) and send the message using the STDIN. The executable will receive the message, process and return an exit code used to know how to handle the message.

> WARNING: This method is only possible if your number of messages are really low! Open system process (ie: PHP runtime) for every single message will cost a lot of resources. If a number of messages are bigger try solve your issues with PHP (good luck), change to HTTP runner or rewrite it to solve your problems.

#### Example

```yml
consumers:
  upload_picture:
    ...
    runner:
      type: command
      options:
        path: "bin/app-console message:cannon"
        args: ["some-param","other-param","fooo"]
```

### HTTP

This is the best choice available. The runner will send the message using a POST request with the message content as the request body.
The runner will handle the messages depending on the request status code and content.

#### Headers
The message-cannon send some headers when sending one message, this headers can be from the message or from the `headers` option of the consumer.runner config.
The rabbitMQ consumer will send this headers:

Header name | Type | From
----------- | -----|-----
`Content-Type` | string | message contentType header
`Content-Encoding` | string | message contentEncoding header
`Correlation-Id` | string | message CorrelationId param
`Message-Id` | string | message MessageId param
`Message-Deaths` | int | number of times the message received a NACK (this is useful with retries using dead-letters)


#### Responses

status code | default | override
----------- | ------- | --------
5xx | `4`: ExitNACKRequeue | option `return-on-5xx`
4xx | `5`: ExitRetry | N/A
2xx with ignore-output  | `0`: ACK | N/A
2xx without ignore-output | N/A | expect a json response with a `response-code` field

#### Example

Config file:
```yml
consumers:
  upload_picture:
    ...
    runner:
      type: http
      ignore-output: false
      options:
        url: "https://localhost/receive-messages/upload-picture"
        return-on-5xx: 3 # ExitNACK
        headers:
          Authorization: Basic YWxhZGRpbjpvcGVuc2VzYW1l
          Content-Type: application/json #override the Content-Type from message
```

Server reponse:

```json
{"response-code": 4, "error": "some nasty error here", "trace": "some trace as string"}
```

## Return codes:

We create some constants to represent some operations available to messages, every runner has some way to get this information from the callbacks.

Return code | name | rabbitMQ
----------- | ------- | --------
`0`| ACK | Ack
`1`| ExitFailed | Reject[requeue: false]
`3`| ExitNACK | Nack[requeue: `false`]
`4`| ExitNACKRequeue | Nack[requeue: `true`]
`5`| ExitRetry | Nack[requeue: `true`]
`-1`| ExitTimeout | Nack[requeue: `true`]
`-`| invalid code | Reject[requeue: true]

## Example of config file

You can see an example of config file [here](cannon.yml.dist)

### Environment variables

You can use environment variables inside the yaml file. The sintax is like the syntax used inside the docker-compose file.
To use a required variable just use like this: `${ENV_NAME}` and to put an default value you can use: `${ENV_NAME:=some-value}`. Ie:
```yaml
connections:
    default:
      dsn: "amqp://${RABBITMQ_USER:=guest}:${RABBITMQ_PASSWORD}@${RABBITMQ_HOST:=rabbitmq}:${RABBITMQ_PORT:=5672}${RABBITMQ_VHOST:=/}"

```
 will use the default values when make the connection dsn.

# License
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fleandro-lugaresi%2Fmessage-cannon.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fleandro-lugaresi%2Fmessage-cannon?ref=badge_large)

---

This project adheres to the Contributor Covenant [code of conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.
We appreciate your contribution. Please refer to our [contributing guidelines](CONTRIBUTING.md) for further information.