package dice

import (
	"fmt"
	"sort"
	"sync/atomic"
	"time"

	"smallseal/dice/attrs"
	"smallseal/dice/exts"
	"smallseal/dice/types"
	"smallseal/utils"
)

type Dice struct {
	GroupInfoManager GroupInfoManager

	attrsManager *attrs.AttrsManager
	gameSystem   utils.SyncMap[string, *types.GameSystemTemplateV2]

	CallbackForSendMsg utils.SyncMap[string, func(msg *types.MsgToReply)]

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
	mctx := &types.MsgContext{Dice: d, AdapterId: adapterId, TextTemplateMap: DefaultTextMap}
	mctx.AttrsManager = d.attrsManager

	// mctx.MessageType = msg.MessageType
	// mctx.IsPrivate = mctx.MessageType == "private"

	msg.Message = msg.Segments.ToText()
	if msg.MessageType != "group" && msg.MessageType != "private" {
		return
	}

	// 处理命令
	groupInfo, ok := d.GroupInfoManager.Load(msg.GroupID)

	if !ok {
		groupInfo = &types.GroupInfo{
			GroupId: msg.GroupID,
			System:  "coc7",
		}

		// 初始化扩展激活状态映射表
		groupInfo.ExtActiveStates = &utils.SyncMap[string, bool]{}
		// 根据扩展的AutoActive属性设置初始激活状态
		for _, ext := range d.GetExtList() {
			if ext.AutoActive {
				groupInfo.SetExtensionActive(ext.Name, true)
			}
		}
		// 使用新的方法获取激活的扩展列表
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

	// 遍历激活的扩展，获取指令列表
	cmdLst := []string{}
	activeExtensions := groupInfo.GetActiveExtensions(d.GetExtList())
	for _, ext := range activeExtensions {
		for cmd := range ext.CmdMap {
			cmdLst = append(cmdLst, cmd)
		}
	}
	// 更新ActivatedExtList以保持兼容性
	groupInfo.ActivatedExtList = activeExtensions
	sort.Sort(sort.Reverse(sort.StringSlice(cmdLst)))

	platformPrefix := "QQ"
	cmdArgs := types.CommandParse(msg.Message, cmdLst, d.Config.CommandPrefix, platformPrefix, false)

	if cmdArgs != nil {
		mctx.CommandId = d.getNextCommandID()
	}

	// 收到群 test(1111) 内 XX(222) 的消息: 好看 (1232611291)
	// if msg.MessageType == "group" {
	// 	// TODO(Szzrain):  需要优化的写法，不应根据 CommandID 来判断是否是指令，而应该根据 cmdArgs 是否 match 到指令来判断
	// 	if mctx.CommandID != 0 {
	// 		// 关闭状态下，如果被@，且是第一个被@的，那么视为开启
	// 		if !mctx.IsCurGroupBotOn && cmdArgs.AmIBeMentionedFirst {
	// 			mctx.IsCurGroupBotOn = true
	// 		}

	// 		log.Infof("收到群(%s)内<%s>(%s)的指令: %s", msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
	// 	} else {
	// 		doLog := true
	// 		if d.Config.OnlyLogCommandInGroup {
	// 			// 检查上级选项
	// 			doLog = false
	// 		}
	// 		if doLog {
	// 		}
	// 		if doLog {
	// 			log.Infof("收到群(%s)内<%s>(%s)的消息: %s", msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
	// 			// fmt.Printf("消息长度 %v 内容 %v \n", len(msg.Message), []byte(msg.Message))
	// 		}
	// 	}
	// }

	// // Note(Szzrain): 这里的代码本来在敏感词检测下面，会产生预期之外的行为，所以挪到这里
	// if msg.MessageType == "private" {
	// 	// TODO(Szzrain): 需要优化的写法，不应根据 CommandID 来判断是否是指令，而应该根据 cmdArgs 是否 match 到指令来判断，同上
	// 	if mctx.CommandID != 0 {
	// 		log.Infof("收到<%s>(%s)的私聊指令: %s", msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
	// 	} else if !d.Config.OnlyLogCommandInPrivate {
	// 		log.Infof("收到<%s>(%s)的私聊消息: %s", msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
	// 	}
	// }

	// Note(Szzrain): 赋值临时变量，不然有些地方没法用
	// SetTempVars(mctx, msg.Sender.Nickname)

	// 只遍历激活的扩展进行指令处理
	for _, _i := range activeExtensions {
		i := _i // 保留引用(我记得从go的什么版本开始不用如此处理了)
		// 检查扩展是否真正激活
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
	// TODO: 整几个 delegate 在这等着，先跑钩子，再跑delegate，adapter们都去注册delegate
	// text := msg.Segment.ToText()
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

// GetExtList 获取扩展列表
func (d *Dice) GetExtList() []*types.ExtInfo {
	return d.ExtList
}
