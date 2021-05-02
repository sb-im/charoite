package livestream

import (
	"fmt"
	"strconv"

	pb "github.com/SB-IM/pb/signal"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pion/webrtc/v3"
)

func (p *publisher) sendOffer(sdp *webrtc.SessionDescription) error {
	payload, err := pb.EncodeSDP(sdp, p.meta)
	if err != nil {
		return fmt.Errorf("could not encode sdp: %w", err)
	}
	t := p.client.Publish(p.config.OfferTopic, byte(p.config.Qos), p.config.Retained, payload)
	// Handle the token in a go routine so this loop keeps sending messages regardless of delivery status
	go func() {
		<-t.Done()
		if t.Error() != nil {
			p.logger.Err(t.Error()).Msgf("could not publish to %s", p.config.OfferTopic)
		}
	}()
	return nil
}

// recvAnswer is a one time subscriber.
// The caller must check if result in channel is nil.
func (p *publisher) recvAnswer() <-chan *webrtc.SessionDescription {
	ch := make(chan *webrtc.SessionDescription, 1)
	topic := p.config.AnswerTopicPrefix + "/" + p.meta.Id + "/" + strconv.Itoa(int(p.meta.TrackSource))
	// Receive remote description with MQTT.
	t := p.client.Subscribe(topic, byte(p.config.Qos), func(c mqtt.Client, m mqtt.Message) {
		defer func() {
			c.Unsubscribe(topic)
			// Close channel so receiver never block even if subscribe failed.
			close(ch)
		}()

		sdp, err := pb.DecodeSDP(m.Payload())
		if err != nil {
			p.logger.Err(err).Msg("could not decode sdp")
			return
		}
		ch <- sdp
	})
	// the connection handler is called in a goroutine so blocking here would hot cause an issue. However as blocking
	// in other handlers does cause problems its best to just assume we should not block
	go func() {
		<-t.Done()
		if t.Error() != nil {
			p.logger.Err(t.Error()).Msgf("could not subscribe to %s", topic)
		} else {
			p.logger.Info().Msgf("subscribed to %s", topic)
		}
	}()

	return ch
}

// sendCandidate sends candidate to remote webRTC peer via MQTT.
// The publish topic is unique to this edge device.
func (p *publisher) sendCandidate(candidate *webrtc.ICECandidate) error {
	payload, err := pb.EncodeCandidate(candidate)
	if err != nil {
		return fmt.Errorf("could not encode candidate: %w", err)
	}
	topic := p.config.CandidateSendTopicPrefix + "/" + p.meta.Id + "/" + strconv.Itoa(int(p.meta.TrackSource))
	t := p.client.Publish(topic, byte(p.config.Qos), p.config.Retained, payload)
	// Handle the token in a go routine so this loop keeps sending messages regardless of delivery status
	go func() {
		<-t.Done()
		if t.Error() != nil {
			p.logger.Err(t.Error()).Msgf("could not publish to %s", p.config.CandidateSendTopicPrefix)
		}
	}()
	return nil
}

// recvCandidate is not a one time subscriber.
// The caller must check if result in channel is nil.
// sendCandidate receive candidate from remote webRTC peer via MQTT.
// The subscription topic is unique to this edge device.
func (p *publisher) recvCandidate() <-chan string {
	// TODO: Figure how to properly close channel.
	ch := make(chan string, 2) // Make buffer 2 because we have at least 2 sendings.
	topic := p.config.CandidateRecvTopicPrefix + "/" + p.meta.Id + "/" + strconv.Itoa(int(p.meta.TrackSource))
	// Receive remote ICE candidate with MQTT.
	t := p.client.Subscribe(topic, byte(p.config.Qos), func(c mqtt.Client, m mqtt.Message) {
		candidate, err := pb.DecodeCandidate(m.Payload())
		if err != nil {
			p.logger.Err(err).Msg("could not decode candidate")
			close(ch)
			return
		}
		ch <- candidate
	})
	// the connection handler is called in a goroutine so blocking here would hot cause an issue. However as blocking
	// in other handlers does cause problems its best to just assume we should not block
	go func() {
		<-t.Done()
		if t.Error() != nil {
			p.logger.Err(t.Error()).Msgf("could not subscribe to %s", topic)
		} else {
			p.logger.Info().Msgf("subscribed to %s", topic)
		}
	}()
	return ch
}
