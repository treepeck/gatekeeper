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

See [judo](https://github.com/treepeck/judo) to learn how to set up a local
development environment.

## License

Copyright (c) 2025 Artem Bielikov

This project is available under the Mozilla Public License, v. 2.0.<br/>
See the [LICENSE](LICENSE) file for details.