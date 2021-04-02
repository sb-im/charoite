package subscriber

import (
	"encoding/json"
	"net/http"

	pb "github.com/SB-IM/pb/signal"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"

	"github.com/SB-IM/skywalker/internal/broadcast/cfg"
	"github.com/SB-IM/skywalker/internal/broadcast/session"
	webrtcx "github.com/SB-IM/skywalker/internal/broadcast/webrtc"
)

// Subscriber stands for a subscriber webRTC peer.
type Subscriber struct {
	config cfg.WebRTCConfigOptions
	logger zerolog.Logger

	// sessions must be created before used by publisher and is shared between publishers ans subscribers.
	// It's only read by subscriber.
	sessions session.Sessions
}

// New returns a new Subscriber.
func New(sessions session.Sessions, logger *zerolog.Logger, config cfg.WebRTCConfigOptions) *Subscriber {
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
	r.HandleFunc("/v1/broadcast/signal", s.handleSignal()).
		Methods(http.MethodPost).
		Headers("Content-Type", "application/json")

	return r
}

// handleSignal handles HTTP subscriber.
func (s *Subscriber) handleSignal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// w.Header().Set("Access-Control-Allow-Origin", "*")
		// w.Header().Set("Access-Control-Allow-Methods", "*")
		// w.Header().Set("Access-Control-Allow-Headers", "*")

		var offer pb.SessionDescription
		if err := json.NewDecoder(r.Body).Decode(&offer); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ses, ok := s.sessions[session.MachineID(offer.Id)]
		if !ok {
			http.Error(w, "wrong id", http.StatusBadRequest)
			return
		}
		videoTrack, ok := ses[offer.TrackSource]
		if !ok {
			http.Error(w, "wrong track_source", http.StatusBadRequest)
			return
		}

		wcx := webrtcx.New(videoTrack, s.config, &s.logger)
		// TODO: handle blocking case with timeout for channels.
		wcx.OfferChan <- &offer
		if err := wcx.CreateSubscriber(); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		answer := <-wcx.AnswerChan
		if err := json.NewEncoder(w).Encode(&answer); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
