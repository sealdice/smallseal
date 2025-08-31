package dice

import (
	"fmt"
	"smallseal/dice/types"
)

func ReplyToSenderRaw(ctx *types.MsgContext, msg *types.Message, text string, flag string) {
	inGroup := msg.MessageType == "group"
	if inGroup {
		ReplyGroupRaw(ctx, msg, text, flag)
	} else {
		ReplyPersonRaw(ctx, msg, text, flag)
	}
}

func ReplyToSender(ctx *types.MsgContext, msg *types.Message, text string) {
	ReplyToSenderRaw(ctx, msg, text, "")
}

// func ReplyToSenderNoCheck(ctx *types.MsgContext, msg *types.Message, text string) {
// 	go replyToSenderRawNoCheck(ctx, msg, text, "")
// }

func ReplyGroupRaw(ctx *types.MsgContext, msg *types.Message, text string, flag string) {
	fmt.Printf("%s\n", text)
}

func ReplyPersonRaw(ctx *types.MsgContext, msg *types.Message, text string, flag string) {
	fmt.Printf("%s\n", text)
}
