package livestream

const (
	protocolRTP  = "rtp"
	protocolRTSP = "rtsp"
)

type DroneBroadcastConfigOptions struct {
	UUID string
	MQTTClientConfigOptions
	WebRTCConfigOptions
	StreamSource // Currently only RTP is supported
}

type DeportBroadcastConfigOptions struct {
	UUID string
	MQTTClientConfigOptions
	WebRTCConfigOptions
	StreamSource // Currently mainly RTSP, the other one is RTP
}

type broadcastConfigOptions struct {
	MQTTClientConfigOptions
	WebRTCConfigOptions
}

type MQTTClientConfigOptions struct {
	OfferTopicPrefix         string
	AnswerTopicPrefix        string
	CandidateSendTopicPrefix string // Opposite to cloud's CandidateRecvTopicPrefix topic
	CandidateRecvTopicPrefix string // Opposite to cloud's CandidateSendTopicPrefix topic.
	HookStreamTopicPrefix    string
	Qos                      uint
	Retained                 bool
}

type WebRTCConfigOptions struct {
	ICEServer  string
	Username   string
	Credential string
}

type StreamSource struct {
	Protocol string // rtp or rtsp
	RTSPSourceConfigOptions
	RTPSourceConfigOptions
}

type RTPSourceConfigOptions struct {
	Host string
	Port int
}

type RTSPSourceConfigOptions struct {
	Addr string
}
