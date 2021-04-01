package broadcast

import (
	"fmt"

	pb "github.com/SB-IM/pb/signal"
	"github.com/rs/zerolog"
)

type machineID string

type middleFunc func(id machineID, trackSource pb.TrackSource, subscriber *subscriberChans)

type subscriberChans struct {
	offerChan  chan *pb.SessionDescription
	answerChan chan *pb.SessionDescription
}

// session consists of three peers, one publisher, one local transfer and multiple subscribers.
// A session is driven by pub/sub events.
type session struct {
	// id is the id of a publisher and also id of this session.
	// A subscriber subscribes video track identified by this unique id.
	id machineID

	// trackSource is the source of video track, either drone or monitor.
	trackSource pb.TrackSource

	// publisherChans contains edge device publisher delivering channels.
	// The channels must be initialized before use.
	publisherChans *publisherChans

	// subscriberChans contains browser subscriber delivering channels.
	// The channels must be initialized before use.
	subscriberChans *subscriberChans

	logger zerolog.Logger
	config WebRTCConfigOptions
}

func newSession(publisherChans *publisherChans, logger *zerolog.Logger, config WebRTCConfigOptions) *session {
	return &session{
		publisherChans: publisherChans,
		subscriberChans: &subscriberChans{
			offerChan:  make(chan *pb.SessionDescription, 1),
			answerChan: make(chan *pb.SessionDescription, 1),
		},
		logger: *logger,
		config: config,
	}
}

// start starts a session broadcasting video track from one publisher peer to at least one subscriber peer.
func (s *session) start(middle middleFunc) error {
	localTrack, err := createLocalTrack()
	if err != nil {
		return err
	}
	s.logger.Debug().Msg("created local track")

	if err := s.createPublisher(localTrack); err != nil {
		return fmt.Errorf("failed to crate publisher: %w", err)
	}
	s.logger.Debug().Str("id", string(s.id)).Int32("track_source", int32(s.trackSource)).Msg("created a publisher")

	middle(s.id, s.trackSource, s.subscriberChans)

	go func() {
		// Use a loop to start endless subscriber sessions.
		for {
			if err := s.createSubscriber(localTrack); err != nil {
				s.logger.Err(err).Msg("failed to create subscriber")
			}
		}
	}()

	return nil
}
