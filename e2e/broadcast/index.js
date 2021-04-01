(() => {
  let pc = new RTCPeerConnection();
  pc.ontrack = function (event) {
    var el = document.createElement(event.track.kind);
    el.srcObject = event.streams[0];
    el.autoplay = true;
    el.controls = true;

    document.getElementById("remoteVideos").appendChild(el);
  };

  pc.addTransceiver("video");

  function dial() {
    const conn = new WebSocket(`ws://localhost:8080/ws/webrtc`);

    conn.addEventListener("close", (ev) => {
      console.log(
        `WebSocket Disconnected code: ${ev.code}, reason: ${ev.reason}`
      );
    });
    conn.addEventListener("open", (ev) => {
      console.info("websocket connected");

      for (let i = 0; i < 2; i++) {
        pc.createOffer()
          .then((offer) => {
            pc.setLocalDescription(offer);

            // Standard message schema.
            msg = {
              id: "fa955cc6881b4b45b49ffbf2d81e7223",
              track_source: i,
              sdp: offer,
            };

            str = JSON.stringify(msg);
            console.log(`sent offer ${i} ${str}`);

            conn.send(str);
          })
          .catch(alert);
      }
    });

    // This is where we handle messages received.
    conn.addEventListener("message", (ev) => {
      // Receiving SDP answer
      console.log(`Received SDP answer from server: ${ev.data}`);

      // .then((res) => res.json())
      // .then((res) => pc.setRemoteDescription(res))
      try {
        answer = JSON.parse(ev.data);
        pc.setRemoteDescription(answer.sdp);
      } catch (err) {
        console.error(err);
      }
    });
  }
  dial();
})();
