# Skywalker

Skywalker walks above the cloud.

## What is Skywalker project?

Skywalker is cloud service managing and controlling all edge devices, processing IOT data, etc. It also manages IOT users and provides friendly APIs to front-end.

Currently, Skywalker includes:

- `Broadcast`: forwards video streams from edge devices.

## How to run?

All sub-processes are executed through sub commands under `skywalker` command.

Make sure you have the following tools installed:

- `Go`
- `GNU make`
- `golangci-lint` (Optional)
- `Docker`
- `Docker-compose` (Optional)
- `mosquitto` (Optional, can be on Docker)

### Run `broadcast`

```bash
$ make
```
