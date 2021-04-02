package session

import (
	pb "github.com/SB-IM/pb/signal"
	"github.com/pion/webrtc/v3"
)

// MachineID is unique id for edge device.
type MachineID string

// Session maps track source to local video track.
type Session map[pb.TrackSource]*webrtc.TrackLocalStaticRTP

// Sessions maps machine id to session.
type Sessions map[MachineID]Session
