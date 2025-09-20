package types

import (
	"smallseal/dice/attrs"
	"smallseal/utils"

	"github.com/samber/lo"
	ds "github.com/sealdice/dicescript"
)

type MsgContext struct {
	CommandId int64
	AdapterId string

	IsCurGroupBotOn bool
	IsPrivate       bool

	CommandHideFlag string // 这个是干啥的，已经忘了

	MessageScene string // 聊天场景类型: group(群聊) private(私聊) channel(频道) guild(服务器) 等

	Group  *GroupInfo
	Player *GroupPlayerInfo

	AttrsManager *attrs.AttrsManager
	GameSystem   *GameSystemTemplateV2

	TextTemplateMap TextTemplateWithWeightDict

	CommandInfo  map[string]any
	DelegateText string

	vm   *ds.Context
	Dice DiceLike
}

func (ctx *MsgContext) GetVM() *ds.Context {
	if ctx.vm == nil {
		ctx.vm = newVM(ctx.Group.GroupId, ctx.Player.UserId, ctx.AttrsManager, ctx.GameSystem, ctx.TextTemplateMap)
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

func (ctx *MsgContext) EvalFString(expr string, flags *ds.RollConfig) (*ds.VMValue, string, error) {
	return ctx.EvalBase("\x1e"+expr+"\x1e", flags)
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

func (ctx *MsgContext) Copy() *MsgContext {
	c1 := *ctx
	c1.vm = nil
	return &c1
}

func (ctx *MsgContext) GetCharTemplate() *GameSystemTemplateV2 {
	group := ctx.Group
	// 有system优先system
	if group.System != "" {
		v, _ := ctx.Dice.GameSystemMapLoad(group.System)
		if v != nil {
			return v
		}
		// 返回这个单纯是为了不让st将其覆盖
		// 这种情况属于卡片的规则模板被删除了
		return &GameSystemTemplateV2{
			Name:     group.System,
			FullName: "空白模板",
			AliasMap: new(utils.SyncMap[string, string]),
		}
	}

	// 没有system，查看扩展的启动情况
	// TODO: 我觉得根据扩展选模板可以有一个通用机制，不要写死
	// if group.ExtGetActive("coc7") != nil {
	// 	v, _ := ctx.Dice.GameSystemMapLoad.Load("coc7")
	// 	return v
	// }

	// if group.ExtGetActive("dnd5e") != nil {
	// 	v, _ := ctx.Dice.GameSystemMapLoad.Load("dnd5e")
	// 	return v
	// }

	// 啥都没有，返回个空白模板
	blankTmpl := &GameSystemTemplateV2{
		Name:     "空白模板",
		FullName: "空白模板",
		AliasMap: new(utils.SyncMap[string, string]),
	}
	return blankTmpl
}

func (ctx *MsgContext) GetDefaultDicePoints() string {
	var diceExpr string
	if ctx.Player != nil {
		diceExpr = ctx.Player.DiceSideExpr
	}
	if diceExpr == "" && ctx.Group != nil {
		diceExpr = ctx.Group.DiceSideExpr
	}
	if diceExpr == "" {
		diceExpr = "d100"
	}
	return diceExpr
}
