package dice

import (
	"fmt"
	"sort"
	"sync/atomic"
	"time"

	"github.com/sealdice/smallseal/dice/attrs"
	"github.com/sealdice/smallseal/dice/exts"
	"github.com/sealdice/smallseal/dice/types"
	"github.com/sealdice/smallseal/utils"
)

type Dice struct {
	GroupInfoManager GroupInfoManager

	attrsManager *attrs.AttrsManager
	gameSystem   utils.SyncMap[string, *types.GameSystemTemplateV2]

	CallbackForSendMsg utils.SyncMap[string, func(msg *types.MsgToReply)]

	inboundHooks  hookRegistry[types.MessageInHook]
	outboundHooks hookRegistry[types.MessageOutHook]
	eventHooks    hookRegistry[types.EventHook]

	ExtList []*types.ExtInfo

	curCommandID atomic.Int64

	Config struct {
		CommandPrefix []string
	}
}

func NewDice() *Dice {
	d := &Dice{
		attrsManager:     &attrs.AttrsManager{},
		GroupInfoManager: NewDefaultGroupInfoManager(),

		CallbackForSendMsg: utils.SyncMap[string, func(msg *types.MsgToReply)]{},
	}

	d.attrsManager.Init()
	d.Config.CommandPrefix = []string{"."}

	coc7, err := types.LoadGameSystemTemplate("./coc7.yaml")
	if err != nil {
		panic(err)
	}
	d.gameSystem.Store(coc7.Name, coc7)

	exts.RegisterBuiltinExtCore(d)
	exts.RegisterBuiltinExtCoc7(d)
	exts.RegisterBuiltinExtDnd5e(d)

	return d
}

func (d *Dice) getNextCommandID() int64 {
	return d.curCommandID.Add(1)
}

func (d *Dice) Execute(adapterId string, msg *types.Message) {
	if msg == nil {
		return
	}

	mctx := &types.MsgContext{Dice: d, AdapterId: adapterId, TextTemplateMap: DefaultTextMap}
	mctx.AttrsManager = d.attrsManager

	msg.Message = msg.Segments.ToText()
	if msg.MessageType != "group" && msg.MessageType != "private" {
		return
	}

	groupInfo, ok := d.GroupInfoManager.Load(msg.GroupID)

	if !ok {
		groupInfo = &types.GroupInfo{
			GroupId: msg.GroupID,
			System:  "coc7",
		}

		groupInfo.ExtActiveStates = &utils.SyncMap[string, bool]{}
		for _, ext := range d.GetExtList() {
			if ext.AutoActive {
				groupInfo.SetExtensionActive(ext.Name, true)
			}
		}
		groupInfo.ActivatedExtList = groupInfo.GetActiveExtensions(d.GetExtList())
		d.GroupInfoManager.Store(msg.GroupID, groupInfo)
	}

	mctx.GameSystem, _ = d.gameSystem.Load(groupInfo.System)
	mctx.Group = groupInfo
	mctx.Player = &types.GroupPlayerInfo{
		UserId: msg.Sender.UserID,
		Name:   msg.Sender.Nickname,
	}

	groupInfo.UpdatedAtTime = time.Now().Unix()

	activeExtensions := groupInfo.GetActiveExtensions(d.GetExtList())
	groupInfo.ActivatedExtList = activeExtensions

	if d.runMessageInHooks(adapterId, msg, mctx) {
		return
	}

	cmdLst := []string{}
	for _, ext := range activeExtensions {
		for cmd := range ext.CmdMap {
			cmdLst = append(cmdLst, cmd)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(cmdLst)))

	platformPrefix := "QQ"
	cmdArgs := types.CommandParse(msg.Message, cmdLst, d.Config.CommandPrefix, platformPrefix, false)

	if cmdArgs != nil {
		mctx.CommandId = d.getNextCommandID()
	}

	for _, _i := range activeExtensions {
		i := _i
		if !mctx.Group.IsExtensionActive(i.Name) {
			continue
		}

		if i.OnNotCommandReceived != nil {
			i.OnNotCommandReceived(mctx, msg)
		}
		if i.OnCommandReceived != nil && cmdArgs != nil {
			i.OnCommandReceived(mctx, msg, cmdArgs)
		}

		if cmdArgs != nil {
			if cmd, ok := i.CmdMap[cmdArgs.Command]; ok {
				cmd.Solve(mctx, msg, cmdArgs)
			}
		}
	}
}

func (d *Dice) SendReply(msg *types.MsgToReply) {
	if msg == nil {
		return
	}

	if d.runMessageOutHooks(msg.AdapterId, msg) {
		return
	}

	d.CallbackForSendMsg.Range(func(key string, value func(msg *types.MsgToReply)) bool {
		value(msg)
		return true
	})
}

func (d *Dice) AttrsSetIO(io attrs.AttrsIO) {
	d.attrsManager.SetIO(io)
}

func (d *Dice) GameSystemMapLoad(name string) (*types.GameSystemTemplateV2, error) {
	if gameSystem, ok := d.gameSystem.Load(name); ok {
		return gameSystem, nil
	}
	return nil, fmt.Errorf("game system not found: %s", name)
}

func (d *Dice) RegisterMessageInHook(name string, priority types.HookPriority, hook types.MessageInHook) (types.HookHandle, error) {
	return d.inboundHooks.register(name, priority, hook)
}

func (d *Dice) UnregisterMessageInHook(handle types.HookHandle) bool {
	return d.inboundHooks.unregister(handle)
}

func (d *Dice) RegisterMessageOutHook(name string, priority types.HookPriority, hook types.MessageOutHook) (types.HookHandle, error) {
	return d.outboundHooks.register(name, priority, hook)
}

func (d *Dice) UnregisterMessageOutHook(handle types.HookHandle) bool {
	return d.outboundHooks.unregister(handle)
}

func (d *Dice) RegisterEventHook(name string, priority types.HookPriority, hook types.EventHook) (types.HookHandle, error) {
	return d.eventHooks.register(name, priority, hook)
}

func (d *Dice) UnregisterEventHook(handle types.HookHandle) bool {
	return d.eventHooks.unregister(handle)
}

func (d *Dice) DispatchEvent(adapterID string, evt *types.AdapterEvent) {
	if evt == nil {
		return
	}

	d.runEventHooks(adapterID, evt)
}

func (d *Dice) runMessageInHooks(adapterID string, msg *types.Message, mctx *types.MsgContext) bool {
	entries := d.inboundHooks.snapshot()
	if len(entries) == 0 {
		return false
	}

	for _, entry := range entries {
		switch entry.handler(d, adapterID, msg, mctx) {
		case types.HookResultContinue:
			continue
		case types.HookResultStop:
			return false
		case types.HookResultAbort:
			return true
		default:
			continue
		}
	}

	return false
}

func (d *Dice) runMessageOutHooks(adapterID string, reply *types.MsgToReply) bool {
	entries := d.outboundHooks.snapshot()
	if len(entries) == 0 {
		return false
	}

	for _, entry := range entries {
		switch entry.handler(d, adapterID, reply) {
		case types.HookResultContinue:
			continue
		case types.HookResultStop:
			return false
		case types.HookResultAbort:
			return true
		default:
			continue
		}
	}

	return false
}

func (d *Dice) runEventHooks(adapterID string, evt *types.AdapterEvent) {
	entries := d.eventHooks.snapshot()
	if len(entries) == 0 {
		return
	}

	for _, entry := range entries {
		switch entry.handler(d, adapterID, evt) {
		case types.HookResultContinue:
			continue
		case types.HookResultStop:
			return
		case types.HookResultAbort:
			return
		default:
			continue
		}
	}
}
