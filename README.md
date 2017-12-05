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

If you already tried to use some long running code with PHP you probably notice some problems like:
- Doctrine connection closed;
- Entities outdated;
- A large amount of ram used by consumers sleeping;
- The process is dead but still running (supervisor think it's alive). 

message-cannon is a binary used to solve this problem we faced in PHP projects.
The idea is to run the consumers in a go binary and send the messages to callbacks using `runners`.

## Runners

### Command

This is the first runner developed and it will open an executable(PHP, python, ruby, bash) and send the message using the STDIN. The executable will receive the message, process and return an exit code used to know how to handle the message.

> WARNING: This method is only possible if your number of messages are really low! Open system process (ie: PHP runtime) for every single message will cost a lot of resources. If a number of messages are bigger try solve your issues with PHP (good luck), change to HTTP runner or rewrite in a better language.

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

status code | default | override
----------- | ------- | --------
5xx | `4`: ExitNACKRequeue | option `return-on-5xx`
4xx | `5`: ExitRetry | N/A
2xx  | `0`: ACK | if json response has a `response-code` field

#### Example

Config file: 
```yml
consumers:
  upload_picture:
    ...
    runner:
      type: http
      options:
        url: "https://localhost/receive-messages/upload-picture"
        return-on-5xx: 3 # ExitNACK
        headers:
          Authorization: Basic YWxhZGRpbjpvcGVuc2VzYW1l
          Content-Type: application/json
```

Server reponse:

```json
{"response-code": 4, "error": "some nasty error here", "trace": "some trace as string"}
```

## Return codes:

We create some constants to represent some operations available to messages, every runner has some way to get this information from the callbacks.

-  `0`: ACK
-  `1`: ExitFailed
-  `3`: ExitNACK
-  `4`: ExitNACKRequeue
-  `5`: ExitRetry

## Example of config file

You can see an example of config file [here](cannon.yml.dist)

---

This project adheres to the Contributor Covenant [code of conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.
We appreciate your contribution. Please refer to our [contributing guidelines](CONTRIBUTING.md) for further information.