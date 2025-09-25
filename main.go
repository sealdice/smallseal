package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/peterh/liner"

	"github.com/sealdice/smallseal/dice"
	"github.com/sealdice/smallseal/dice/types"
)

var (
	historyFn = filepath.Join(os.TempDir(), ".dicescript_history")
)

func main() {
	d := dice.NewDice()

	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)
	line.SetCompleter(func(line string) (c []string) {
		return
	})

	if f, err := os.Open(historyFn); err == nil {
		_, _ = line.ReadHistory(f)
		_ = f.Close()
	}

	fmt.Println("DiceScript Shell v0.0.1")
	ccTimes := 0

	d.CallbackForSendMsg.Store("console", func(msg *types.MsgToReply) {
		fmt.Printf("%s\n", msg.Segments.ToText())
	})

	for {
		if text, err := line.Prompt(">>> "); err == nil {
			if strings.TrimSpace(text) == "" {
				continue
			}
			line.AppendHistory(text)

			d.Execute("", &types.Message{
				MessageType: "group",
				GroupID:     "1000",

				Sender: types.SenderBase{
					UserID:   "123",
					Nickname: "seal",
				},
				Segments: []types.IMessageElement{
					&types.TextElement{
						Content: text,
					},
				},
			})

			// err := vm.Run(text)
			// // fmt.Println(vm.GetAsmText())
			// if err != nil {
			// 	fmt.Printf("错误: %s\n", err.Error())
			// } else {
			// 	rest := vm.RestInput
			// 	if rest != "" {
			// 		rest = fmt.Sprintf(" 剩余文本: %s", rest)
			// 	}
			// 	fmt.Printf("过程: %s\n", vm.GetDetailText())
			// 	fmt.Printf("结果: %s%s\n", vm.Ret.ToString(), rest)
			// 	fmt.Printf("栈顶: %d 层数:%d 算力: %d\n", vm.StackTop(), vm.Depth(), vm.NumOpCount)
			// }

		} else if err == liner.ErrPromptAborted {
			if ccTimes >= 0 {
				fmt.Print("Interrupted")
				break
			} else {
				ccTimes += 1
				fmt.Println("Input Ctrl-c once more to exit")
			}
		} else {
			fmt.Print("Error reading line: ", err)
		}
	}

	if f, err := os.Create(historyFn); err != nil {
		fmt.Println("Error writing history file: ", err)
	} else {
		_, _ = line.WriteHistory(f)
		_ = f.Close()
	}

}
