package pure

type Channel string

const (
	ChannelEmail Channel = "email"
	ChannelSMS   Channel = "sms"
	ChannelInApp Channel = "inapp"
)

func (c Channel) String() string { return string(c) }

func AllChannels() []Channel {
	return []Channel{ChannelEmail, ChannelSMS, ChannelInApp}
}
