package livestream

const (
	protocolRTP  = "rtp"
	protocolRTSP = "rtsp"
	protocolRTMP = "rtmp"
)

type PublisherConfigOptions struct {
	UUID string
	MQTTClientConfigOptions
	WebRTCConfigOptions

	// Currently only RTP is supported for drone.
	// Currently mainly RTSP, the other one is RTP for deport.
	StreamSource
}

type broadcastConfigOptions struct {
	MQTTClientConfigOptions
	WebRTCConfigOptions

	ConsumeStreamOnDemand bool
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
	Protocol string // rtp or rtsp or rtmp
	RTSPSourceConfigOptions
	RTPOrRTMPSourceConfigOptions

	ConsumeStreamOnDemand bool
}

type RTPOrRTMPSourceConfigOptions struct {
	Host string
	Port int
}

type RTSPSourceConfigOptions struct {
	Addr string
}
