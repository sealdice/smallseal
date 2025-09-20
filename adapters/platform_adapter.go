package adapters

import (
	"fmt"
)

// PlatformAdapter 平台适配器接口 - 纯协议接口，不依赖业务上下文
type PlatformAdapter interface {
	// 连接管理
	IsAlive() bool

	// 消息发送
	MsgSendToGroup(request *MessageSendRequest) (bool, error)
	MsgSendToPerson(request *MessageSendRequest) (bool, error)

	// 文件发送
	MsgSendFileToPerson(request *MessageSendFileRequest) (bool, error)
	MsgSendFileToGroup(request *MessageSendFileRequest) (bool, error)

	// 消息操作
	MsgEdit(request *MessageOperationRequest) (bool, error)
	MsgRecall(request *MessageOperationRequest) (bool, error)

	// 群组操作
	GroupMemberBan(request *GroupOperationBanRequest) (bool, error)
	GroupMemberKick(request *GroupOperationKickRequest) (bool, error)

	GroupQuit(request *GroupOperationQuitRequest) (bool, error)
	GroupCardNameSet(request *GroupOperationCardNameSetRequest) (bool, error)
	GroupInfoGet(groupID any) (*GroupInfo, error)

	// 好友操作
	FriendDelete(request *FriendOperationRequest) (bool, error)
	FriendAdd(request *FriendOperationRequest) (bool, error)

	SetCallback(callback AdapterCallback)
}

// 实现检查
var (
// _ PlatformAdapter = (*PlatformAdapterMilky)(nil)
// _ PlatformAdapter = (*PlatformAdapterLagrangeGo)(nil)
)

func FormatDiceIDQQ(diceQQ string) string {
	return fmt.Sprintf("QQ:%s", diceQQ)
}

func FormatDiceIDQQGroup(diceQQ string) string {
	return fmt.Sprintf("QQ-Group:%s", diceQQ)
}

func FormatDiceIDQQCh(userID string) string {
	return fmt.Sprintf("QQ-CH:%s", userID)
}

func FormatDiceIDQQChGroup(guildID, channelID string) string {
	return fmt.Sprintf("QQ-CH-Group:%s-%s", guildID, channelID)
}
