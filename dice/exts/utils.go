package exts

import (
	"fmt"
	"strings"

	"github.com/samber/lo"

	"github.com/sealdice/smallseal/dice/types"
)

// SetCardType 这几个先不动，后面改名字或者移除
func SetCardType(mctx *types.MsgContext, curType string) {
	am := mctx.AttrsManager
	curAttrs, err := am.Load(mctx.Group.GroupId, mctx.Player.UserId) // 即 LoadByCtx
	if err != nil {
		return
	}
	curAttrs.SetSheetType(curType)
}

func ReadCardType(mctx *types.MsgContext) string {
	am := mctx.AttrsManager
	curAttrs := lo.Must(am.Load(mctx.Group.GroupId, mctx.Player.UserId))
	return curAttrs.SheetType
}

func ReadCardTypeEx(mctx *types.MsgContext, curType string) string {
	var extra string

	am := mctx.AttrsManager
	curAttrs := lo.Must(am.Load(mctx.Group.GroupId, mctx.Player.UserId))
	cardType := curAttrs.SheetType

	if cardType != "" {
		if cardType != curType {
			extra = fmt.Sprintf("\n这似乎是一张 %s 人物卡，请确认当前游戏类型", cardType)
		}
	}
	return extra
}

// GetCtxProxyFirst 第三个参数是否取消再观察一下
func GetCtxProxyFirst(ctx *types.MsgContext, cmdArgs *types.CmdArgs) *types.MsgContext {
	return GetCtxProxyAtPosRaw(ctx, cmdArgs, 0, true)
}

// GetCtxProxyAtPos
func GetCtxProxyAtPos(ctx *types.MsgContext, cmdArgs *types.CmdArgs, pos int) *types.MsgContext {
	return GetCtxProxyAtPosRaw(ctx, cmdArgs, pos, true)
}

// GetCtxProxyAtPosRaw 等着后续再加 @名字 版本，以及 @team 版本
func GetCtxProxyAtPosRaw(ctx *types.MsgContext, cmdArgs *types.CmdArgs, pos int, setTempVar bool) *types.MsgContext {
	cur := 0
	for _, i := range cmdArgs.At {
		// 有关 EndPoint 先注释
		// if i.UserID == ctx.EndPoint.UserID {
		// 	continue
		// } else if strings.HasPrefix(ctx.EndPoint.UserID, "OpenQQ:") {
		// 	// 特殊处理 OpenQQ频道
		// 	uid := strings.TrimPrefix(i.UserID, "OpenQQCH:")
		// 	diceId := strings.TrimPrefix(ctx.EndPoint.UserID, "OpenQQ:")
		// 	if uid == diceId {
		// 		continue
		// 	}
		// }

		if pos != cur {
			cur++
			continue
		}

		// 这个其实无用，因为有@其他bot的指令不会进入到solve阶段
		if ctx.Group != nil {
			isBot := false
			ctx.Group.BotList.Range(func(botUid string, _ bool) bool {
				if i.UserID == botUid {
					isBot = true
					return false
				}
				return true
			})
			if isBot {
				continue
			}
		}

		// TODO: 感觉比较抽象，为啥这个在atInfo里？
		// 原来是有个这样的取值 p := ctx.Group.PlayerGet(ctx.Dice.DBOperator, i.UserID) 先简略一下
		// mctx, _ := i.CopyCtx(ctx)
		mctx := ctx.Copy()
		mctx.Player.UserId = i.UserID

		if setTempVar {
			if mctx.Player != ctx.Player {
				SetTempVars(mctx, "???")
			} else {
				SetTempVars(mctx, mctx.Player.Name)
			}
		}

		if mctx.Player.UserId == ctx.Player.UserId {
			// 并非代骰
			ctx.DelegateText = ""
		}
		return mctx
	}
	ctx.DelegateText = ""
	return ctx
}

func UserIDExtract(uid string) string {
	index := strings.Index(uid, ":")
	if index != -1 {
		return uid[index+1:]
	}
	return uid
}

func SetPlayerGroupCardByTemplate(ctx *types.MsgContext, tmpl string) (string, error) {
	// ctx.Player.TempValueAlias = nil // 防止dnd的hp被转为“生命值”

	// config := ctx.GenDefaultRollVmConfig()
	// config.HookFuncValueStore = func(ctx *ds.Context, name string, v *ds.VMValue) (overwrite *ds.VMValue, solved bool) {
	// 	return nil, true
	// }
	// v := ctx.EvalFString(tmpl, config)
	// if v.vm.Error != nil {
	// 	ctx.Dice.Logger.Infof("SN指令模板错误: %v", v.vm.Error.Error())
	// 	return "", v.vm.Error
	// }

	// text := v.ToString()
	// if ctx.EndPoint.Platform == "QQ" && len(text) >= 60 { // Note(Xiangze-Li): 2023-08-09实测群名片长度限制为59个英文字符, 20个中文字符是可行的, 但分别判断过于繁琐
	// 	return text, ErrGroupCardOverlong
	// }

	// ctx.EndPoint.Adapter.SetGroupCardName(ctx, text)
	return "", nil
}
