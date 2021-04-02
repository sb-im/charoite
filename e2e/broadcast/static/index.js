(() => {
  for (let i = 0; i < 2; i++) {
    let pc = new RTCPeerConnection({
      iceServers: [
        {
          urls: "stun:stun.l.google.com:19302",
        },
      ],
    });

    let log = (msg) => {
      document.getElementById(`log${i}`).innerHTML += msg + "<br>";
    };

    pc.ontrack = function (event) {
      var el = document.createElement(event.track.kind);
      el.srcObject = event.streams[0];
      el.autoplay = true;
      el.controls = true;

      document.getElementById("remoteVideos").appendChild(el);
    };

    pc.oniceconnectionstatechange = (e) => log(pc.iceConnectionState);

    pc.addTransceiver("video");

    pc.createOffer()
      .then((offer) => {
        pc.setLocalDescription(offer);
        console.log(`Sending offer ${i}: ${offer}`);

        sdp = {
          id: "fa955cc6881b4b45b49ffbf2d81e7223",
          track_source: i,
          sdp: offer,
        };

        return fetch("http://localhost:8080/v1/broadcast/signal", {
          method: "post",
          headers: {
            Accept: "application/json",
            "Content-Type": "application/json",
            'Access-Control-Allow-Origin':'*'
          },
          mode: 'no-cors',
          body: JSON.stringify(sdp),
        });
      })
      .then((res) => res.json())
      .then((res) => {
        console.log(`Received answer ${i}: ${res}`);

        pc.setRemoteDescription(res);
      })
      .catch(log);
  }
})();
