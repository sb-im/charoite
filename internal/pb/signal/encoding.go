package signal

import (
	"encoding/json"

	"github.com/pion/webrtc/v3"
	"google.golang.org/protobuf/proto"
)

// EncodeSDP encodes webrtc.SessionDescription with metadata to protobuf payload.
// For Skywalker broadcast, meta param may be nil.
func EncodeSDP(sdp *webrtc.SessionDescription, meta *Meta) ([]byte, error) {
	b, err := json.Marshal(sdp)
	if err != nil {
		return nil, err
	}

	msg := SessionDescription{
		Meta: meta,
		Sdp:  string(b),
	}
	return proto.Marshal(&msg)
}

// DecodeSDP decodes protobuf payload SessionDescription to webrtc.SessionDescription.
// It ignores the metadata.
// Mainly used by Sphinx livestream.
func DecodeSDP(payload []byte) (*webrtc.SessionDescription, error) {
	var msg SessionDescription
	if err := proto.Unmarshal(payload, &msg); err != nil {
		return nil, err
	}
	var sdp webrtc.SessionDescription
	if err := json.Unmarshal([]byte(msg.Sdp), &sdp); err != nil {
		panic(err)
	}
	return &sdp, nil
}

// EncodeCandidate encodes webrtc.ICECandidate to protobuf payload.
func EncodeCandidate(candidate *webrtc.ICECandidate) ([]byte, error) {
	msg := ICECandidate{
		Candidate: candidate.ToJSON().Candidate,
	}
	return proto.Marshal(&msg)
}

// DecodeCandidate decodes protobuf payload ICECandidate to string form webrtc.ICECandidate.
func DecodeCandidate(payload []byte) (string, error) {
	var candidate ICECandidate
	if err := proto.Unmarshal(payload, &candidate); err != nil {
		return "", err
	}
	return candidate.Candidate, nil
}
