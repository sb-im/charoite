package livestream

import (
	"fmt"
	"os"
	"strconv"

	pb "github.com/SB-IM/pb/signal"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pion/webrtc/v3"
	"google.golang.org/protobuf/proto"
)

func (p *publisher) sendOffer(sdp *webrtc.SessionDescription) error {
	payload, err := p.encodeSDP(sdp)
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
	topic := p.config.AnswerTopicSuffix + "/" + p.id + "/" + strconv.Itoa(int(p.trackSource))
	// Receive remote description with MQTT.
	t := p.client.Subscribe(topic, byte(p.config.Qos), func(c mqtt.Client, m mqtt.Message) {
		defer func() {
			c.Unsubscribe(topic)
			// Close channel so receiver never block even if subscribe failed.
			close(ch)
		}()

		sdp, err := decodeSDP(m.Payload())
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
	payload, err := encodeCandidate(candidate)
	if err != nil {
		return fmt.Errorf("could not encode candidate: %w", err)
	}
	topic := p.config.CandidateSendTopicSuffix + "/" + p.id + "/" + strconv.Itoa(int(p.trackSource))
	t := p.client.Publish(topic, byte(p.config.Qos), p.config.Retained, payload)
	// Handle the token in a go routine so this loop keeps sending messages regardless of delivery status
	go func() {
		<-t.Done()
		if t.Error() != nil {
			p.logger.Err(t.Error()).Msgf("could not publish to %s", p.config.CandidateSendTopicSuffix)
		}
	}()
	return nil
}

// recvCandidate is not a one time subscriber.
// The caller must check if result in channel is nil.
// sendCandidate receive candidate from remote webRTC peer via MQTT.
// The subscription topic is unique to this edge device.
func (p *publisher) recvCandidate() <-chan *webrtc.ICECandidate {
	// TODO: Figure how to properly close channel.
	ch := make(chan *webrtc.ICECandidate, 2) // Make buffer 2 because we have at least 2 sendings.
	topic := p.config.CandidateRecvTopicSuffix + "/" + p.id + "/" + strconv.Itoa(int(p.trackSource))
	// Receive remote ICE candidate with MQTT.
	t := p.client.Subscribe(topic, byte(p.config.Qos), func(c mqtt.Client, m mqtt.Message) {
		candidate, err := decodeCandidate(m.Payload())
		if err != nil {
			p.logger.Err(err).Msg("could not decode candidate")
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

// encodeSDP also adds an machine id field value.
func (p *publisher) encodeSDP(sdp *webrtc.SessionDescription) ([]byte, error) {
	msg := pb.SessionDescription{
		Sdp: &pb.SDP{
			Type: sdp.Type.String(),
			Sdp:  sdp.SDP,
		},
		Id:          p.id,
		TrackSource: p.trackSource,
	}
	return proto.Marshal(&msg)
}

func decodeSDP(payload []byte) (*webrtc.SessionDescription, error) {
	var sdp pb.SessionDescription
	if err := proto.Unmarshal(payload, &sdp); err != nil {
		return nil, err
	}
	return &webrtc.SessionDescription{
		Type: webrtc.NewSDPType(sdp.Sdp.Type),
		SDP:  sdp.Sdp.Sdp,
	}, nil
}

func encodeCandidate(candidate *webrtc.ICECandidate) ([]byte, error) {
	msg := pb.ICECandidate{
		Foundation:     candidate.Foundation,
		Priority:       candidate.Priority,
		Address:        candidate.Address,
		Protocol:       int32(candidate.Protocol),
		Port:           uint32(candidate.Port),
		Type:           int32(candidate.Typ),
		Component:      uint32(candidate.Typ),
		RelatedAddress: candidate.RelatedAddress,
		RelatedPort:    uint32(candidate.RelatedPort),
		TcpType:        candidate.TCPType,
	}
	return proto.Marshal(&msg)
}

func decodeCandidate(payload []byte) (*webrtc.ICECandidate, error) {
	var candidate pb.ICECandidate
	if err := proto.Unmarshal(payload, &candidate); err != nil {
		return nil, err
	}
	return &webrtc.ICECandidate{
		Foundation:     candidate.Foundation,
		Priority:       candidate.Priority,
		Address:        candidate.Address,
		Protocol:       webrtc.ICEProtocol(candidate.Protocol),
		Port:           uint16(candidate.Port),
		Typ:            webrtc.ICECandidateType(candidate.Type),
		Component:      uint16(candidate.Component),
		RelatedAddress: candidate.RelatedAddress,
		RelatedPort:    uint16(candidate.RelatedPort),
		TCPType:        candidate.TcpType,
	}, nil
}

func machineID() ([]byte, error) {
	return os.ReadFile("/etc/machine-id")
}
