package livestream

import (
	"fmt"
	"net"

	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog"
)

// rtpListener starts a UDP listener and consumes data stream.
func rtpListener(address string, videoTrack webrtc.TrackLocal, logger *zerolog.Logger) error {
	videoTrackSample := videoTrack.(*webrtc.TrackLocalStaticRTP)

	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return fmt.Errorf("could not resolve address of %s into udp address: %w", address, err)
	}

	listener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("listen UDP: %w", err)
	}
	defer listener.Close()
	logger.Info().Str("address", udpAddr.String()).Msg("UDP server started")

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
