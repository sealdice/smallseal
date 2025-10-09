package dice

import (
	"fmt"
	"sort"
	"strings"
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

	masterList utils.SyncMap[string, bool]
}

func NewDice() *Dice {
	d := &Dice{
		attrsManager:     &attrs.AttrsManager{},
		GroupInfoManager: NewDefaultGroupInfoManager(),

		CallbackForSendMsg: utils.SyncMap[string, func(msg *types.MsgToReply)]{},

		masterList: utils.SyncMap[string, bool]{},
	}

	d.attrsManager.Init()
	d.Config.CommandPrefix = []string{".", "。"}

	for _, asset := range exts.BuiltinGameSystemTemplateAssets() {
		gs, err := types.LoadGameSystemTemplateFromData(asset.Data, asset.Filename)
		if err != nil {
			panic(err)
		}
		d.gameSystem.Store(gs.Name, gs)
	}

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

	mctx := &types.MsgContext{Dice: d, AdapterId: adapterId, TextTemplateMap: DefaultTextMap, FallbackTextTemplate: DefaultTextMap}
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
			Active:  true,
		}

		groupInfo.ExtActiveStates = &utils.SyncMap[string, bool]{}
		groupInfo.BotList = &utils.SyncMap[string, bool]{}
		groupInfo.Players = &utils.SyncMap[string, *types.GroupPlayerInfo]{}
		for _, ext := range d.GetExtList() {
			if ext.AutoActive {
				groupInfo.SetExtensionActive(ext.Name, true)
			}
		}
		groupInfo.ActivatedExtList = groupInfo.GetActiveExtensions(d.GetExtList())
		d.GroupInfoManager.Store(msg.GroupID, groupInfo)
	}

	if groupInfo.BotList == nil {
		groupInfo.BotList = &utils.SyncMap[string, bool]{}
	}
	if groupInfo.Players == nil {
		groupInfo.Players = &utils.SyncMap[string, *types.GroupPlayerInfo]{}
	}

	mctx.GameSystem, _ = d.gameSystem.Load(groupInfo.System)
	mctx.Group = groupInfo

	mctx.IsCurGroupBotOn = groupInfo.Active

	player, exists := groupInfo.Players.Load(msg.Sender.UserID)
	if !exists {
		player = &types.GroupPlayerInfo{
			UserId: msg.Sender.UserID,
			Name:   msg.Sender.Nickname,
		}
		groupInfo.Players.Store(msg.Sender.UserID, player)
	} else if msg.Sender.Nickname != "" {
		if player.Name == "" {
			player.Name = msg.Sender.Nickname
		}
	}

	player.InGroup = msg.MessageType == "group"
	mctx.Player = player

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

	sendHelp := func(cmd *types.CmdItemInfo) {
		helpText := ""
		if cmd.HelpFunc != nil {
			helpText = cmd.HelpFunc(false)
		}
		if helpText == "" {
			helpText = cmd.Help
		}
		if helpText == "" {
			short := strings.TrimSpace(cmd.ShortHelp)
			if short != "" {
				helpText = fmt.Sprintf("%s:\n%s", cmd.Name, short)
			} else {
				helpText = fmt.Sprintf("%s 指令暂无帮助信息", cmd.Name)
			}
		}
		exts.ReplyToSender(mctx, msg, helpText)
	}

	solved := false
	allowWhenInactive := func(cmd *types.CmdItemInfo) bool {
		if cmd == nil {
			return false
		}
		switch strings.ToLower(cmd.Name) {
		case "bot", "dismiss":
			return true
		default:
			return false
		}
	}

	groupActive := mctx.Group == nil || mctx.Group.Active

	for _, _i := range activeExtensions {
		i := _i
		if mctx.Group != nil && !mctx.Group.IsExtensionActive(i.Name) {
			continue
		}

		if groupActive && i.OnNotCommandReceived != nil {
			i.OnNotCommandReceived(mctx, msg)
		}
		if groupActive && i.OnCommandReceived != nil && cmdArgs != nil {
			i.OnCommandReceived(mctx, msg, cmdArgs)
		}

		if cmdArgs != nil && !solved {
			if cmd, ok := i.CmdMap[cmdArgs.Command]; ok {
				if !groupActive && !allowWhenInactive(cmd) {
					continue
				}
				result := cmd.Solve(mctx, msg, cmdArgs)
				if result.Solved {
					solved = true
					if result.ShowHelp {
						sendHelp(cmd)
					}
				}
			}
		}
	}
}

func (d *Dice) MasterAdd(uid string) {
	if uid == "" {
		return
	}
	d.masterList.Store(uid, true)
}

func (d *Dice) MasterRemove(uid string) bool {
	if uid == "" {
		return false
	}
	return d.masterList.Delete(uid)
}

func (d *Dice) ListMasters() []string {
	masters := make([]string, 0)
	d.masterList.Range(func(key string, _ bool) bool {
		masters = append(masters, key)
		return true
	})
	sort.Strings(masters)
	return masters
}

func (d *Dice) IsMaster(uid string) bool {
	if uid == "" {
		return false
	}
	_, exists := d.masterList.Load(uid)
	return exists
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

func (d *Dice) PersistGroupInfo(groupID string, info *types.GroupInfo) {
	if d == nil || d.GroupInfoManager == nil {
		return
	}
	if groupID == "" || info == nil {
		return
	}
	d.GroupInfoManager.Store(groupID, info)
}

func (d *Dice) AttrsSetIO(io attrs.AttrsIO) {
	d.attrsManager.SetIO(io)
}

// SaveAll 手动保存所有未保存的属性数据
func (d *Dice) SaveAll() error {
	if d.attrsManager == nil {
		return fmt.Errorf("属性管理器未初始化")
	}
	return d.attrsManager.CheckForSave()
}

func (d *Dice) GameSystemMapLoad(name string) (*types.GameSystemTemplateV2, error) {
	if gameSystem, ok := d.gameSystem.Load(name); ok {
		return gameSystem, nil
	}
	return nil, fmt.Errorf("game system not found: %s", name)
}

func (d *Dice) GameSystemMapGet() *utils.SyncMap[string, *types.GameSystemTemplateV2] {
	return &d.gameSystem
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
