package types

type DiceLike interface {
	RegisterExtension(extInfo *ExtInfo)
	GameSystemMapLoad(name string) (*GameSystemTemplateV2, error)
	ExtFind(s string, fromJS bool) *ExtInfo
	GetExtList() []*ExtInfo

	SendReply(msg *MsgToReply)
}
