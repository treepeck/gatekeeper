[![License: MPL 2.0](https://img.shields.io/badge/License-MPL%202.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)
[![Go Reference](https://pkg.go.dev/badge/github.com/treepeck/gatekeeper.svg)](https://pkg.go.dev/github.com/treepeck/gatekeeper)

The main idea is to decouple core application logic from WebSocket connection<br/>
management, enabling multiple modular instances to publish events to a central server.

Gatekeeper doesn't know who processes events and how, or whether anyone processes<br/>
them at all. Its sole responsibility is to accept and forward events from a large<br/>
number of concurrently connected clients.

The central server performs the heavy lifting: updating its internal state,<br/>
optionally accessing the database, and publishing responses back to a RabbitMQ<br/>
exchange.  These responses are then consumed by the corresponding room, maintained<br/>
by the Gatekeeper.

## Architecture

![Architecture](./doc/arch.png)

## Local installation

First install the Go compiler version 1.24.4 or newer (see https://go.dev/dl).

You will also need a RabbitMQ server instance up and running <br/>
(see https://www.rabbitmq.com/docs/download).

Once everything is installed, clone this repository:

```
git clone https://github.com/treepeck/gatekeeper
cd gatekeeper
```

Gatekeeper uses environment variables to specify the RabbitMQ connection URL and<br/>
the port on which it will listen for WebSocket connections. Here is an example of how<br/>
to create and populate a `.env` file in the gatekeeper folder:

```
touch .env
echo "RABBITMQ_URL=amqp://guest:guest@localhost:5672/" >> .env
echo "ADDR=:3503" >> .env
```

Run this command to start the server:

```
go run cmd/gatekeeper/main.go
```

## License

Copyright (c) 2025 Artem Bielikov

This project is available under the Mozilla Public License, v. 2.0.<br/>
See the [LICENSE](LICENSE) file for details.