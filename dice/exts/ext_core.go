package exts

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"smallseal/dice/types"

	ds "github.com/sealdice/dicescript"
)

func RegisterBuiltinExtCore(dice types.DiceLike) {
	theExt := &types.ExtInfo{
		Name:       "core",
		Version:    "1.0.0",
		Brief:      "核心逻辑模块，该扩展即使被关闭也会依然生效",
		Author:     "SealDice-Team",
		AutoActive: true, // 是否自动开启
		Official:   true,
	}

	cmdMap := map[string]*types.CmdItemInfo{}

	helpRoll := ".r <表达式> [<原因>] // 骰点指令\n.rh <表达式> <原因> // 暗骰"
	cmdRoll := &types.CmdItemInfo{
		EnableExecuteTimesParse: true,
		Name:                    "roll",
		ShortHelp:               helpRoll,
		Help:                    "骰点:\n" + helpRoll,
		Solve: func(ctx *types.MsgContext, msg *types.Message, cmdArgs *types.CmdArgs) types.CmdExecuteResult {
			var text string
			var diceResult int64
			var diceResultExists bool
			var detail string

			if cmdArgs.IsArgEqual(1, "help") {
				return types.CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			// TODO: SystemTemplate和Config字段在当前版本中不存在，暂时注释掉
			// ctx.SystemTemplate = ctx.Group.GetCharTemplate(ctx.Dice)
			// if ctx.Dice.Config.CommandCompatibleMode {
			//	if (cmdArgs.Command == "rd" || cmdArgs.Command == "rhd" || cmdArgs.Command == "rdh") && len(cmdArgs.Args) >= 1 {
			//		if m, _ := regexp.MatchString(`^\d|优势|劣势|\+|-`, cmdArgs.CleanArgs); m {
			//			if cmdArgs.IsSpaceBeforeArgs {
			//				cmdArgs.CleanArgs = "d " + cmdArgs.CleanArgs
			//			} else {
			//				cmdArgs.CleanArgs = "d" + cmdArgs.CleanArgs
			//			}
			//		}
			//	}
			// }

			var r *ds.VMValue
			var commandInfoItems []any

			rollOne := func() *types.CmdExecuteResult {
				forWhat := ""
				var matched string

				vm := ctx.GetVM()
				if len(cmdArgs.Args) >= 1 { //nolint:nestif
					var err error
					r, detail, err = ctx.EvalBase(cmdArgs.CleanArgs, &ds.RollConfig{
						DisableStmts: true,
					})

					// TODO: r变量和相关方法不存在，暂时注释掉
					if r != nil && !(vm.IsCalculateExists() || vm.IsComputedLoaded) {
						forWhat = cmdArgs.CleanArgs

						defExpr := "d"
						// TODO: ctx.diceExprOverwrite字段在MsgContext中不存在，暂时注释掉
						// if ctx.diceExprOverwrite != "" {
						//	defExpr = ctx.diceExprOverwrite
						// }
						r, detail, err = ctx.EvalBase(defExpr, &ds.RollConfig{
							// DefaultDiceSideNum: getDefaultDicePoints(ctx),
							DisableStmts: true,
						})
					}

					if r != nil && r.TypeId == ds.VMTypeInt {
						diceResult = int64(r.MustReadInt())
						diceResultExists = true
					}

					if err == nil {
						matched = vm.Matched
						if forWhat == "" {
							forWhat = vm.RestInput
						}
					} else {
						errs := err.Error()
						if strings.HasPrefix(errs, "E1:") || strings.HasPrefix(errs, "E5:") || strings.HasPrefix(errs, "E6:") || strings.HasPrefix(errs, "E7:") || strings.HasPrefix(errs, "E8:") {
							ReplyToSender(ctx, msg, errs)
							return &types.CmdExecuteResult{Matched: true, Solved: true}
						}
						forWhat = cmdArgs.CleanArgs
					}
				}

				VarSetValueStr(ctx, "$t原因", forWhat)
				if forWhat != "" {
					forWhatText := DiceFormatTmpl(ctx, "核心:骰点_原因")
					VarSetValueStr(ctx, "$t原因句子", forWhatText)
				} else {
					VarSetValueStr(ctx, "$t原因句子", "")
				}

				if diceResultExists { //nolint:nestif
					detailWrap := ""
					if detail != "" {
						detailWrap = "=" + detail
						re := regexp.MustCompile(`\[((\d+)d\d+)\=(\d+)\]`)
						match := re.FindStringSubmatch(detail)
						if len(match) > 0 {
							num := match[2]
							if num == "1" && (match[1] == matched || match[1] == "1"+matched) {
								detailWrap = ""
							}
						}
					}

					// 指令信息标记
					item := map[string]interface{}{
						"expr":   matched,
						"result": diceResult,
						"reason": forWhat,
					}
					if forWhat == "" {
						delete(item, "reason")
					}
					commandInfoItems = append(commandInfoItems, item)

					VarSetValueStr(ctx, "$t表达式文本", matched)
					VarSetValueStr(ctx, "$t计算过程", detailWrap)
					VarSetValueInt64(ctx, "$t计算结果", diceResult)
				} else {
					var val int64
					var detail string
					// dicePoints := getDefaultDicePoints(ctx)

					dicePoints := "100"
					// TODO: ctx.diceExprOverwrite字段在MsgContext中不存在，暂时注释掉
					// if ctx.diceExprOverwrite != "" {
					//	r, detail, _ = DiceExprEvalBase(ctx, cmdArgs.CleanArgs, RollExtraFlags{
					//		DefaultDiceSideNum: dicePoints,
					//		DisableBlock:       true,
					//	})
					//	if r != nil && r.TypeId == ds.VMTypeInt {
					//		valX, _ := r.ReadInt()
					//		val = int64(valX)
					//	}
					// } else {
					r := ctx.Eval("d", &ds.RollConfig{
						DefaultDiceSideExpr: dicePoints,
						DisableStmts:        true,
					})
					if r != nil && r.TypeId == ds.VMTypeInt {
						valX, _ := r.ReadInt()
						val = int64(valX)
					}
					// }

					// 指令信息标记
					item := map[string]any{
						"expr":       fmt.Sprintf("D%s", dicePoints),
						"reason":     forWhat,
						"dicePoints": dicePoints,
						"result":     val,
					}
					if forWhat == "" {
						delete(item, "reason")
					}
					commandInfoItems = append(commandInfoItems, item)

					VarSetValueStr(ctx, "$t表达式文本", fmt.Sprintf("D%s", dicePoints))
					VarSetValueStr(ctx, "$t计算过程", detail)
					VarSetValueInt64(ctx, "$t计算结果", val)
				}
				return nil
			}

			if cmdArgs.SpecialExecuteTimes > 1 {
				VarSetValueInt64(ctx, "$t次数", int64(cmdArgs.SpecialExecuteTimes))
				// if cmdArgs.SpecialExecuteTimes > int(ctx.Dice.Config.MaxExecuteTime) {
				// 	ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰点_轮数过多警告"))
				// 	return types.CmdExecuteResult{Matched: true, Solved: true}
				// }
				var texts []string
				for range cmdArgs.SpecialExecuteTimes {
					ret := rollOne()
					if ret != nil {
						return *ret
					}
					texts = append(texts, DiceFormatTmpl(ctx, "核心:骰点_单项结果文本"))
				}
				VarSetValueStr(ctx, "$t结果文本", strings.Join(texts, "\n"))
				text = DiceFormatTmpl(ctx, "核心:骰点_多轮")
			} else {
				ret := rollOne()
				if ret != nil {
					return *ret
				}
				VarSetValueStr(ctx, "$t结果文本", DiceFormatTmpl(ctx, "核心:骰点_单项结果文本"))
				text = DiceFormatTmpl(ctx, "核心:骰点")
			}

			isHide := strings.Contains(cmdArgs.Command, "h")

			// 指令信息
			commandInfo := map[string]any{
				"cmd":    "roll",
				"pcName": ctx.Player.Name,
				"items":  commandInfoItems,
			}
			if isHide {
				commandInfo["hide"] = isHide
			}
			// TODO: ctx.CommandInfo字段在MsgContext中不存在，暂时注释掉
			// ctx.CommandInfo = commandInfo
			_ = commandInfo // 避免未使用变量警告

			if kw := cmdArgs.GetKwarg("asm"); r != nil && kw != nil {
				// TODO: ctx.PrivilegeLevel字段在MsgContext中不存在，暂时注释掉
				// if ctx.PrivilegeLevel >= 40 {
				//	asm := r.GetAsmText()
				//	text += "\n" + asm
				// }
			}

			if kw := cmdArgs.GetKwarg("ci"); kw != nil {
				// TODO: ctx.CommandInfo字段在MsgContext中不存在，暂时注释掉
				// info, err := json.Marshal(ctx.CommandInfo)
				// if err == nil {
				//	text += "\n" + string(info)
				// } else {
				//	text += "\n" + "指令信息无法序列化"
				// }
				info, err := json.Marshal(commandInfo)
				if err == nil {
					text += "\n" + string(info)
				} else {
					text += "\n" + "指令信息无法序列化"
				}
			}

			if isHide {
				if msg.Platform == "QQ-CH" {
					ReplyToSender(ctx, msg, "QQ频道内尚不支持暗骰")
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}

				if ctx.Group != nil {
					if ctx.IsPrivate {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
					} else {
						ctx.CommandHideFlag = ctx.Group.GroupId
						prefix := DiceFormatTmpl(ctx, "核心:暗骰_私聊_前缀")
						ReplyGroup(ctx, msg, DiceFormatTmpl(ctx, "核心:暗骰_群内"))
						ReplyPerson(ctx, msg, prefix+text)
					}
				} else {
					ReplyToSender(ctx, msg, text)
				}
				return types.CmdExecuteResult{Matched: true, Solved: true}
			}

			ReplyToSender(ctx, msg, text)
			return types.CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	helpRollX := ".rx <表达式> <原因> // 骰点指令\n.rxh <表达式> <原因> // 暗骰"
	cmdRollX := &types.CmdItemInfo{
		Name:          "roll",
		ShortHelp:     helpRoll,
		Help:          "骰点(和r相同，但支持代骰):\n" + helpRollX,
		AllowDelegate: true,
		Solve: func(ctx *types.MsgContext, msg *types.Message, cmdArgs *types.CmdArgs) types.CmdExecuteResult {
			// mctx := GetCtxProxyFirst(ctx, cmdArgs)
			return cmdRoll.Solve(ctx, msg, cmdArgs)
		},
	}

	textHelp := ".text <文本模板> // 文本指令，例: .text 看看手气: {1d16}"
	cmdText := &types.CmdItemInfo{
		Name:      "text",
		ShortHelp: textHelp,
		Help:      "文本模板指令:\n" + textHelp,
		Solve: func(ctx *types.MsgContext, msg *types.Message, cmdArgs *types.CmdArgs) types.CmdExecuteResult {
			// if ctx.Dice.Config.TextCmdTrustOnly {
			// 	// 检查master和信任权限
			// 	// 拒绝无权限访问
			// 	if ctx.PrivilegeLevel < 70 {
			// 		ReplyToSender(ctx, msg, "你不具备Master权限")
			// 		return CmdExecuteResult{Matched: true, Solved: true}
			// 	}
			// }
			if cmdArgs.IsArgEqual(1, "help") {
				return types.CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			tmpl := ctx.GetCharTemplate()
			ctx.Eval(tmpl.PreloadCode, nil)
			val := cmdArgs.GetArgN(1)

			if val != "" {
				// ctx.Player.TempValueAlias = nil // 防止dnd的hp被转为“生命值”
				r, _, err := ctx.EvalFString(cmdArgs.CleanArgs, &ds.RollConfig{DisableStmts: true})

				if err == nil {
					text := r.ToString()

					if kw := cmdArgs.GetKwarg("asm"); r != nil && kw != nil {
						// if ctx.PrivilegeLevel >= 40 {
						// 	asm := r.GetAsmText()
						// 	text += "\n" + asm
						// }
						text += "\n" + ctx.GetVM().GetAsmText()
					}

					seemsCommand := false
					if strings.HasPrefix(text, ".") || strings.HasPrefix(text, "。") || strings.HasPrefix(text, "!") || strings.HasPrefix(text, "/") {
						seemsCommand = true
						if strings.HasPrefix(text, "..") || strings.HasPrefix(text, "。。") || strings.HasPrefix(text, "!!") {
							seemsCommand = false
						}
					}

					if seemsCommand {
						ReplyToSender(ctx, msg, "你可能在利用text让骰子发出指令文本，这被视为恶意行为并已经记录")
					} else {
						ReplyToSender(ctx, msg, text)
					}
				} else {
					ReplyToSender(ctx, msg, "执行出错:"+err.Error())
				}
				return types.CmdExecuteResult{Matched: true, Solved: true}
			}
			return types.CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		},
	}

	cmdMap["r"] = cmdRoll
	cmdMap["rd"] = cmdRoll
	cmdMap["roll"] = cmdRoll
	cmdMap["rh"] = cmdRoll
	cmdMap["rhd"] = cmdRoll
	cmdMap["rdh"] = cmdRoll
	cmdMap["rx"] = cmdRollX
	cmdMap["rxh"] = cmdRollX
	cmdMap["rhx"] = cmdRollX

	cmdMap["text"] = cmdText

	theExt.CmdMap = cmdMap

	dice.RegisterExtension(theExt)
}
