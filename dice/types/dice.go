package types

type DiceLike interface {
	RegisterExtension(extInfo *ExtInfo)
	GameSystemMapLoad(name string) (*GameSystemTemplateV2, error)
	ExtFind(s string, fromJS bool) *ExtInfo
	GetExtList() []*ExtInfo

	SendReply(msg *MsgToReply)

	RegisterMessageInHook(name string, priority HookPriority, hook MessageInHook) (HookHandle, error)
	UnregisterMessageInHook(handle HookHandle) bool

	RegisterMessageOutHook(name string, priority HookPriority, hook MessageOutHook) (HookHandle, error)
	UnregisterMessageOutHook(handle HookHandle) bool

	RegisterEventHook(name string, priority HookPriority, hook EventHook) (HookHandle, error)
	UnregisterEventHook(handle HookHandle) bool

	DispatchEvent(adapterID string, evt *AdapterEvent)
}
