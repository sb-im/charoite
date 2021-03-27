package livestream

import (
	"bytes"
	"context"
	"strconv"

	mqttclient "github.com/SB-IM/mqtt-client"
	pb "github.com/SB-IM/pb/signal"
	"github.com/rs/zerolog/log"
)

var id string

func init() {
	mID, err := machineID()
	if err != nil || mID == nil {
		panic(err)
	}
	id = string(bytes.TrimSuffix(mID, []byte("\n")))
}

// Livestream broadcasts live stream, it either consumes RTP stream from a RTP client,
// or pulls RTSP stream from a RTSP server. The underlining transportation is WebRTC.
type Livestream interface {
	Publish() error
	ID() string
	TrackSource() pb.TrackSource
}

func NewRTPPublisher(ctx context.Context, configOptions RTPBroadcastConfigOptions) Livestream {
	return &publisher{
		id:          id,
		trackSource: pb.TrackSource_DRONE,
		config: broadcastConfigOptions{
			configOptions.TopicConfigOptions,
			configOptions.WebRTCConfigOptions,
		},
		client:      mqttclient.FromContext(ctx),
		createTrack: videoTrackRTP,
		streamSource: func() string {
			return configOptions.RTPHost + ":" + strconv.Itoa(configOptions.RTPPort)
		},
		liveStream: rtpListener,
		logger:     *log.Ctx(ctx),
	}
}

func NewRTSPPublisher(ctx context.Context, configOptions RTSPBroadcastConfigOptions) Livestream {
	return &publisher{
		id:          id,
		trackSource: pb.TrackSource_MONITOR,
		config: broadcastConfigOptions{
			configOptions.TopicConfigOptions,
			configOptions.WebRTCConfigOptions,
		},
		client:      mqttclient.FromContext(ctx),
		createTrack: videoTrackSample,
		streamSource: func() string {
			return configOptions.RTSPAddr
		},
		liveStream: consumeRTSP,
		logger:     *log.Ctx(ctx),
	}
}
