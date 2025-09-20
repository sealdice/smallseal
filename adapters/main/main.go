package main

// 一个测试文件，配合 Lagrange.Milky 使用

import (
	"encoding/json"
	"fmt"
	"runtime/debug"

	"smallseal/adapters"
	"smallseal/dice"
	"smallseal/dice/types"
)

type AdapterCallbackBase2 struct {
	dice *dice.Dice
}

func (cb *AdapterCallbackBase2) OnError(err error) {
	fmt.Printf("OnError: %v\n", err)
	debug.PrintStack()
}

func (cb *AdapterCallbackBase2) OnMessageReceived(info *adapters.MessageSendCallbackInfo) {
	jsonInfo, err := json.Marshal(info)
	if err != nil {
		fmt.Printf("OnMessageReceived: marshal err, info=%v, err=%v\n", info, err)
		return
	}
	fmt.Printf("OnMessageReceived: %v, msg=%s\n", string(jsonInfo), info.Message.Segments.ToText())
	// 将接收到的消息转发给dice对象处理
	if cb.dice != nil && info.Message != nil {
		cb.dice.Execute("", info.Message)
	}
}

func NewMilkyConnItem() *adapters.PlatformAdapterMilky {
	adapter := &adapters.PlatformAdapterMilky{
		Token:       "",
		WsGateway:   "ws://127.0.0.1:3100/event",
		RestGateway: "http://127.0.0.1:3100",
	}
	return adapter
}

func main() {
	fmt.Println("Small Seal v0.0.1")

	d := dice.NewDice()
	conn := NewMilkyConnItem()

	fmt.Println("conn.Serve() =", conn.Serve())

	// 创建回调实例并传入dice对象
	callback := &AdapterCallbackBase2{
		dice: d,
	}
	conn.SetCallback(callback)

	d.CallbackForSendMsg.Store("milky", func(msg *types.MsgToReply) {
		if conn != nil {
			fmt.Println("callback send msg, msg=", msg.Segments.ToText())
			jsonInfo, err := json.Marshal(msg)
			if err != nil {
				fmt.Printf("callback send msg, marshal err, msg=%v, err=%v\n", msg, err)
				return
			}
			fmt.Printf("callback send msg, json=%s\n", string(jsonInfo))

			if msg.MessageType == "private" {
				conn.MsgSendToPerson(&adapters.MessageSendRequest{
					TargetId: msg.SendTo.UserId,
					Segments: msg.Segments,
				})
			}

			if msg.MessageType == "group" {
				conn.MsgSendToGroup(&adapters.MessageSendRequest{
					TargetId: msg.SendTo.GroupId,
					Segments: msg.Segments,
				})
			}
		}
	})

	fmt.Println("等待消息中...")

	// 保持程序运行，等待消息到来
	select {}
}
