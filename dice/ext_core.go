package dice

import (
	"fmt"
	"regexp"

	"smallseal/dice/types"
)

func RegisterBuiltinExtCore(dice *Dice) {
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

			if cmdArgs.IsArgEqual(1, "help") {
				return types.CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			// ctx.SystemTemplate = ctx.Group.GetCharTemplate(ctx.Dice)
			if (cmdArgs.Command == "rd" || cmdArgs.Command == "rhd" || cmdArgs.Command == "rdh") && len(cmdArgs.Args) >= 1 {
				if m, _ := regexp.MatchString(`^\d|优势|劣势|\+|-`, cmdArgs.CleanArgs); m {
					if cmdArgs.IsSpaceBeforeArgs {
						cmdArgs.CleanArgs = "d " + cmdArgs.CleanArgs
					} else {
						cmdArgs.CleanArgs = "d" + cmdArgs.CleanArgs
					}
				}
			}

			tmpl, _ := dice.gameSystem.Load(ctx.Group.System)
			vm := newVM(ctx, dice.AttrsManager, tmpl)

			expr := cmdArgs.CleanArgs

			if expr == "" {
				expr = "d"
			}

			err := vm.Run(expr)

			if err == nil {
				var forWhat string
				matched := vm.Matched
				if forWhat == "" {
					forWhat = vm.RestInput
				}

				if forWhat != "" {
					text = fmt.Sprintf("由于%s, %s = %s = %s", forWhat, matched, vm.GetDetailText(), vm.Ret.ToString())
				} else {
					text = fmt.Sprintf("%s = %s = %s", matched, vm.GetDetailText(), vm.Ret.ToString())
				}
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

	cmdMap["r"] = cmdRoll
	cmdMap["rd"] = cmdRoll
	cmdMap["roll"] = cmdRoll
	cmdMap["rh"] = cmdRoll
	cmdMap["rhd"] = cmdRoll
	cmdMap["rdh"] = cmdRoll
	cmdMap["rx"] = cmdRollX
	cmdMap["rxh"] = cmdRollX
	cmdMap["rhx"] = cmdRollX

	theExt.CmdMap = cmdMap

	dice.RegisterExtension(theExt)
}
