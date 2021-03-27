package livestream

type RTPBroadcastConfigOptions struct {
	TopicConfigOptions
	WebRTCConfigOptions
	RTPSourceConfigOptions
}

type RTSPBroadcastConfigOptions struct {
	TopicConfigOptions
	WebRTCConfigOptions
	RTSPSourceConfigOptions
}

type broadcastConfigOptions struct {
	TopicConfigOptions
	WebRTCConfigOptions
}

type TopicConfigOptions struct {
	OfferTopic  string
	AnswerTopic string
}

type WebRTCConfigOptions struct {
	ICEServer string
}

type RTPSourceConfigOptions struct {
	RTPHost string
	RTPPort int
}

type RTSPSourceConfigOptions struct {
	RTSPAddr string
}
