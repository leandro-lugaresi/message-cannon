# message-cannon
Consume messages from message systems (RabbitMQ, NATS, Kafka) and pipe to other applications.


[![Release](https://img.shields.io/github/release/leandro-lugaresi/message-cannon.svg?style=flat-square)](https://github.com/leandro-lugaresi/message-cannon/releases/latest)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)](LICENSE.md)
[![Build Status](https://travis-ci.org/leandro-lugaresi/message-cannon.svg?branch=master&style=flat-square)](https://travis-ci.org/leandro-lugaresi/message-cannon)
[![Coverage Status](https://img.shields.io/codecov/c/github/leandro-lugaresi/message-cannon/master.svg?style=flat-square)](https://codecov.io/gh/leandro-lugaresi/message-cannon)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/leandro-lugaresi/message-cannon)
[![Go Report Card](https://goreportcard.com/badge/github.com/leandro-lugaresi/message-cannon?style=flat-square)](https://goreportcard.com/report/github.com/leandro-lugaresi/message-cannon)
[![Say Thanks!](https://img.shields.io/badge/Say%20Thanks-!-1EAEDB.svg)](https://saythanks.io/to/leandro-lugaresi)
[![Powered By: GoReleaser](https://img.shields.io/badge/powered%20by-goreleaser-green.svg?style=flat-square)](https://github.com/goreleaser)

---

If you already try to use some long running code with PHP you probably notice some problems:
- Doctrine connection closed;
- Entities outdated;
- A large amount of ram used by consumers sleeping;
- Supervisord to solve problems with stability. 

message-cannon is a binary used to solve this problem in PHP(legacy) systems.
The idea is run the consumers in go and send the messages to callbacks using `runners`.

## Runners
### Command
This is the first runner developed and it will open an executable(php, python, ruby, bash) and send the message using the stdin. The executable will receive the message, process and return an exit code used to know how to handle the message.
Status codes:
-  `0`: ACK
-  `1`: ExitFailed
-  `3`: ExitNACK
-  `4`: ExitNACKRequeue
-  `5`: ExitRetry
> WARNING: This method is only possible if your number of messages are low! Open system process for every single message will cost some resources. If a number of messages are bigger try solve your issues with php (good luck) or rewrite in a better language.

---

This project adheres to the Contributor Covenant [code of conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.
We appreciate your contribution. Please refer to our [contributing guidelines](CONTRIBUTING.md) for further information.