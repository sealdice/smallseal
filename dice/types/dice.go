package types

type DiceLike interface {
	RegisterExtension(extInfo *ExtInfo)
	GameSystemMapLoad(name string) (*GameSystemTemplateV2, error)

	SendReply(msg *MsgToReply)
}
