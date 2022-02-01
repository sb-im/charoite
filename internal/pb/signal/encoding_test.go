package signal

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/pion/webrtc/v3"
)

func TestSDPEncoding(t *testing.T) {
	sdp := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  "abc",
	}

	t.Run("full", func(t *testing.T) {
		b, err := EncodeSDP(&sdp, &Meta{
			Id:          "abc",
			TrackSource: TrackSource_DRONE,
		})
		if err != nil {
			t.Error(err)
		}
		if b == nil {
			t.Fatalf("encoded protobuf payload is nil")
		}
	})

	t.Run("without meta", func(t *testing.T) {
		b, err := EncodeSDP(&sdp, nil)
		if err != nil {
			t.Error(err)
		}
		if b == nil {
			t.Fatalf("encoded protobuf payload is nil")
		}

		sdp, err := DecodeSDP(b)
		if err != nil {
			t.Fatalf("could not decode SDP: %v", err)
		}
		if sdp.Type != webrtc.SDPTypeOffer {
			t.Fatalf("type is incorrect, got %s want %s", sdp.Type, webrtc.SDPTypeOffer)
		}
		if sdp.SDP != "abc" {
			t.Fatalf("sdp is incorrect, got %s want %s", sdp.SDP, "abc")
		}
	})
}

func TestCandidateEncoding(t *testing.T) {
	candidate := webrtc.ICECandidate{
		Foundation: "abc",
		Priority:   1,
		Address:    "abc",
	}

	b, err := EncodeCandidate(&candidate)
	if err != nil {
		t.Fatal(err)
	}
	if b == nil {
		t.Fatal("nil encoded brotobuf payload")
	}

	c, err := DecodeCandidate(b)
	if err != nil {
		t.Fatal(err)
	}
	if c == "" {
		t.Fatal("empty candidate")
	}
}

func TestTrackSourceEncoding(t *testing.T) {
	data := struct {
		Meta *Meta `json:"meta,omitempty"`
	}{
		Meta: &Meta{
			Id:          "abc",
			TrackSource: TrackSource_DRONE,
		},
	}
	b, err := json.Marshal(&data)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%s", b)
}
