package livestream

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/deepch/vdk/av"
	"github.com/deepch/vdk/codec/h264parser"
	"github.com/deepch/vdk/format/rtspv2"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/rs/zerolog"
)

// consumeRTSP connects to an RTSP URL and pulls media.
// Convert H264 to Annex-B, then write to videoTrack which sends to all PeerConnections.
func consumeRTSP(ctx context.Context, address string, videoTrack webrtc.TrackLocal, logger *zerolog.Logger) error {
	videoTrackSample := videoTrack.(*webrtc.TrackLocalStaticSample)

	annexbNALUStartCode := func() []byte { return []byte{0x00, 0x00, 0x00, 0x01} }

	// Use a loop in case RTSP stream is stopped os we can retry.
	for {
		logger.Info().Str("address", address).Msg("dialing RTSP server")
		session, err := rtspv2.Dial(rtspv2.RTSPClientOptions{
			URL:              address,
			DialTimeout:      3 * time.Second,
			ReadWriteTimeout: 3 * time.Second,
			DisableAudio:     true,
		})
		if err != nil {
			return fmt.Errorf("rtsp dial error: %w", err)
		}

		codecs := session.CodecData
		for i, t := range codecs {
			logger.Info().Int("i", i).Str("type", t.Type().String()).Msg("stream codec")
		}
		if codecs[0].Type() != av.H264 {
			return fmt.Errorf("wrong codec type: %s. RTSP feed must begin with a H264 codec", codecs[0].Type())
		}
		if len(codecs) != 1 {
			logger.Info().Msg("ignoring all but the first stream")
		}

		var previousTime time.Duration
		for {
			pkt := <-session.OutgoingPacketQueue

			if pkt.Idx != 0 {
				// audio or other stream, skip it
				continue
			}

			pkt.Data = pkt.Data[4:]

			// For every key-frame pre-pend the SPS and PPS
			if pkt.IsKeyFrame {
				pkt.Data = append(annexbNALUStartCode(), pkt.Data...)
				pkt.Data = append(codecs[0].(h264parser.CodecData).PPS(), pkt.Data...)
				pkt.Data = append(annexbNALUStartCode(), pkt.Data...)
				pkt.Data = append(codecs[0].(h264parser.CodecData).SPS(), pkt.Data...)
				pkt.Data = append(annexbNALUStartCode(), pkt.Data...)
			}

			bufferDuration := pkt.Time - previousTime
			previousTime = pkt.Time

			if err = videoTrackSample.WriteSample(media.Sample{Data: pkt.Data, Duration: bufferDuration}); err != nil && err != io.ErrClosedPipe {
				return fmt.Errorf("could not write videoTrackSample: %w", err)
			}

			select {
			case <-ctx.Done():
				logger.Info().Str("err", ctx.Err().Error()).Msg("context is done, exiting live streaming")
				return nil
			default:
			}
		}
	}
}
