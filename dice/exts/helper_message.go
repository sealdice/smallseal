package exts

import (
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

func ReplyRaw(ctx *types.MsgContext, msg *types.Message, text string, flag string, messageType string) {
	var sendToGroupId string
	if messageType == "group" {
		sendToGroupId = msg.GroupID
	}

	ctx.Dice.SendReply(&types.MsgToReply{
		AdapterId: msg.Platform,
		CommandId: ctx.CommandId,
		Sender: types.MsgSenderInfo{
			Platform: msg.Platform,
			GroupId:  msg.GroupID,
			UserId:   "", // 骰子作为发送者，暂时留空
			Nickname: "", // 骰子昵称，暂时留空
		},
		SendTo: types.MsgSendToInfo{
			Platform: msg.Platform,
			GroupId:  sendToGroupId,
			UserId:   msg.Sender.UserID,
			Nickname: msg.Sender.Nickname,
		},
		Time:        msg.Time,
		MessageType: messageType,
		Segments:    types.MessageSegments{&types.TextElement{Content: text}},
	})
}

func ReplyGroupRaw(ctx *types.MsgContext, msg *types.Message, text string, flag string) {
	ReplyRaw(ctx, msg, text, flag, "group")
}

func ReplyPersonRaw(ctx *types.MsgContext, msg *types.Message, text string, flag string) {
	ReplyRaw(ctx, msg, text, flag, "private")
}

func ReplyGroup(ctx *types.MsgContext, msg *types.Message, text string) {
	ReplyGroupRaw(ctx, msg, text, "")
}

func ReplyPerson(ctx *types.MsgContext, msg *types.Message, text string) {
	ReplyPersonRaw(ctx, msg, text, "")
}

func CompatibleReplace(ctx *types.MsgContext, s string) string {
	// 我去，这真麻烦。我感觉这个应该弄到钩子上去
	// s = ctx.TranslateSplit(s)

	// // 匹配 #{DRAW-$1}, 其中$1执行最短匹配且允许左侧右侧各有一个花括号
	// // #{DRAW-aaa} => aaa
	// // #{DRAW-{aaa} => {aaa
	// // #{DRAW-aaa}} => aaa}
	// // #{DRAW-{aaa}} => {aaa}
	// // 这允许在牌组名中使用不含空格的表达式(主要是为了变量)
	// re := regexp.MustCompile(`#\{DRAW-(\{?\S+?\}?)\}`)
	// s = re.ReplaceAllString(s, "###DRAW-$1###")

	// if ctx != nil {
	// 	s = DeckRewrite(s, func(deckName string) string {
	// 		// 如果牌组名中含有表达式, 在此进行求值
	// 		// 不含表达式也无妨, 求值完还是原来的字符串
	// 		r, _, err := DiceExprTextBase(ctx, deckName, RollExtraFlags{})
	// 		if err == nil {
	// 			deckName = r.ToString()
	// 		}

	// 		exists, result, err := deckDraw(ctx, deckName, false)
	// 		if !exists {
	// 			return "<%未知牌组-" + deckName + "%>"
	// 		}
	// 		if err != nil {
	// 			return "<%抽取错误-" + deckName + "%>"
	// 		}
	// 		return result
	// 	})
	// }
	return s
}
