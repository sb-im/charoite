package livestream

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog"
)

// rtpListener starts a UDP listener and consumes data stream.
func rtpListener(address string, videoTrack webrtc.TrackLocal, logger zerolog.Logger) error {
	videoTrackSample := videoTrack.(*webrtc.TrackLocalStaticRTP)

	host, port := parseRTPAddress(address)

	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(host), Port: port})
	if err != nil {
		return fmt.Errorf("listen UDP: %w", err)
	}
	logger.Info().Str("host", host).Int("port", port).Msg("UDP server started")
	defer listener.Close()

	inboundRTPPacket := make([]byte, 1600) // UDP MTU
	for {
		n, _, err := listener.ReadFrom(inboundRTPPacket)
		if err != nil {
			return fmt.Errorf("error during read: %w", err)
		}

		if _, err = videoTrackSample.Write(inboundRTPPacket[:n]); err != nil {
			return fmt.Errorf("could not write videoTrackSample: %w", err)
		}
	}
}

func parseRTPAddress(address string) (string, int) {
	ss := strings.Split(address, ":")
	if len(ss) != 2 {
		panic("invalid address")
	}

	port, err := strconv.Atoi(ss[1])
	if err != nil {
		panic(err)
	}

	return ss[0], port
}
