package types

// 消息返回的标准格式

type MsgSendToInfo struct {
	Platform string `jsbind:"platform"    json:"platform"` // 当前平台
	GroupId  string `jsbind:"groupId"     json:"groupId"`  // 群号，如果是群聊消息
	UserId   string `jsbind:"userId"   json:"userId"`
	Nickname string `jsbind:"nickname" json:"nickname"`
}

type MsgSenderInfo struct {
	Platform string `jsbind:"platform"    json:"platform"` // 当前平台
	GroupId  string `jsbind:"groupId"     json:"groupId"`  // 群号，如果是群聊消息
	UserId   string `jsbind:"userId"   json:"userId"`
	Nickname string `jsbind:"nickname" json:"nickname"`
}

type MsgToReply struct {
	AdapterId string
	CommandId int64

	Sender MsgSenderInfo
	SendTo MsgSendToInfo

	Time        int64  `jsbind:"time"        json:"time"`        // 发送时间
	MessageType string `jsbind:"messageType" json:"messageType"` // group private

	Segments    MessageSegments `jsbind:"segment" json:"segments" yaml:"-"`
	CommandData any             // 制定一个格式，返回一些更加本质的东西，到外面去二次套壳

	CommandFormatInfo []*CommandFormatInfo
}
