package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/peterh/liner"

	"github.com/sealdice/smallseal/dice"
	"github.com/sealdice/smallseal/dice/types"
)

var (
	historyFn = filepath.Join(os.TempDir(), ".dicescript_textmap_demo_history")
	rng       = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func main() {
	d := dice.NewDice()

	var ctxStore sync.Map

	// 输入重写 Hook - 每次随机选择一种风格
	if _, err := d.RegisterMessageInHook("examples/cli-textmap", types.HookPriorityNormal, func(_ types.DiceLike, _ string, _msg *types.Message, mctx *types.MsgContext) types.HookResult {
		// 随机选择一种输入重写风格
		selectedStyle := inputRewriteStyles[rng.Intn(len(inputRewriteStyles))]
		mctx.TextTemplateMap = selectedStyle
		mctx.FallbackTextTemplate = dice.DefaultTextMap
		ctxStore.Store(mctx, time.Now())
		return types.HookResultContinue
	}); err != nil {
		panic(err)
	}

	// 输出重写 Hook - 用所有风格重写一遍
	if _, err := d.RegisterMessageOutHook("examples/cli-textmap", types.HookPriorityNormal, func(_ types.DiceLike, _ string, reply *types.MsgToReply) types.HookResult {
		cleanupStaleContexts(&ctxStore, 5*time.Second)
		if reply == nil || reply.CommandId == 0 || len(reply.CommandFormatInfo) == 0 {
			return types.HookResultContinue
		}

		mctx := findContextByCommandID(&ctxStore, reply.CommandId)
		if mctx == nil {
			return types.HookResultContinue
		}

		original := reply.Segments.ToText()
		var builder strings.Builder
		builder.WriteString("[原始输出]\n")
		builder.WriteString(original)
		builder.WriteString("\n")

		// 用所有输出重写风格依次重写
		for _, style := range outputRewriteStyles {
			reformatted := rebuildWithTextMap(mctx, reply, style.TextMap)
			if strings.TrimSpace(reformatted) == "" {
				continue
			}
			builder.WriteString("\n")
			builder.WriteString(style.Name)
			builder.WriteString("\n")
			builder.WriteString(reformatted)
			builder.WriteString("\n")
		}

		builder.WriteString("================")
		reply.Segments = types.MessageSegments{
			&types.TextElement{Content: builder.String()},
		}

		return types.HookResultContinue
	}); err != nil {
		panic(err)
	}

	d.CallbackForSendMsg.Store("console", func(msg *types.MsgToReply) {
		fmt.Printf("%s\n", msg.Segments.ToText())
	})

	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)

	if f, err := os.Open(historyFn); err == nil {
		_, _ = line.ReadHistory(f)
		_ = f.Close()
	}

	fmt.Println("CLI TextMap Persona DEMO")
	fmt.Println("这个 Demo 会演示了两种方式对骰子的输出进行重写。第一种是在输入时替换原始模板，第二种是在输出时根据骰点信息对结果进行重构")
	fmt.Println("示例模板台词由AI生成")

	for {
		text, err := line.Prompt(">>> ")
		if err != nil {
			break
		}

		if strings.TrimSpace(text) == "" {
			continue
		}
		line.AppendHistory(text)

		d.Execute("", &types.Message{
			MessageType: "group",
			GroupID:     "cli-demo-group",
			Sender: types.SenderBase{
				UserID:   "cli-user",
				Nickname: "CLI",
			},
			Segments: types.MessageSegments{
				&types.TextElement{Content: text},
			},
		})
	}

	if f, err := os.Create(historyFn); err == nil {
		_, _ = line.WriteHistory(f)
		_ = f.Close()
	}
}
