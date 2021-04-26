# Sphinx

Sphinx at the edge gateway.

## What is Sphinx project?

Sphinx is a service consists of multiple processes running at NVIDIA NANO of edge device. It bridges every edge process component and cloud service, things like WebRTC videos forwarding, MQTT messages proxy, etc.

Currently, Sphinx includes:

- Livestream: Drone video and webcam monitor video forwarding through WebRTC.

## How to run?

All sub-processes are executed through sub commands under `sphinx` command.

Make sure you have the following tools installed:

- `Go`
- `GNU make`
- `golangci-lint` (Optional)
- `Docker`
- `Docker-compose` (Optional)
- `mosquitto` (Optional, can be on Docker)

## Run `livestream`

```bash
$ make
```
