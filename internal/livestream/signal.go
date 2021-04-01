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
	t := p.client.Publish(p.config.OfferTopic, 1, true, payload)
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
	topic := p.config.AnswerTopic + "/" + p.id + "/" + strconv.Itoa(int(p.trackSource))
	// Receive remote description with MQTT.
	t := p.client.Subscribe(topic, 1, func(c mqtt.Client, m mqtt.Message) {
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

// encodeSDP also adds an machine id field value.
func (p *publisher) encodeSDP(sdp *webrtc.SessionDescription) ([]byte, error) {
	msg := pb.SessionDescription{
		Sdp: &pb.SDP{
			Type:        int32(sdp.Type),
			Description: []byte(sdp.SDP),
		},
		Id:          p.id,
		TrackSource: p.trackSource,
	}
	payload, err := proto.Marshal(&msg)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func decodeSDP(payload []byte) (*webrtc.SessionDescription, error) {
	var sdp pb.SessionDescription
	if err := proto.Unmarshal(payload, &sdp); err != nil {
		return nil, err
	}
	return &webrtc.SessionDescription{
		Type: webrtc.SDPType(sdp.Sdp.Type),
		SDP:  string(sdp.Sdp.Description),
	}, nil
}

func machineID() ([]byte, error) {
	return os.ReadFile("/etc/machine-id")
}
