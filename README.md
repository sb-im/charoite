# Sphinx

Sphinx runs at the edge gateway.

## What is Sphinx project?

Sphinx is a service running on NVIDIA NANO of edge device. It ingests and publishes video stream to cloud service [skywalker](https://github.com/sb-im/skywalker).

The core technology Sphinx built upon is [Pion WebRTC](https://github.com/pion/webrtc).

Currently, Sphinx includes:

- Livestream: Drone video and webcam monitor video forwarding through WebRTC.
- Hookstream: WebRTC ICE state hook.

### Supported ingestion protocols

- RTP
- RTSP
- RTMP

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
