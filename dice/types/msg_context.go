package types

type MsgContext struct {
	CommandID int64

	IsCurGroupBotOn bool

	MessageType string
	MsgID       string `json:"msgID"`
	Message     string `json:"message"`

	Group  *GroupInfo
	Player *GroupPlayerInfo
}
