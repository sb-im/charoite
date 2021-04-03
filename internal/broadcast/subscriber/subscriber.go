package subscriber

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"sync"

	pb "github.com/SB-IM/pb/signal"
	"github.com/gorilla/mux"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog"

	"github.com/SB-IM/skywalker/internal/broadcast/cfg"
	webrtcx "github.com/SB-IM/skywalker/internal/broadcast/webrtc"
)

// Subscriber stands for a subscriber webRTC peer.
type Subscriber struct {
	config cfg.WebRTCConfigOptions
	logger zerolog.Logger

	// sessions must be created before used by publisher and is shared between publishers ans subscribers.
	// It's only read by subscriber.
	sessions *sync.Map
}

// New returns a new Subscriber.
func New(sessions *sync.Map, logger *zerolog.Logger, config cfg.WebRTCConfigOptions) *Subscriber {
	l := logger.With().Str("component", "Subscriber").Logger()
	return &Subscriber{
		sessions: sessions,
		config:   config,
		logger:   l,
	}
}

// Signal performs webRTC signaling for all subscriber peers.
func (s *Subscriber) Signal() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/v1/broadcast/signal", s.handleSignal()).Methods(http.MethodPost)

	if os.Getenv("DEBUG") == "true" {
		r.Handle("/", http.FileServer(http.Dir("e2e/broadcast/static")))
	}
	s.logger.Debug().Msg("registered signal HTTP handler")
	return r
}

// handleSignal handles HTTP subscriber.
func (s *Subscriber) handleSignal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var offer pb.SessionDescription
		if err := json.NewDecoder(r.Body).Decode(&offer); err != nil {
			s.logger.Err(err).Msg("could not decode request json body")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		logger := s.logger.With().Str("id", offer.Id).Int32("track_source", int32(offer.TrackSource)).Logger()
		logger.Debug().Msg("received offer from subscriber")

		sessionID := offer.Id + strconv.Itoa(int(offer.TrackSource))
		value, ok := s.sessions.Load(sessionID)
		if !ok {
			logger.Warn().Msg("no machine id or track source found in existing sessions")
			http.Error(w, "wrong id or track source", http.StatusBadRequest)
			return
		}

		wcx := webrtcx.New(s.config, &logger)
		// TODO: handle blocking case with timeout for channels.
		wcx.OfferChan <- &offer
		if err := wcx.CreateSubscriber(value.(*webrtc.TrackLocalStaticRTP)); err != nil {
			logger.Err(err).Msg("failed to create subscriber")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		logger.Debug().Msg("successfully created subscriber")

		answer := <-wcx.AnswerChan
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(answer.Sdp); err != nil {
			logger.Err(err).Msg("could not encode json response body")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		logger.Debug().Msg("sent answer to subscriber")
	}
}
