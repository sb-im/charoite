package broadcast

type ConfigOptions struct {
	WebRTCConfigOptions
	TopicConfigOptions
	WSServerConfigOptions
}

type WebRTCConfigOptions struct {
	ICEServer string
}

type TopicConfigOptions struct {
	OfferTopic  string
	AnswerTopic string
}

type WSServerConfigOptions struct {
	Host string
	Port int
	Path string
}
