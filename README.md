# OpenTelemetry Tracing Example

This repository demonstrates the integration of [OpenTelemetry](opentelemetry) for [tracing](tracing) across two services: one written in Python and the other in Go. It serves as a practical example of implementing distributed tracing to monitor and troubleshoot the interactions between services.

## Overview

- **Python Service**: Exposes a simple API for adding numbers together
- **Go Service**: Uses the python service to sum two numbers

The primary goal is to illustrate how OpenTelemetry traces the requests flowing between these services, providing insights into the system's behavior and performance.

## Getting Started

### Prerequisites

- [Docker](docker)
- [Docker Compose](docker-compose)


### Running the Services
```bash
    docker-compose -f compose.yaml up -d --build
```


### Stopping the Services
```bash
    docker-compose -f compose.yaml down
```


## Golang service

The go service exposes an endpoint `POST /sum` which accepts two fields `a` and `b`.

To test the service feel free to run:
```Bash
curl -v --data "a=50&b=300" localhost:8080/sum
```

## Traces

The traces can be accessed through jaeger default host: http://localhost:16686


## Metrics

Metrics can be accessed through prometheus client on: http://localhost:9090


[opentelemetry]: https://opentelemetry.io/
[tracing]: https://opentelemetry.io/docs/concepts/signals/traces/
[docker]: https://www.docker.com/
[docker-compose]: https://docs.docker.com/compose/
