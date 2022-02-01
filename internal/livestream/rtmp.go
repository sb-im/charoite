package livestream

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
	flvtag "github.com/yutopp/go-flv/tag"
	"github.com/yutopp/go-rtmp"
	rtmpmsg "github.com/yutopp/go-rtmp/message"
)

const (
	headerLengthField = 4
	spsId             = 0x67
	ppsId             = 0x68
)

func consumeRTMP(ctx context.Context, address string, videoTrack webrtc.TrackLocal, logger *zerolog.Logger) error {
	l, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen tcp at %s: %w", address, err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			logger.Err(err).Msg("could not close listener")
		}
	}()

	s := rtmp.NewServer(&rtmp.ServerConfig{
		OnConnect: func(conn net.Conn) (io.ReadWriteCloser, *rtmp.ConnConfig) {
			return conn, &rtmp.ConnConfig{
				Handler: &handler{
					videoTrack: videoTrack,
					logger:     logger,
				},
				ControlState: rtmp.StreamControlStateConfig{
					DefaultBandwidthWindowSize: 6 * 1024 * 1024 / 8,
				},
				Logger: logrus.StandardLogger(),
			}
		},
	})
	defer func() {
		if err := s.Close(); err != nil {
			logger.Err(err).Msg("could not close rtmp server")
		}
	}()

	logger.Info().Str("address", address).Msg("starting rtmp server")

	return s.Serve(l)
}

type handler struct {
	rtmp.DefaultHandler

	videoTrack webrtc.TrackLocal

	sps []byte
	pps []byte

	logger *zerolog.Logger
}

func (h *handler) OnConnect(timestamp uint32, _ *rtmpmsg.NetConnectionConnect) error {
	h.logger.Info().Msg("client is connecting")
	return nil
}

func (h *handler) OnCreateStream(timestamp uint32, _ *rtmpmsg.NetConnectionCreateStream) error {
	h.logger.Info().Msg("client is creating stream")
	return nil
}

func (h *handler) OnPublish(ctx *rtmp.StreamContext, timestamp uint32, cmd *rtmpmsg.NetStreamPublish) error {
	h.logger.Info().Msg("client is publishing stream")

	if cmd.PublishingName == "" {
		return errors.New("PublishingName is empty")
	}

	return nil
}

func (h *handler) OnVideo(timestamp uint32, payload io.Reader) error {
	var video flvtag.VideoData
	if err := flvtag.DecodeVideoData(payload, &video); err != nil {
		return err
	}

	data := new(bytes.Buffer)
	if _, err := io.Copy(data, video.Data); err != nil {
		return err
	}

	hasSpsPps := false
	outBuf := []byte{}
	videoBuffer := data.Bytes()

	switch video.AVCPacketType {
	case flvtag.AVCPacketTypeNALU:
		for offset := 0; offset < len(videoBuffer); {
			bufferLength := int(binary.BigEndian.Uint32(videoBuffer[offset : offset+headerLengthField]))
			if offset+bufferLength >= len(videoBuffer) {
				break
			}

			offset += headerLengthField

			if videoBuffer[offset] == spsId {
				hasSpsPps = true
				h.sps = append(annexBPrefix(), videoBuffer[offset:offset+bufferLength]...)
			} else if videoBuffer[offset] == ppsId {
				hasSpsPps = true
				h.pps = append(annexBPrefix(), videoBuffer[offset:offset+bufferLength]...)
			}

			outBuf = append(outBuf, annexBPrefix()...)
			outBuf = append(outBuf, videoBuffer[offset:offset+bufferLength]...)

			offset += bufferLength
		}
	case flvtag.AVCPacketTypeSequenceHeader:
		const spsCountOffset = 5
		spsCount := videoBuffer[spsCountOffset] & 0x1F
		offset := 6
		h.sps = []byte{}
		for i := 0; i < int(spsCount); i++ {
			spsLen := binary.BigEndian.Uint16(videoBuffer[offset : offset+2])
			offset += 2
			if videoBuffer[offset] != spsId {
				return errors.New("failed to parse SPS")
			}
			h.sps = append(h.sps, annexBPrefix()...)
			h.sps = append(h.sps, videoBuffer[offset:offset+int(spsLen)]...)
			offset += int(spsLen)
		}
		ppsCount := videoBuffer[offset]
		offset++
		for i := 0; i < int(ppsCount); i++ {
			ppsLen := binary.BigEndian.Uint16(videoBuffer[offset : offset+2])
			offset += 2
			if videoBuffer[offset] != ppsId {
				return errors.New("failed to parse PPS")
			}
			h.sps = append(h.sps, annexBPrefix()...)
			h.sps = append(h.sps, videoBuffer[offset:offset+int(ppsLen)]...)
			offset += int(ppsLen)
		}
		return nil
	default:
		h.logger.Warn().Uint8("AVCPacketType", uint8(video.AVCPacketType)).Msg("unknown type")
		return nil
	}

	// We have an unadorned keyframe, append SPS/PPS
	if video.FrameType == flvtag.FrameTypeKeyFrame && !hasSpsPps {
		outBuf = append(append(h.sps, h.pps...), outBuf...)
	}

	return h.videoTrack.(*webrtc.TrackLocalStaticSample).WriteSample(media.Sample{
		Data:     outBuf,
		Duration: time.Second / 30,
	})
}

func (h *handler) OnClose() {
	h.logger.Info().Msg("closing client connection")
}

func annexBPrefix() []byte {
	return []byte{0x00, 0x00, 0x00, 0x01}
}
