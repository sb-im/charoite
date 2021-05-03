package cfg

type ConfigOptions struct {
	WebRTCConfigOptions
	MQTTClientConfigOptions
	ServerConfigOptions
}

type PublisherConfigOptions struct {
	MQTTClientConfigOptions
	WebRTCConfigOptions
}

type WebRTCConfigOptions struct {
	ICEServer      string
	Username       string
	Credential     string
	EnableFrontend bool // Enable static file server handler serving webRTC frontend, useful for debug
}

type MQTTClientConfigOptions struct {
	OfferTopic               string
	AnswerTopicPrefix        string
	CandidateSendTopicPrefix string // Opposite to edge's CandidateRecvTopicPrefix topic
	CandidateRecvTopicPrefix string // Opposite to edge's CandidateSendTopicPrefix topic.
	Qos                      uint
	Retained                 bool
}

type ServerConfigOptions struct {
	Host string
	Port int
}
