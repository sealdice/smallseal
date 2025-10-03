package types

type HookHandle string

type HookPriority int

const (
	HookPriorityLow    HookPriority = -10
	HookPriorityNormal HookPriority = 0
	HookPriorityHigh   HookPriority = 10
)

type HookResult int

const (
	HookResultContinue HookResult = iota
	HookResultStop
	HookResultAbort
)

type AdapterEvent struct {
	Platform   string         // 源平台标识，如 QQ
	PostType   string         // 原始 post_type，例如 notice/request
	Type       string         // 事件类型，例如 group_increase
	SubType    string         // 事件子类型
	Time       int64          // 事件发生时间戳
	GroupID    string         // 相关群 ID（Dice 格式）
	GuildID    string         // 相关频道服务器 ID
	ChannelID  string         // 相关频道 ID
	UserID     string         // 相关用户 ID
	OperatorID string         // 操作者 ID
	Raw        map[string]any // 原始事件数据，便于扩展
}

type MessageInHook func(d DiceLike, adapterID string, msg *Message, ctx *MsgContext) HookResult

type MessageOutHook func(d DiceLike, adapterID string, reply *MsgToReply) HookResult

type EventHook func(d DiceLike, adapterID string, evt *AdapterEvent) HookResult
