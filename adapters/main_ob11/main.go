package main

// 一个用于调试 OneBot11 适配器的简单入口

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/sealdice/smallseal/adapters"
	"github.com/sealdice/smallseal/dice"
	"github.com/sealdice/smallseal/dice/types"

	"go.uber.org/zap"
)

type ob11Callback struct {
	dice *dice.Dice
}

func (cb *ob11Callback) OnError(err error) {
	fmt.Printf("OnError: %v\n", err)
	debug.PrintStack()
}

func (cb *ob11Callback) OnMessageReceived(info *adapters.MessageSendCallbackInfo) {
	jsonInfo, err := json.Marshal(info)
	if err != nil {
		fmt.Printf("OnMessageReceived: marshal err, info=%v, err=%v\n", info, err)
		return
	}
	fmt.Printf("OnMessageReceived: %s, msg=%s\n", string(jsonInfo), info.Message.Segments.ToText())

	if cb.dice != nil && info.Message != nil {
		cb.dice.Execute("", info.Message)
	}
}

func (cb *ob11Callback) OnEvent(evt *types.AdapterEvent) {
	if evt == nil {
		return
	}
	if evt.Type == "heartbeat" {
		return
	}
	jsonEvt, err := json.Marshal(evt)
	if err != nil {
		fmt.Printf("OnEvent: marshal err, evt=%v, err=%v\n", evt, err)
		return
	}
	fmt.Printf("OnEvent: %s\n", string(jsonEvt))
	if cb.dice != nil {
		cb.dice.DispatchEvent("", evt)
	}
}

func newOB11ConnItem() *adapters.PlatformAdapterOB11 {
	reverse := os.Getenv("OB11_WS_REVERSE")
	forward := os.Getenv("OB11_WS_FORWARD")
	if reverse == "" && forward == "" {
		reverse = "ws://127.0.0.1:8100/onebot/v11/ws"
	}

	return &adapters.PlatformAdapterOB11{
		WSReverseURL:  reverse,
		WSForwardAddr: forward,
		AccessToken:   os.Getenv("OB11_ACCESS_TOKEN"),
		Secret:        os.Getenv("OB11_SECRET"),
	}
}

func main() {
	fmt.Println("Small Seal (OB11) v0.0.1")
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	zap.ReplaceGlobals(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	d := dice.NewDice()
	conn := newOB11ConnItem()

	callback := &ob11Callback{dice: d}
	conn.SetCallback(callback)

	go conn.Serve(ctx)

	d.CallbackForSendMsg.Store("ob11", func(msg *types.MsgToReply) {
		if msg == nil {
			return
		}

		if msg.MessageType == "private" {
			_, _ = conn.MsgSendToPerson(&adapters.MessageSendRequest{
				TargetId: msg.SendTo.UserId,
				Segments: msg.Segments,
			})
		}

		if msg.MessageType == "group" {
			_, _ = conn.MsgSendToGroup(&adapters.MessageSendRequest{
				TargetId: msg.SendTo.GroupId,
				Segments: msg.Segments,
			})
		}
	})

	fmt.Println("等待消息中... 使用 Ctrl+C 退出")

	<-ctx.Done()
	conn.Close()
}
