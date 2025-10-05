package types

import "github.com/sealdice/smallseal/utils"

type DiceLike interface {
	RegisterExtension(extInfo *ExtInfo)
	GameSystemMapGet() *utils.SyncMap[string, *GameSystemTemplateV2]

	GameSystemMapLoad(name string) (*GameSystemTemplateV2, error)
	ExtFind(s string, fromJS bool) *ExtInfo
	GetExtList() []*ExtInfo
	MasterAdd(uid string)
	MasterRemove(uid string) bool
	ListMasters() []string
	IsMaster(uid string) bool

	PersistGroupInfo(groupID string, info *GroupInfo)
	SendReply(msg *MsgToReply)

	RegisterMessageInHook(name string, priority HookPriority, hook MessageInHook) (HookHandle, error)
	UnregisterMessageInHook(handle HookHandle) bool

	RegisterMessageOutHook(name string, priority HookPriority, hook MessageOutHook) (HookHandle, error)
	UnregisterMessageOutHook(handle HookHandle) bool

	RegisterEventHook(name string, priority HookPriority, hook EventHook) (HookHandle, error)
	UnregisterEventHook(handle HookHandle) bool

	DispatchEvent(adapterID string, evt *AdapterEvent)
}
