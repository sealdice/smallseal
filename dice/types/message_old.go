package types

import "fmt"

type SenderBase struct {
	Nickname  string `jsbind:"nickname" json:"nickname"`
	UserID    string `jsbind:"userId"   json:"userId"`
	GroupRole string `json:"-"` // 群内角色 admin管理员 owner群主
}

// Message 消息的重要信息
// 时间
// 发送地点(群聊/私聊)
// 人物(是谁发的)
// 内容
type Message struct {
	Time        int64      `jsbind:"time"        json:"time"`        // 发送时间
	MessageType string     `jsbind:"messageType" json:"messageType"` // group private
	GroupID     string     `jsbind:"groupId"     json:"groupId"`     // 群号，如果是群聊消息
	GuildID     string     `jsbind:"guildId"     json:"guildId"`     // 服务器群组号，会在discord,kook,dodo等平台见到
	ChannelID   string     `jsbind:"channelId"   json:"channelId"`
	Sender      SenderBase `jsbind:"sender"      json:"sender"`   // 发送者
	Message     string     `jsbind:"message"     json:"message"`  // 消息内容
	RawID       any        `jsbind:"rawId"       json:"rawId"`    // 原始信息ID，用于处理撤回等
	Platform    string     `jsbind:"platform"    json:"platform"` // 当前平台
	GroupName   string     `json:"groupName"`
	// Note(Szzrain): 这里是消息段，为了支持多种消息类型，目前只有 Milky 支持，其他平台也应该尽快迁移支持，并使用 Session.ExecuteNew 方法
	Segment MessageSegments `jsbind:"segment" json:"-" yaml:"-"`
}

type CQCommand struct {
	Type      string
	Args      map[string]string
	Overwrite string
}

func (c *CQCommand) Compile() string {
	if c.Overwrite != "" {
		return c.Overwrite
	}
	argsPart := ""
	for k, v := range c.Args {
		argsPart += fmt.Sprintf(",%s=%s", k, v)
	}
	return fmt.Sprintf("[CQ:%s%s]", c.Type, argsPart)
}
