package types

import (
	"smallseal/dice/attrs"

	"github.com/samber/lo"
	ds "github.com/sealdice/dicescript"
)

type MsgContext struct {
	CommandID int64

	IsCurGroupBotOn bool
	IsPrivate       bool

	CommandHideFlag string // 这个是干啥的，已经忘了

	MessageScene string // 聊天场景类型: group(群聊) private(私聊) channel(频道) guild(服务器) 等

	Group  *GroupInfo
	Player *GroupPlayerInfo

	AttrsManager *attrs.AttrsManager
	GameSystem   *GameSystemTemplateV2

	vm   *ds.Context
	Dice DiceLike
}

func (ctx *MsgContext) GetVM() *ds.Context {
	if ctx.vm == nil {
		ctx.vm = newVM(ctx.Group.GroupId, ctx.Player.UserId, ctx.AttrsManager, ctx.GameSystem)
	}
	return ctx.vm
}

// 当前群的群内默认卡
func (ctx *MsgContext) LoadAttrsForCurGroupUser() *attrs.AttributesItem {
	if ctx.Group == nil || ctx.Player == nil {
		return nil
	}
	return lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))
}

func (ctx *MsgContext) LoadAttrsForCurUser() *attrs.AttributesItem {
	if ctx.Player == nil {
		return nil
	}
	return lo.Must(ctx.AttrsManager.LoadById(ctx.Player.UserId))
}

func (ctx *MsgContext) LoadAttrsForCurGroup() *attrs.AttributesItem {
	if ctx.Group == nil {
		return nil
	}
	return lo.Must(ctx.AttrsManager.LoadById(ctx.Group.GroupId))
}

func (ctx *MsgContext) EvalBase(expr string, flags *ds.RollConfig) (*ds.VMValue, string, error) {
	vm := ctx.GetVM()
	prevConfig := vm.Config
	if flags != nil {
		vm.Config = *flags
	}
	err := vm.Run(expr)
	vm.Config = prevConfig

	if err != nil {
		return nil, "", err
	}
	return vm.Ret, vm.GetDetailText(), nil
}

func (ctx *MsgContext) Eval(expr string, flags *ds.RollConfig) *ds.VMValue {
	vm := ctx.GetVM()
	prevConfig := vm.Config
	if flags != nil {
		vm.Config = *flags
	}
	err := vm.Run(expr)
	vm.Config = prevConfig

	if err != nil {
		return nil
	}
	return vm.Ret
}
