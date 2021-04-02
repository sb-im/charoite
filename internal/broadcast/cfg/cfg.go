package cfg

type ConfigOptions struct {
	WebRTCConfigOptions
	TopicConfigOptions
	ServerConfigOptions
}

type PublisherConfigOptions struct {
	TopicConfigOptions
	WebRTCConfigOptions
}

type WebRTCConfigOptions struct {
	ICEServer string
}

type TopicConfigOptions struct {
	OfferTopic  string
	AnswerTopic string
}

type ServerConfigOptions struct {
	Host string
	Port int
}
