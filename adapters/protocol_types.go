package adapters

import "smallseal/dice/types"

type RequestBase struct {
	MessageId string // 消息ID
}

type SimpleUserInfo struct {
	UserId   string // 用户ID
	UserName string // 用户名
}

// AdapterCallback 适配器回调接口
type AdapterCallback interface {
	OnError(err error)
	OnMessageReceived(info *MessageSendCallbackInfo)
}

// MessageSendRequest 发送消息请求
type MessageSendRequest struct {
	RequestBase
	Sender   *SimpleUserInfo         // 发送者 可不填
	Segments []types.IMessageElement // 消息段
	TargetId any                     // 目标用户ID/群组ID, string或int64
}

// MessageSendFileRequest 发送文件请求
type MessageSendFileRequest struct {
	RequestBase
	Sender   *SimpleUserInfo // 发送者 可不填
	FilePath string          // 文件路径
	TargetId any             // 目标用户ID/群组ID, string或int64
}

// GroupOperationRequest 群组操作请求
type GroupOperationKickRequest struct {
	RequestBase
	GroupID any // 群组ID
	UserID  any // 被操作用户ID（如果涉及用户操作）
}

// GroupOperationRequest 群组操作请求
type GroupOperationBanRequest struct {
	RequestBase
	GroupID  any   // 群组ID
	UserID   any   // 被操作用户ID（如果涉及用户操作）
	Duration int64 // 持续时间（如禁言时长）
}

// GroupOperationQuitRequest 群组操作请求
type GroupOperationQuitRequest struct {
	RequestBase
	GroupID any // 群组ID
}

// GroupOperationCardNameSetRequest 群组操作请求
type GroupOperationCardNameSetRequest struct {
	RequestBase
	GroupID any    // 群组ID
	UserID  any    // 用户ID（如果涉及用户操作）
	Name    string // 名称（如群名片）
}

// GroupInfo 群组信息
type GroupInfo struct {
	GroupID   any    // 群组ID
	GroupName string // 群组名称
}

// MessageOperationRequest 消息操作请求
type MessageOperationRequest struct {
	RequestBase
	MessageID any                     // 消息ID
	Segments  []types.IMessageElement // 消息段
}

// FriendOperationRequest 好友操作请求
type FriendOperationRequest struct {
	RequestBase
	UserID any // 用户ID
}
