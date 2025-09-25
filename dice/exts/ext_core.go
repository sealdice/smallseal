package exts

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/sealdice/smallseal/dice/types"

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
			// 		return types.CmdExecuteResult{Matched: true, Solved: true}
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

	cmdBot := &types.CmdItemInfo{
		Name:      "bot",
		ShortHelp: ".bot on/off/about/bye/quit // 开启、关闭、查看信息、退群",
		Help:      "骰子管理:\n.bot on/off/about/bye[exit,quit] // 开启、关闭、查看信息、退群",
		Raw:       true,
		Solve: func(ctx *types.MsgContext, msg *types.Message, cmdArgs *types.CmdArgs) types.CmdExecuteResult {
			inGroup := msg.MessageType == "group"

			ReplyToSender(ctx, msg, "测试用小海豹 v0.1\n海豹核心的精简实现，目前仅有基本骰点和部分coc7指令。\n缺少绝大部分指令，没有数据持久化功能，随时可能关闭，切勿用于跑团")
			return types.CmdExecuteResult{Matched: true, Solved: true}

			if inGroup {
				// 不响应裸指令选项
				// if len(cmdArgs.At) < 1 && ctx.Dice.Config.IgnoreUnaddressedBotCmd {
				// 	return types.CmdExecuteResult{Matched: true, Solved: false}
				// }
				// 不响应at其他人
				if cmdArgs.SomeoneBeMentionedButNotMe {
					return types.CmdExecuteResult{Matched: true, Solved: false}
				}
			}

			if len(cmdArgs.Args) > 0 && !cmdArgs.IsArgEqual(1, "about") { //nolint:nestif
				if cmdArgs.SomeoneBeMentionedButNotMe {
					return types.CmdExecuteResult{Matched: true, Solved: false}
				}

				cmdArgs.ChopPrefixToArgsWith("on", "off")

				matchNumber := func() (bool, bool) {
					txt := cmdArgs.GetArgN(2)
					if len(txt) >= 4 {
						// if strings.HasSuffix(ctx.EndPoint.UserID, txt) {
						// 	return true, txt != ""
						// }
					}
					return false, txt != ""
				}

				isMe, exists := matchNumber()
				if exists && !isMe {
					return types.CmdExecuteResult{Matched: true, Solved: false}
				}

				if cmdArgs.IsArgEqual(1, "on") {
					// if msg.Platform != "QQ-CH" && !ctx.Dice.Config.BotExtFreeSwitch && ctx.PrivilegeLevel < 40 {
					// 	ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_无权限_非master/管理/邀请者"))
					// 	return types.CmdExecuteResult{Matched: true, Solved: true}
					// }

					if ctx.IsPrivate {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
						return types.CmdExecuteResult{Matched: true, Solved: true}
					}

					// SetBotOnAtGroup(ctx, msg.GroupID)
					// TODO：ServiceAtNew此处忽略是否合理？
					// ctx.Group, _ = ctx.Session.ServiceAtNew.Load(msg.GroupID)
					ctx.IsCurGroupBotOn = true

					text := DiceFormatTmpl(ctx, "核心:骰子开启")
					if ctx.Group.LogOn {
						text += "\n请特别注意: 日志记录处于开启状态"
					}
					ReplyToSender(ctx, msg, text)

					return types.CmdExecuteResult{Matched: true, Solved: true}
				} else if cmdArgs.IsArgEqual(1, "off") {
					// if msg.Platform != "QQ-CH" && !ctx.Dice.Config.BotExtFreeSwitch && ctx.PrivilegeLevel < 40 {
					// 	ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_无权限_非master/管理/邀请者"))
					// 	return types.CmdExecuteResult{Matched: true, Solved: true}
					// }

					if ctx.IsPrivate {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
						return types.CmdExecuteResult{Matched: true, Solved: true}
					}

					// SetBotOffAtGroup(ctx, ctx.Group.GroupID)
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰子关闭"))
					return types.CmdExecuteResult{Matched: true, Solved: true}
				} else if cmdArgs.IsArgEqual(1, "bye", "exit", "quit") {
					if cmdArgs.GetArgN(2) != "" {
						return types.CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
					}

					if ctx.IsPrivate {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
						return types.CmdExecuteResult{Matched: true, Solved: true}
					}

					// if ctx.PrivilegeLevel < 40 {
					// 	ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_无权限_非master/管理"))
					// 	return types.CmdExecuteResult{Matched: true, Solved: true}
					// }

					if !cmdArgs.AmIBeMentioned {
						// 裸指令，如果当前群内开启，予以提示
						if ctx.IsCurGroupBotOn {
							ReplyToSender(ctx, msg, "[退群指令] 请@我使用这个命令，以进行确认")
						}
						return types.CmdExecuteResult{Matched: true, Solved: true}
					}

					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰子退群预告"))

					// userName := ctx.Dice.Parent.TryGetUserName(msg.Sender.UserID)
					// txt := fmt.Sprintf("指令退群: 于群组<%s>(%s)中告别，操作者:<%s>(%s)",
					// 	ctx.Group.GroupName, msg.GroupID, userName, msg.Sender.UserID)
					// d.Logger.Info(txt)
					// ctx.Notice(txt)

					// SetBotOffAtGroup(ctx, ctx.Group.GroupID)
					// time.Sleep(3 * time.Second)
					// ctx.Group.DiceIDExistsMap.Delete(ctx.EndPoint.UserID)
					// ctx.Group.UpdatedAtTime = time.Now().Unix()
					// ctx.EndPoint.Adapter.QuitGroup(ctx, msg.GroupID)

					return types.CmdExecuteResult{Matched: true, Solved: true}
				} else if cmdArgs.IsArgEqual(1, "save") {
					// d.Save(false)

					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰子保存设置"))
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}

				return types.CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			if cmdArgs.SomeoneBeMentionedButNotMe {
				return types.CmdExecuteResult{Matched: false, Solved: false}
			}

			activeCount := 0
			serveCount := 0
			// Pinenutn: Range模板 ServiceAtNew重构代码
			// d.ImSession.ServiceAtNew.Range(func(_ string, gp *GroupInfo) bool {
			// 	// Pinenutn: ServiceAtNew重构
			// 	if gp.GroupID != "" &&
			// 		!strings.HasPrefix(gp.GroupID, "PG-") &&
			// 		gp.DiceIDExistsMap.Exists(ctx.EndPoint.UserID) {
			// 		serveCount++
			// 		if gp.DiceIDActiveMap.Exists(ctx.EndPoint.UserID) {
			// 			activeCount++
			// 		}
			// 	}
			// 	return true
			// })

			// onlineVer := ""
			// if d.Parent.AppVersionOnline != nil {
			// 	ver := d.Parent.AppVersionOnline
			// 	// 如果当前不是最新版，那么提示
			// 	if ver.VersionLatestCode != VERSION_CODE {
			// 		onlineVer = "\n最新版本: " + ver.VersionLatestDetail + "\n"
			// 	}
			// }

			// var groupWorkInfo, activeText string
			// if inGroup {
			// 	activeText = "关闭"
			// 	if ctx.Group.IsActive(ctx) {
			// 		activeText = "开启"
			// 	}
			// 	groupWorkInfo = "\n群内工作状态: " + activeText
			// }

			VarSetValueInt64(ctx, "$t供职群数", int64(serveCount))
			VarSetValueInt64(ctx, "$t启用群数", int64(activeCount))
			// VarSetValueStr(ctx, "$t群内工作状态", groupWorkInfo)
			// VarSetValueStr(ctx, "$t群内工作状态_仅状态", activeText)
			// ver := VERSION.String()
			// arch := runtime.GOARCH
			// if arch != "386" && arch != "amd64" {
			// 	ver = fmt.Sprintf("%s %s", ver, arch)
			// }
			// baseText := fmt.Sprintf("SealDice %s%s", ver, onlineVer)
			extText := DiceFormatTmpl(ctx, "核心:骰子状态附加文本")
			if extText != "" {
				extText = "\n" + extText
			}
			// text := baseText + extText

			// ReplyToSender(ctx, msg, text)

			return types.CmdExecuteResult{Matched: true, Solved: true}
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

	helpExt := ".ext list // 查看所有扩展状态\n.ext on <扩展名> // 开启指定扩展\n.ext off <扩展名> // 关闭指定扩展"
	cmdExt := &types.CmdItemInfo{
		Name:      "ext",
		ShortHelp: helpExt,
		Help:      "扩展管理:\n" + helpExt,
		Solve: func(ctx *types.MsgContext, msg *types.Message, cmdArgs *types.CmdArgs) types.CmdExecuteResult {
			if cmdArgs.IsArgEqual(1, "help") {
				return types.CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			if msg.MessageType != "group" {
				ReplyToSender(ctx, msg, "扩展管理指令仅在群聊中可用")
				return types.CmdExecuteResult{Matched: true, Solved: true}
			}

			switch cmdArgs.GetArgN(1) {
			case "list":
				var result strings.Builder
				result.WriteString("当前扩展状态:\n")
				for _, ext := range ctx.Dice.GetExtList() {
					status := "❌ 关闭"
					if ctx.Group.IsExtensionActive(ext.Name) {
						status = "✅ 开启"
					}
					result.WriteString(fmt.Sprintf("%s: %s\n", ext.Name, status))
				}
				ReplyToSender(ctx, msg, result.String())

			case "on":
				extName := cmdArgs.GetArgN(2)
				if extName == "" {
					ReplyToSender(ctx, msg, "请指定要开启的扩展名")
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}

				// 查找扩展是否存在
				var targetExt *types.ExtInfo
				for _, ext := range ctx.Dice.GetExtList() {
					if ext.Name == extName {
						targetExt = ext
						break
					}
				}

				if targetExt == nil {
					ReplyToSender(ctx, msg, fmt.Sprintf("未找到扩展: %s", extName))
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}

				if ctx.Group.IsExtensionActive(extName) {
					ReplyToSender(ctx, msg, fmt.Sprintf("扩展 %s 已经是开启状态", extName))
				} else {
					ctx.Group.SetExtensionActive(extName, true)
					// 更新ActivatedExtList
					ctx.Group.ActivatedExtList = ctx.Group.GetActiveExtensions(ctx.Dice.GetExtList())
					ReplyToSender(ctx, msg, fmt.Sprintf("已开启扩展: %s", extName))
				}

			case "off":
				extName := cmdArgs.GetArgN(2)
				if extName == "" {
					ReplyToSender(ctx, msg, "请指定要关闭的扩展名")
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}

				// 不允许关闭core扩展
				if extName == "core" {
					ReplyToSender(ctx, msg, "核心扩展不能被关闭")
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}

				// 查找扩展是否存在
				var targetExt *types.ExtInfo
				for _, ext := range ctx.Dice.GetExtList() {
					if ext.Name == extName {
						targetExt = ext
						break
					}
				}

				if targetExt == nil {
					ReplyToSender(ctx, msg, fmt.Sprintf("未找到扩展: %s", extName))
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}

				if !ctx.Group.IsExtensionActive(extName) {
					ReplyToSender(ctx, msg, fmt.Sprintf("扩展 %s 已经是关闭状态", extName))
				} else {
					ctx.Group.SetExtensionActive(extName, false)
					// 更新ActivatedExtList
					ctx.Group.ActivatedExtList = ctx.Group.GetActiveExtensions(ctx.Dice.GetExtList())
					ReplyToSender(ctx, msg, fmt.Sprintf("已关闭扩展: %s", extName))
				}

			default:
				return types.CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			return types.CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	cmdMap["text"] = cmdText
	cmdMap["bot"] = cmdBot
	cmdMap["ext"] = cmdExt

	theExt.CmdMap = cmdMap

	dice.RegisterExtension(theExt)
}
