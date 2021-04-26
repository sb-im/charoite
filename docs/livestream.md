# Livestream

## Stream inputs

device | transport protocol | codec | direction | address | remark
------ | ------------------ | ----- | --------- | ------- | ------
Drone | RTP | H.264 | push | udp://0.0.0.0:5004 | |
Hikvision webcam | RTSP | H.264 | pull | rtsp://admin:12345@192.0.0.64:554/h264/ch1/sub/av_stream | Use sub channel instead of main one
