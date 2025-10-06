package exts

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
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

			if (cmdArgs.Command == "rd" || cmdArgs.Command == "rhd" || cmdArgs.Command == "rdh") && len(cmdArgs.Args) >= 1 {
				if m, _ := regexp.MatchString(`^\d|优势|劣势|\+|-`, cmdArgs.CleanArgs); m {
					if cmdArgs.IsSpaceBeforeArgs {
						cmdArgs.CleanArgs = "d " + cmdArgs.CleanArgs
					} else {
						cmdArgs.CleanArgs = "d" + cmdArgs.CleanArgs
					}
				}
			}

			var r *ds.VMValue
			var commandInfoItems []any

			rollOne := func() *types.CmdExecuteResult {
				forWhat := ""
				var matched string

				vm := ctx.GetVM()
				if len(cmdArgs.Args) >= 1 { //nolint:nestif
					var err error
					r, detail, err = ctx.EvalBase(cmdArgs.CleanArgs, &ds.RollConfig{
						DefaultDiceSideExpr: ctx.GetDefaultDicePoints(),
						DisableStmts:        true,
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
							DefaultDiceSideExpr: ctx.GetDefaultDicePoints(),
							DisableStmts:        true,
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

					dicePoints := ctx.GetDefaultDicePoints()
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
						DefaultDiceSideExpr: ctx.GetDefaultDicePoints(),
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
			mctx := GetCtxProxyFirst(ctx, cmdArgs)
			return cmdRoll.Solve(mctx, msg, cmdArgs)
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
			ctx.Eval(tmpl.InitScript, nil)
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

	collectIDs := func(ctx *types.MsgContext, cmdArgs *types.CmdArgs, includeArgs bool) []string {
		seen := map[string]struct{}{}
		add := func(id string) {
			id = strings.TrimSpace(id)
			if id == "" {
				return
			}
			seen[id] = struct{}{}
		}
		for _, at := range cmdArgs.At {
			add(at.UserID)
		}
		if includeArgs {
			for _, raw := range cmdArgs.Args[1:] {
				raw = strings.TrimSpace(raw)
				if raw == "" {
					continue
				}
				if strings.EqualFold(raw, "me") {
					add(ctx.Player.UserId)
					continue
				}
				add(raw)
			}
		}
		ids := make([]string, 0, len(seen))
		for id := range seen {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		return ids
	}

	getActiveExts := func(ctx *types.MsgContext) []*types.ExtInfo {
		if ctx.Group != nil {
			return ctx.Group.GetActiveExtensions(ctx.Dice.GetExtList())
		}
		return ctx.Dice.GetExtList()
	}

	cmdHelp := &types.CmdItemInfo{
		Name:      "help",
		ShortHelp: ".help [指令] // 查看帮助",
		Help:      "帮助:\n.help [指令] // 查看指定指令帮助\n.help // 查看当前可用指令",
		Solve: func(ctx *types.MsgContext, msg *types.Message, cmdArgs *types.CmdArgs) types.CmdExecuteResult {
			if cmdArgs.IsArgEqual(1, "help") {
				return types.CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			target := strings.ToLower(cmdArgs.GetArgN(1))
			activeExts := getActiveExts(ctx)
			lookup := map[string]*types.CmdItemInfo{}
			for _, ext := range activeExts {
				for name, info := range ext.CmdMap {
					if info == nil {
						continue
					}
					canonical := info.Name
					if canonical == "" {
						canonical = name
					}
					for _, key := range []string{name, canonical} {
						key = strings.ToLower(key)
						if key == "" {
							continue
						}
						if _, exists := lookup[key]; !exists {
							lookup[key] = info
						}
					}
				}
			}
			if target != "" {
				if info, ok := lookup[target]; ok {
					text := info.Help
					if text == "" {
						text = info.ShortHelp
					}
					if text == "" {
						text = fmt.Sprintf("指令 %s 暂无帮助信息", target)
					}
					ReplyToSender(ctx, msg, text)
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}
				ReplyToSender(ctx, msg, fmt.Sprintf("未找到指令: %s", target))
				return types.CmdExecuteResult{Matched: true, Solved: true}
			}

			var builder strings.Builder
			builder.WriteString("可用指令列表:\n")
			for _, ext := range activeExts {
				entries := map[string]string{}
				for name, info := range ext.CmdMap {
					if info == nil {
						continue
					}
					canonical := info.Name
					if canonical == "" {
						canonical = name
					}
					key := strings.ToLower(canonical)
					if _, exists := entries[key]; exists {
						continue
					}
					short := info.ShortHelp
					if short == "" {
						short = "." + canonical
					}
					entries[key] = short
				}
				if len(entries) == 0 {
					continue
				}
				builder.WriteString(fmt.Sprintf("[%s]\n", ext.Name))
				keys := make([]string, 0, len(entries))
				for key := range entries {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				for _, key := range keys {
					entry := entries[key]
					builder.WriteString(entry)
					if !strings.HasSuffix(entry, "\n") {
						builder.WriteString("\n")
					}
				}
			}
			ReplyToSender(ctx, msg, builder.String())
			return types.CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	cmdBot := &types.CmdItemInfo{
		Name:      "bot",
		ShortHelp: ".bot on/off/about/bye // 管理骰子状态",
		Help:      "骰子管理:\n.bot on // 开启当前群服务\n.bot off // 关闭当前群服务\n.bot about // 查看骰子信息\n.bot bye // 退群前提示",
		Raw:       true,
		Solve: func(ctx *types.MsgContext, msg *types.Message, cmdArgs *types.CmdArgs) types.CmdExecuteResult {
			if cmdArgs.IsArgEqual(1, "help") {
				return types.CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			action := strings.ToLower(cmdArgs.GetArgN(1))
			if action == "" {
				action = "about"
			}
			persistGroupState := func() {
				if ctx.Dice == nil || ctx.Group == nil {
					return
				}
				groupID := msg.GroupID
				if groupID == "" {
					groupID = ctx.Group.GroupId
				}
				if groupID == "" {
					return
				}
				ctx.Dice.PersistGroupInfo(groupID, ctx.Group)
			}
			switch action {
			case "on":
				if ctx.IsPrivate || ctx.Group == nil {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}
				ctx.Group.Active = true
				ctx.IsCurGroupBotOn = true
				persistGroupState()
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰子开启"))
			case "off":
				if ctx.IsPrivate || ctx.Group == nil {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}
				ctx.Group.Active = false
				ctx.IsCurGroupBotOn = false
				persistGroupState()
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰子关闭"))
			case "bye", "quit", "exit":
				if ctx.IsPrivate || ctx.Group == nil {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰子退群预告"))
			case "about":
				info := "测试用小海豹 v0.1\n海豹核心的精简实现，目前仅有基本骰点和部分coc7指令。\n缺少绝大部分指令，没有数据持久化功能，随时可能关闭，切勿用于跑团"
				ReplyToSender(ctx, msg, info)
			default:
				return types.CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			return types.CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	cmdDismiss := &types.CmdItemInfo{
		Name:              "dismiss",
		ShortHelp:         ".dismiss // 退群别名",
		Help:              "退群(映射到 .bot bye):\n.dismiss",
		Raw:               true,
		DisabledInPrivate: true,
		Solve: func(ctx *types.MsgContext, msg *types.Message, cmdArgs *types.CmdArgs) types.CmdExecuteResult {
			if ctx.IsPrivate {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
				return types.CmdExecuteResult{Matched: true, Solved: true}
			}
			cloned := *cmdArgs
			cloned.Args = append([]string{"bye"}, cmdArgs.Args...)
			if cmdArgs.CleanArgs != "" {
				cloned.CleanArgs = strings.TrimSpace("bye " + cmdArgs.CleanArgs)
			} else {
				cloned.CleanArgs = "bye"
			}
			cloned.RawArgs = cloned.CleanArgs
			return cmdBot.Solve(ctx, msg, &cloned)
		},
	}

	botListHelp := ".botlist add @A ... // 标记其他机器人\n.botlist del @A ... // 移除标记\n.botlist list // 查看当前列表"
	cmdBotList := &types.CmdItemInfo{
		Name:      "botlist",
		ShortHelp: botListHelp,
		Help:      "机器人列表:\n" + botListHelp,
		Raw:       true,
		Solve: func(ctx *types.MsgContext, msg *types.Message, cmdArgs *types.CmdArgs) types.CmdExecuteResult {
			if ctx.IsPrivate || ctx.Group == nil {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
				return types.CmdExecuteResult{Matched: true, Solved: true}
			}
			action := strings.ToLower(cmdArgs.GetArgN(1))
			if action == "" {
				action = "list"
			}
			ctx.Group.EnsureBotList()
			switch action {
			case "add":
				ids := collectIDs(ctx, cmdArgs, true)
				if len(ids) == 0 {
					ReplyToSender(ctx, msg, "请通过@或参数指定要标记的账号")
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}
				for _, id := range ids {
					ctx.Group.BotList.Store(id, true)
				}
				ReplyToSender(ctx, msg, fmt.Sprintf("已标记 %d 个账号为机器人", len(ids)))
			case "del", "rm":
				ids := collectIDs(ctx, cmdArgs, true)
				if len(ids) == 0 {
					ReplyToSender(ctx, msg, "请通过@或参数指定要移除的账号")
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}
				removed := 0
				for _, id := range ids {
					if ctx.Group.BotList.Delete(id) {
						removed++
					}
				}
				ReplyToSender(ctx, msg, fmt.Sprintf("已移除 %d 个机器人标记", removed))
			case "list", "show":
				bots := []string{}
				ctx.Group.BotList.Range(func(id string, _ bool) bool {
					bots = append(bots, id)
					return true
				})
				if len(bots) == 0 {
					ReplyToSender(ctx, msg, "当前未配置其他机器人")
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}
				sort.Strings(bots)
				ReplyToSender(ctx, msg, "群内其他机器人列表:\n- "+strings.Join(bots, "\n- "))
			default:
				return types.CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			return types.CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	masterHelp := ".master add @A // 添加骰主\n.master del @A // 移除骰主\n.master list // 查看骰主列表"
	cmdMaster := &types.CmdItemInfo{
		Name:          "master",
		ShortHelp:     masterHelp,
		Help:          "骰主指令(精简版):\n" + masterHelp,
		AllowDelegate: true,
		Solve: func(ctx *types.MsgContext, msg *types.Message, cmdArgs *types.CmdArgs) types.CmdExecuteResult {
			action := strings.ToLower(cmdArgs.GetArgN(1))
			switch action {
			case "help":
				return types.CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			case "add":
				ids := collectIDs(ctx, cmdArgs, true)
				if len(ids) == 0 {
					ReplyToSender(ctx, msg, "请通过@或参数指定要添加的账号")
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}
				for _, id := range ids {
					ctx.Dice.MasterAdd(id)
				}
				ReplyToSender(ctx, msg, fmt.Sprintf("已添加 %d 位master", len(ids)))
			case "del", "rm":
				ids := collectIDs(ctx, cmdArgs, true)
				if len(ids) == 0 {
					ReplyToSender(ctx, msg, "请通过@或参数指定要移除的账号")
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}
				removed := 0
				for _, id := range ids {
					if ctx.Dice.MasterRemove(id) {
						removed++
					}
				}
				ReplyToSender(ctx, msg, fmt.Sprintf("已移除 %d 位master", removed))
			case "list", "":
				masters := ctx.Dice.ListMasters()
				if len(masters) == 0 {
					ReplyToSender(ctx, msg, "尚未配置master")
				} else {
					ReplyToSender(ctx, msg, "当前master列表:\n- "+strings.Join(masters, "\n- "))
				}
			default:
				ReplyToSender(ctx, msg, "当前实现仅支持 add/del/list 指令")
			}
			return types.CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	helpNN := ".nn // 查看当前角色名\n.nn <角色名> // 设置角色名\n.nn clr // 重置为平台昵称"
	cmdNN := &types.CmdItemInfo{
		Name:      "nn",
		ShortHelp: helpNN,
		Help:      "角色名设置:\n" + helpNN,
		Solve: func(ctx *types.MsgContext, msg *types.Message, cmdArgs *types.CmdArgs) types.CmdExecuteResult {
			val := strings.ToLower(cmdArgs.GetArgN(1))
			switch val {
			case "":
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:昵称_当前"))
			case "help":
				return types.CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			case "clr", "reset":
				newName := msg.Sender.Nickname
				VarSetValueStr(ctx, "$t旧昵称", fmt.Sprintf("<%s>", ctx.Player.Name))
				VarSetValueStr(ctx, "$t旧昵称_RAW", ctx.Player.Name)
				ctx.Player.Name = newName
				ctx.Player.UpdatedAtTime = time.Now().Unix()
				VarSetValueStr(ctx, "$t玩家", fmt.Sprintf("<%s>", ctx.Player.Name))
				VarSetValueStr(ctx, "$t玩家_RAW", ctx.Player.Name)
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:昵称_重置"))
			default:
				clean := strings.TrimSpace(cmdArgs.CleanArgs)
				if clean == "" {
					clean = cmdArgs.GetArgN(1)
				}
				VarSetValueStr(ctx, "$t旧昵称", fmt.Sprintf("<%s>", ctx.Player.Name))
				VarSetValueStr(ctx, "$t旧昵称_RAW", ctx.Player.Name)
				ctx.Player.Name = clean
				ctx.Player.UpdatedAtTime = time.Now().Unix()
				VarSetValueStr(ctx, "$t玩家", fmt.Sprintf("<%s>", ctx.Player.Name))
				VarSetValueStr(ctx, "$t玩家_RAW", ctx.Player.Name)
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:昵称_改名"))
			}
			if ctx.Group != nil {
				ctx.Group.Players.Store(ctx.Player.UserId, ctx.Player)
			}
			return types.CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	cmdUserID := &types.CmdItemInfo{
		Name:      "userid",
		ShortHelp: ".userid // 查看当前帐号和群组ID",
		Help:      "查看ID:\n.userid // 查看当前帐号和群组ID",
		Solve: func(ctx *types.MsgContext, msg *types.Message, cmdArgs *types.CmdArgs) types.CmdExecuteResult {
			if cmdArgs.IsArgEqual(1, "help") {
				return types.CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			text := fmt.Sprintf("个人账号ID为 %s", ctx.Player.UserId)
			if ctx.Group != nil && !ctx.IsPrivate {
				text += fmt.Sprintf("\n当前群组ID为 %s", ctx.Group.GroupId)
			}
			ReplyToSender(ctx, msg, text)
			return types.CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	helpSet := ".set info // 查看当前面数\n.set <面数或表达式> // 设置群默认骰子面数\n.set dnd // 切换到DND模式\n.set coc // 切换到COC模式\n.set clr // 清除群内面数设置"
	cmdSet := &types.CmdItemInfo{
		Name:      "set",
		ShortHelp: helpSet,
		Help:      "设定骰子面数:\n" + helpSet,
		Solve: func(ctx *types.MsgContext, msg *types.Message, cmdArgs *types.CmdArgs) types.CmdExecuteResult {
			if cmdArgs.IsArgEqual(1, "help") {
				return types.CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			if ctx.Group == nil {
				ReplyToSender(ctx, msg, "仅群聊中可设置默认面数")
				return types.CmdExecuteResult{Matched: true, Solved: true}
			}
			persistGroupState := func() {
				if ctx.Dice == nil || ctx.Group == nil {
					return
				}
				groupID := msg.GroupID
				if groupID == "" {
					groupID = ctx.Group.GroupId
				}
				if groupID != "" {
					ctx.Dice.PersistGroupInfo(groupID, ctx.Group)
				}
			}
			sub := strings.ToLower(cmdArgs.GetArgN(1))
			switch sub {
			case "", "info":
				expr := ctx.Group.DiceSideExpr
				if expr == "" {
					expr = ctx.GetDefaultDicePoints()
				}
				text := fmt.Sprintf("当前群默认骰子面数: %s", strings.ToUpper(expr))
				if ctx.Group.System != "" {
					text += fmt.Sprintf("\n当前规则模板: %s", ctx.Group.System)
				}
				ReplyToSender(ctx, msg, text)
			case "clr":
				ctx.Group.DiceSideExpr = ""
				ctx.Group.System = ""
				ctx.Group.UpdatedAtTime = time.Now().Unix()
				persistGroupState()
				ReplyToSender(ctx, msg, "已清除群默认骰子面数设置")

			case "list":
				text := "当前可用规则:\n"
				idx := 1
				ctx.Dice.GameSystemMapGet().Range(func(key string, tmpl *types.GameSystemTemplateV2) bool {
					text += fmt.Sprintf("%2d. [%s] %s %s\n", idx, tmpl.Name, tmpl.FullName, tmpl.Commands.Set.DiceSideExpr)
					idx++
					return true
				})
				text += "\n可使用 .set <规则名> 切换规则，如 .set coc7"
				ReplyToSender(ctx, msg, text)
			default:
				var found bool
				ctx.Dice.GameSystemMapGet().Range(func(key string, tmpl *types.GameSystemTemplateV2) bool {
					isMatch := false
					for _, k := range tmpl.Commands.Set.Keys {
						if strings.EqualFold(sub, k) {
							isMatch = true
							break
						}
					}
					if isMatch {
						ctx.Group.System = key
						ctx.GameSystem = tmpl
						ctx.Group.DiceSideExpr = tmpl.Commands.Set.DiceSideExpr
						ctx.Group.UpdatedAtTime = time.Now().Unix()

						extNames := []string{}
						for _, name := range tmpl.Commands.Set.RelatedExt {
							// 开启相关扩展
							ei := ctx.Dice.ExtFind(name, false)
							extNames = append(extNames, name)
							if ei != nil {
								ctx.Group.ExtActive(ei)
								ctx.Group.ActivatedExtList = ctx.Group.GetActiveExtensions(ctx.Dice.GetExtList())
							}
						}

						ReplyToSender(ctx, msg, fmt.Sprintf("已切换至 %s(%s) 规则，默认骰子面数 %s，自动启用关联扩展: %s", tmpl.FullName, tmpl.Name, strings.ToUpper(tmpl.Commands.Set.DiceSideExpr), strings.Join(extNames, ", ")))
						persistGroupState()
						found = true
						return false
					}
					return true
				})
				if found {
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}

				expr := strings.ToLower(cmdArgs.GetArgN(1))
				if expr == "" {
					return types.CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}
				if _, err := strconv.Atoi(expr); err == nil {
					expr = "d" + expr
				}
				ctx.Group.DiceSideExpr = expr
				ctx.Group.UpdatedAtTime = time.Now().Unix()
				persistGroupState()
				ReplyToSender(ctx, msg, fmt.Sprintf("已将群默认骰子面数设置为 %s", strings.ToUpper(expr)))
			}
			return types.CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	helpCh := ".pc new <角色名> // 新建角色并绑卡\n" +
		".pc tag [<角色名> | <角色序号>] // 当前群绑卡/解除绑卡(不填角色名)\n" +
		".pc untagAll [<角色名> | <角色序号>] // 全部群解绑(不填即当前卡)\n" +
		".pc list // 列出当前角色和序号\n" +
		".pc rename <新角色名> // 将当前绑定角色改名\n" +
		".pc rename <角色名|序号> <新角色名> // 将指定角色改名 \n" +
		".pc save [<角色名>] // [不绑卡]保存角色，角色名可省略\n" +
		".pc load (<角色名> | <角色序号>) // [不绑卡]加载角色\n" +
		".pc del/rm (<角色名> | <角色序号>) // 删除角色 角色序号可用pc list查询\n" +
		"> 注: 海豹各群数据独立(多张空白卡)，单群游戏不需要存角色。"

	cmdChar := &types.CmdItemInfo{
		Name:      "pc",
		ShortHelp: helpCh,
		Help:      "角色管理:\n" + helpCh,
		Solve: func(ctx *types.MsgContext, msg *types.Message, cmdArgs *types.CmdArgs) (result types.CmdExecuteResult) {
			if ctx.AttrsManager == nil {
				ReplyToSender(ctx, msg, "角色系统尚未初始化")
				return types.CmdExecuteResult{Matched: true, Solved: true}
			}

			cmdArgs.ChopPrefixToArgsWith("list", "lst", "load", "save", "del", "rm", "new", "tag", "untagAll", "rename")
			val1 := strings.ToLower(cmdArgs.GetArgN(1))
			am := ctx.AttrsManager

			defer func() {
				if rec := recover(); rec != nil {
					if err, ok := rec.(error); ok {
						ReplyToSender(ctx, msg, fmt.Sprintf("错误: %s\n", err.Error()))
						result = types.CmdExecuteResult{Matched: true, Solved: true}
					} else {
						panic(rec)
					}
				}
			}()

			getNicknameRaw := func(usePlayerName bool, tryIndex bool) string {
				name := cmdArgs.CleanArgsChopRest

				if tryIndex {
					if idx, err := strconv.ParseInt(name, 10, 64); err == nil && idx > 0 {
						items := lo.Must(am.GetCharacterList(ctx.Player.UserId))
						if idx <= int64(len(items)) {
							return items[idx-1].Name
						}
					}
				}

				if usePlayerName && name == "" {
					name = ctx.Player.Name
				}
				name = strings.ReplaceAll(name, "\n", "")
				name = strings.ReplaceAll(name, "\r", "")
				if len(name) > 90 {
					name = name[:90]
				}
				return name
			}

			getNickname := func() string {
				return getNicknameRaw(true, true)
			}

			getBindingId := func() string {
				id, _ := am.CharGetBindingId(ctx.Group.GroupId, ctx.Player.UserId)
				return id
			}

			setCurPlayerName := func(name string) {
				ctx.Player.Name = name
				ctx.Player.UpdatedAtTime = time.Now().Unix()
				if ctx.Group != nil {
					ctx.Group.Players.Store(ctx.Player.UserId, ctx.Player)
				}
			}

			switch val1 {
			case "list", "lst":
				list := lo.Must(am.GetCharacterList(ctx.Player.UserId))
				bindingId := getBindingId()

				if len(list) == 0 {
					ReplyToSender(ctx, msg, fmt.Sprintf("<%s>当前还没有角色列表", ctx.Player.Name))
				} else {
					rows := make([]string, 0, len(list))
					for idx, item := range list {
						prefix := "[×]"
						if item.BindingGroupsNum > 0 {
							prefix = "[★]"
						}
						if bindingId == item.ID {
							prefix = "[√]"
						}
						suffix := ""
						if item.SheetType != "" {
							suffix = fmt.Sprintf(" #%s", item.SheetType)
						}
						rows = append(rows, fmt.Sprintf("%2d %s %s%s", idx+1, prefix, item.Name, suffix))
					}
					ReplyToSender(ctx, msg, fmt.Sprintf("<%s>的角色列表为:\n%s\n[√]已绑 [×]未绑 [★]其他群绑定", ctx.Player.Name, strings.Join(rows, "\n")))
				}
				return types.CmdExecuteResult{Matched: true, Solved: true}
			case "new":
				name := getNicknameRaw(true, false)
				if name == "" {
					return types.CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}

				VarSetValueStr(ctx, "$t角色名", name)
				if !am.CharCheckExists(ctx.Player.UserId, name) {
					item := lo.Must(am.CharNew(ctx.Player.UserId, name, ctx.Group.System))
					lo.Must0(am.CharBind(item.ID, ctx.Group.GroupId, ctx.Player.UserId))
					setCurPlayerName(name)

					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_新建"))
				} else {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_新建_已存在"))
				}

				if ctx.Player.AutoSetNameTemplate != "" {
					_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
				}
				return types.CmdExecuteResult{Matched: true, Solved: true}
			case "rename":
				a := cmdArgs.GetArgN(2)
				b := cmdArgs.GetArgN(3)
				var charId string

				if b == "" {
					b = a
					charId = getBindingId()
				} else {
					charId, _ = am.CharIdGetByName(ctx.Player.UserId, a)
				}

				if a != "" && b != "" {
					if charId != "" {
						if !am.CharCheckExists(ctx.Player.UserId, b) {
							attrs := lo.Must(am.LoadById(charId))
							attrs.Name = b
							if charId == getBindingId() {
								setCurPlayerName(b)
							}
							lo.Must0(am.Save(attrs))
							ReplyToSender(ctx, msg, "操作完成")
						} else {
							ReplyToSender(ctx, msg, "此角色名已存在")
						}
					} else {
						ReplyToSender(ctx, msg, "未找到此角色")
					}
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}
			case "tag":
				name := getNicknameRaw(false, true)

				VarSetValueStr(ctx, "$t角色名", name)
				if name != "" {
					charId := lo.Must(am.CharIdGetByName(ctx.Player.UserId, name))
					if charId == "" {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_绑定_失败"))
					} else {
						lo.Must0(am.CharBind(charId, ctx.Group.GroupId, ctx.Player.UserId))
						setCurPlayerName(name)
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_绑定_成功"))
					}
				} else {
					charId := getBindingId()
					if charId == "" {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_绑定_并未绑定"))
					} else {
						lo.Must0(am.CharBind("", ctx.Group.GroupId, ctx.Player.UserId))
						attrs := lo.Must(am.LoadById(charId))
						name = attrs.Name
						setCurPlayerName(name)
						VarSetValueStr(ctx, "$t角色名", name)
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_绑定_解除"))
					}
				}
				if ctx.Player.AutoSetNameTemplate != "" {
					_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
				}
				return types.CmdExecuteResult{Matched: true, Solved: true}
			case "load":
				name := getNicknameRaw(false, true)
				VarSetValueStr(ctx, "$t角色名", name)

				charId := lo.Must(am.CharIdGetByName(ctx.Player.UserId, name))
				if charId == "" {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_角色不存在"))
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}

				attrsCur := lo.Must(am.Load(ctx.Group.GroupId, ctx.Player.UserId))
				if attrsCur == nil {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_角色不存在"))
				} else {
					attrs := lo.Must(am.LoadById(charId))
					attrsCur.Clear()
					attrs.Range(func(key string, value *ds.VMValue) bool {
						attrsCur.Store(key, value)
						return true
					})
					lo.Must0(am.Save(attrsCur))
					setCurPlayerName(name)

					if ctx.Player.AutoSetNameTemplate != "" {
						_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
					}

					VarSetValueStr(ctx, "$t玩家", fmt.Sprintf("<%s>", ctx.Player.Name))
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_加载成功"))
				}
				return types.CmdExecuteResult{Matched: true, Solved: true}
			case "save":
				name := getNickname()
				if name == "" {
					return types.CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}

				if !am.CharCheckExists(ctx.Player.UserId, name) {
					newItem, _ := am.CharNew(ctx.Player.UserId, name, ctx.Group.System)
					attrs := lo.Must(am.Load(ctx.Group.GroupId, ctx.Player.UserId))

					if newItem != nil {
						attrsNew := lo.Must(am.LoadById(newItem.ID))
						attrs.Range(func(key string, value *ds.VMValue) bool {
							attrsNew.Store(key, value)
							return true
						})
						lo.Must0(am.Save(attrsNew))
						VarSetValueStr(ctx, "$t角色名", name)
						VarSetValueStr(ctx, "$t新角色名", fmt.Sprintf("<%s>", name))
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_储存成功"))
					} else {
						VarSetValueStr(ctx, "$t角色名", name)
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_储存失败_已绑定"))
					}
				} else {
					ReplyToSender(ctx, msg, "此角色名已存在")
				}
				return types.CmdExecuteResult{Matched: true, Solved: true}
			case "untagall":
				name := getNicknameRaw(false, true)
				var charId string
				if name == "" {
					charId = getBindingId()
				} else {
					charId, _ = am.CharIdGetByName(ctx.Player.UserId, name)
				}

				var lst []string
				if charId != "" {
					lst = am.CharUnbindAll(charId)
				}

				for _, gid := range lst {
					if gid == ctx.Group.GroupId {
						ctx.Player.Name = msg.Sender.Nickname
						ctx.Player.UpdatedAtTime = time.Now().Unix()
						if ctx.Player.AutoSetNameTemplate != "" {
							_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
						}
					}
				}

				if len(lst) > 0 {
					ReplyToSender(ctx, msg, "绑定已全部解除:\n"+strings.Join(lst, "\n"))
				} else {
					ReplyToSender(ctx, msg, "这张卡片并未绑定到任何群")
				}
				return types.CmdExecuteResult{Matched: true, Solved: true}
			case "del", "rm":
				name := getNicknameRaw(false, true)
				if name == "" {
					return types.CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}

				VarSetValueStr(ctx, "$t角色名", name)
				charId, _ := am.CharIdGetByName(ctx.Player.UserId, name)
				if charId == "" {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_角色不存在"))
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}

				lst := am.CharGetBindingGroupIdList(charId)
				if len(lst) > 0 {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_删除失败_已绑定"))
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}

				if err := am.CharDelete(charId); err != nil {
					ReplyToSender(ctx, msg, "角色删除失败")
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}

				VarSetValueStr(ctx, "$t角色名", name)
				VarSetValueStr(ctx, "$t新角色名", fmt.Sprintf("<%s>", name))

				text := DiceFormatTmpl(ctx, "核心:角色管理_删除成功")
				if getBindingId() == charId {
					VarSetValueStr(ctx, "$t新角色名", fmt.Sprintf("<%s>", msg.Sender.Nickname))
					text += "\n" + DiceFormatTmpl(ctx, "核心:角色管理_删除成功_当前卡")
					ctx.Player.Name = msg.Sender.Nickname
				}
				ReplyToSender(ctx, msg, text)
				return types.CmdExecuteResult{Matched: true, Solved: true}
			}
			return types.CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		},
	}

	helpExt := ".ext list // 查看所有扩展状态\n.ext on <扩展名> // 开启指定扩展\n.ext off <扩展名> // 关闭指定扩展"
	cmdExt := &types.CmdItemInfo{
		Name:      "ext",
		ShortHelp: helpExt,
		Help:      "扩展管理:\n" + helpExt,
		Solve: func(ctx *types.MsgContext, msg *types.Message, cmdArgs *types.CmdArgs) types.CmdExecuteResult {
			if cmdArgs.IsArgEqual(1, "help") {
				return types.CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			if ctx.Group == nil {
				ReplyToSender(ctx, msg, "扩展管理指令仅在群聊中可用")
				return types.CmdExecuteResult{Matched: true, Solved: true}
			}
			action := strings.ToLower(cmdArgs.GetArgN(1))
			if action == "" {
				action = "list"
			}
			switch action {
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
				if len(cmdArgs.Args) < 2 {
					ReplyToSender(ctx, msg, "请指定要开启的扩展名")
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}
				var opened []string
				var missing []string
				for _, raw := range cmdArgs.Args[1:] {
					ext := ctx.Dice.ExtFind(raw, false)
					if ext == nil {
						missing = append(missing, raw)
						continue
					}
					ctx.Group.ExtActive(ext)
					opened = append(opened, ext.Name)
				}
				ctx.Group.ActivatedExtList = ctx.Group.GetActiveExtensions(ctx.Dice.GetExtList())
				parts := []string{}
				if len(opened) > 0 {
					parts = append(parts, fmt.Sprintf("已开启扩展: %s", strings.Join(opened, ", ")))
				}
				if len(missing) > 0 {
					parts = append(parts, fmt.Sprintf("未找到: %s", strings.Join(missing, ", ")))
				}
				ReplyToSender(ctx, msg, strings.Join(parts, "\n"))
			case "off":
				if len(cmdArgs.Args) < 2 {
					ReplyToSender(ctx, msg, "请指定要关闭的扩展名")
					return types.CmdExecuteResult{Matched: true, Solved: true}
				}
				var closed []string
				var failed []string
				for _, raw := range cmdArgs.Args[1:] {
					if strings.EqualFold(raw, "core") {
						failed = append(failed, raw)
						continue
					}
					removed := ctx.Group.ExtInactiveByName(raw)
					if removed == nil {
						ext := ctx.Dice.ExtFind(raw, false)
						if ext == nil || strings.EqualFold(ext.Name, "core") {
							failed = append(failed, raw)
							continue
						}
						removed = ctx.Group.ExtInactiveByName(ext.Name)
					}
					if removed != nil {
						closed = append(closed, removed.Name)
					} else {
						failed = append(failed, raw)
					}
				}
				ctx.Group.ActivatedExtList = ctx.Group.GetActiveExtensions(ctx.Dice.GetExtList())
				parts := []string{}
				if len(closed) > 0 {
					parts = append(parts, fmt.Sprintf("已关闭扩展: %s", strings.Join(closed, ", ")))
				}
				if len(failed) > 0 {
					parts = append(parts, fmt.Sprintf("未找到或不可关闭: %s", strings.Join(failed, ", ")))
				}
				ReplyToSender(ctx, msg, strings.Join(parts, "\n"))
			default:
				ext := ctx.Dice.ExtFind(action, false)
				if ext == nil {
					return types.CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}
				desc := ""
				if ext.GetDescText != nil {
					desc = ext.GetDescText(ext)
				} else if ext.Brief != "" {
					desc = ext.Brief
				}
				info := fmt.Sprintf("> [%s] 版本%s 作者%s", ext.Name, ext.Version, ext.Author)
				if desc != "" {
					info += "\n" + desc
				}
				ReplyToSender(ctx, msg, info)
			}
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

	cmdMap["text"] = cmdText
	cmdMap["help"] = cmdHelp
	cmdMap["bot"] = cmdBot
	cmdMap["dismiss"] = cmdDismiss
	cmdMap["botlist"] = cmdBotList
	cmdMap["master"] = cmdMaster
	cmdMap["nn"] = cmdNN
	cmdMap["userid"] = cmdUserID
	cmdMap["角色"] = cmdChar
	cmdMap["ch"] = cmdChar
	cmdMap["char"] = cmdChar
	cmdMap["character"] = cmdChar
	cmdMap["pc"] = cmdChar
	cmdMap["set"] = cmdSet
	cmdMap["ext"] = cmdExt

	theExt.CmdMap = cmdMap

	dice.RegisterExtension(theExt)
}
