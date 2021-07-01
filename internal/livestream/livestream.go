package livestream

import (
	"context"
	"strconv"

	mqttclient "github.com/SB-IM/mqtt-client"
	pb "github.com/SB-IM/pb/signal"
	"github.com/rs/zerolog/log"
)

// Livestream broadcasts live stream, it either consumes RTP stream from a RTP client,
// or pulls RTSP stream from a RTSP server. The underlining transportation is WebRTC.
type Livestream interface {
	Publish() error
	Meta() *pb.Meta
}

func NewDronePublisher(ctx context.Context, configOptions *PublisherConfigOptions) Livestream {
	return &publisher{
		meta: &pb.Meta{
			Id:          configOptions.UUID,
			TrackSource: pb.TrackSource_DRONE,
		},
		config: broadcastConfigOptions{
			configOptions.MQTTClientConfigOptions,
			configOptions.WebRTCConfigOptions,
			configOptions.ConsumeStreamOnDemand,
		},
		client:      mqttclient.FromContext(ctx),
		createTrack: videoTrackRTP,
		streamSource: func() string {
			return configOptions.Host + ":" + strconv.Itoa(configOptions.Port)
		},
		liveStream: rtpListener,
		logger:     *log.Ctx(ctx),
	}
}

func NewDeportPublisher(ctx context.Context, configOptions *PublisherConfigOptions) Livestream {
	// Default deport stream source is rtsp.
	publisher := &publisher{
		meta: &pb.Meta{
			Id:          configOptions.UUID,
			TrackSource: pb.TrackSource_MONITOR,
		},
		config: broadcastConfigOptions{
			configOptions.MQTTClientConfigOptions,
			configOptions.WebRTCConfigOptions,
			configOptions.ConsumeStreamOnDemand,
		},
		client:      mqttclient.FromContext(ctx),
		createTrack: videoTrackSample,
		streamSource: func() string {
			return configOptions.Addr
		},
		liveStream: consumeRTSP,
		logger:     *log.Ctx(ctx),
	}

	// If it's rtp stream source.
	if configOptions.Protocol == protocolRTP {
		publisher.createTrack = videoTrackRTP
		publisher.streamSource = func() string {
			return configOptions.Host + ":" + strconv.Itoa(configOptions.Port)
		}
		publisher.liveStream = rtpListener
	}

	return publisher
}
