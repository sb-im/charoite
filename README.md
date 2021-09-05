# Skywalker

Skywalker walks above the cloud.

## What is Skywalker project?

Skywalker is a simple and efficient WebRTC SFU server built upon [Pion WebRTC](https://github.com/pion/webrtc).

Currently, Skywalker includes:

- `Broadcast`: forwards video streams from edge devices.
- `turn`: TURN server.

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
