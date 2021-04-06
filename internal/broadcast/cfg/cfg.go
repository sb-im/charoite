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
	AnswerTopicSuffix        string
	CandidateSendTopicSuffix string // Opposite to edge's CandidateRecvTopicSuffix topic
	CandidateRecvTopicSuffix string // Opposite to edge's CandidateSendTopicSuffix topic.
	Qos                      uint
	Retained                 bool
}

type ServerConfigOptions struct {
	Host string
	Port int
}
