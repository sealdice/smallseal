package types

type DiceLike interface {
	RegisterExtension(extInfo *ExtInfo)
	SendReply(msg *MsgToReply)
}
