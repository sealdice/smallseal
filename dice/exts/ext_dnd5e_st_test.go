package exts

import (
	"strings"
	"testing"
	"time"

	ds "github.com/sealdice/dicescript"
	"github.com/stretchr/testify/require"

	"github.com/sealdice/smallseal/dice/attrs"
	"github.com/sealdice/smallseal/dice/types"
	"github.com/sealdice/smallseal/utils"
)

func newDnd5eTemplate(t *testing.T) *types.GameSystemTemplateV2 {
	t.Helper()

	asset, ok := BuiltinGameSystemTemplateAsset("dnd5e.yaml")
	require.True(t, ok)

	tmpl, err := types.LoadGameSystemTemplateFromData(asset.Data, asset.Filename)
	require.NoError(t, err)
	return tmpl
}

func newDnd5eTestContext(t *testing.T) (*types.MsgContext, *types.Message, *stubDice, *types.CmdItemInfo) {
	t.Helper()

	tmpl := newDnd5eTemplate(t)
	stub := newStubDice(tmpl)

	RegisterBuiltinExtCore(stub)
	RegisterBuiltinExtDnd5e(stub)

	ext, ok := stub.extensions["dnd5e"]
	require.True(t, ok, "dnd5e extension should be registered")
	cmd, ok := ext.CmdMap["st"]
	require.True(t, ok, "dnd5e extension should provide st command")

	am := &attrs.AttrsManager{}
	am.SetIO(attrs.NewMemoryAttrsIO())

	group := &types.GroupInfo{
		GroupId: "group-1",
		System:  "dnd5e",
		BotList: &utils.SyncMap[string, bool]{},
	}

	player := &types.GroupPlayerInfo{
		UserId: "user-1",
		Name:   "冒险者",
	}

	ctx := &types.MsgContext{
		Dice:            stub,
		AttrsManager:    am,
		Group:           group,
		Player:          player,
		TextTemplateMap: minimalTextMap(),
		GameSystem:      tmpl,
	}

	_, err := am.Load(group.GroupId, player.UserId)
	require.NoError(t, err)

	msg := &types.Message{
		MessageType: "group",
		GroupID:     group.GroupId,
		Platform:    "test",
		Time:        time.Now().Unix(),
		Sender: types.SenderBase{
			UserID:   player.UserId,
			Nickname: player.Name,
		},
	}

	return ctx, msg, stub, cmd
}

func TestDnd5eStSetsSkillComputedValue(t *testing.T) {
	ctx, msg, stub, cmd := newDnd5eTestContext(t)

	executeStCommands(t, stub, ctx, msg, cmd, ".st 力量:15 运动*0.5:2")

	attrsItem, err := ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId)
	require.NoError(t, err)

	strVal, exists := attrsItem.Load("力量")
	require.True(t, exists, "strength should be recorded")
	require.Equal(t, ds.VMTypeInt, strVal.TypeId)
	require.Equal(t, int64(15), int64(strVal.MustReadInt()))

	skillVal, exists := attrsItem.Load("运动")
	require.True(t, exists, "athletics should be recorded")
	require.Equal(t, ds.VMTypeComputedValue, skillVal.TypeId)

	cd, ok := skillVal.ReadComputed()
	require.True(t, ok)
	require.NotNil(t, cd)
	require.Equal(t, "pbCalc(this.base, this.factor, 力量)", cd.Expr)

	base, _ := cd.Attrs.Load("base")
	require.NotNil(t, base)
	require.Equal(t, int64(2), int64(base.MustReadInt()))

	factor, _ := cd.Attrs.Load("factor")
	require.NotNil(t, factor)
	require.Equal(t, "0.5", factor.ToRepr())

	require.Equal(t, "dnd5e", attrsItem.SheetType)
}

func TestDnd5eStHpDamageUsesTemporaryHitpoints(t *testing.T) {
	ctx, msg, stub, cmd := newDnd5eTestContext(t)

	executeStCommands(t, stub, ctx, msg, cmd, ".st hp:10")

	attrsItem, err := ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId)
	require.NoError(t, err)
	attrsItem.Store("$buff_hp", ds.NewIntVal(4))

	executeStCommands(t, stub, ctx, msg, cmd, ".st hp-3")

	hpVal, exists := attrsItem.Load("hp")
	require.True(t, exists)
	require.Equal(t, int64(10), int64(hpVal.MustReadInt()))

	shield, exists := attrsItem.Load("$buff_hp")
	require.True(t, exists)
	require.Equal(t, int64(1), int64(shield.MustReadInt()))
}

func TestDnd5eStHpIncreaseRespectsCap(t *testing.T) {
	ctx, msg, stub, cmd := newDnd5eTestContext(t)

	executeStCommands(t, stub, ctx, msg, cmd, ".st hp:5", ".st hpmax:10")

	executeStCommands(t, stub, ctx, msg, cmd, ".st hp+8")

	attrsItem, err := ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId)
	require.NoError(t, err)

	hpVal, exists := attrsItem.Load("hp")
	require.True(t, exists)
	require.Equal(t, int64(10), int64(hpVal.MustReadInt()))
}

func TestDnd5eStShowDisplaysTemplateFormatting(t *testing.T) {
	ctx, msg, stub, cmd := newDnd5eTestContext(t)

	stCmd := ".st 力量:12 体质:10 敏捷:10 智力:10 感知:10 魅力:10 hp:10 hpmax:10 熟练:2 运动*:3 特技:1 巧手:0 隐匿:0 调查:0 奥秘:0 历史:0 自然:0 宗教:0 察觉:0 洞悉:0 驯养:0 医疗:0 生存:0 说服:0 欺诈:0 威吓:0 表演:0"
	_, _ = executeStCommand(t, stub, ctx, msg, stCmd, cmd)

	_, reply := executeStCommand(t, stub, ctx, msg, ".st show", cmd)
	require.Contains(t, reply, "LIST ")

	info := strings.TrimSpace(strings.TrimPrefix(reply, "LIST "))

	keySnippets := []string{
		"力量:12",
		"体型:0",
		"熟练:2",
		"hp:10/10",
		"体操:1[1]",
		"运动*:6[3]",
	}

	for _, snippet := range keySnippets {
		require.Containsf(t, info, snippet, "expected show output to contain %q\nfull reply: %s", snippet, reply)
	}
}

// func TestDnd5eRollDetailHidesComputedInternals(t *testing.T) {
// 	ctx, msg, stub, stCmd := newDnd5eTestContext(t)

// 	coreExt, ok := stub.extensions["core"]
// 	require.True(t, ok, "core extension should be registered")

// 	setCmd, ok := coreExt.CmdMap["set"]
// 	require.True(t, ok, "core extension should provide set command")

// 	rollCmd, ok := coreExt.CmdMap["roll"]
// 	require.True(t, ok, "core extension should provide roll command")

// 	_, _ = executeCommandWith(t, stub, ctx, msg, ".set dnd", setCmd, "set")

// 	executeStCommands(t, stub, ctx, msg, stCmd, ".st 敏捷:1", ".st &命中=敏捷调整值")

// 	_, reply := executeCommandWith(t, stub, ctx, msg, ".r 命中", rollCmd, "r", "ra", "rh", "rd")
// 	require.NotEmpty(t, reply)
// 	require.Contains(t, reply, "命中=")
// 	require.NotContains(t, reply, "function abilityModifier")
// }

func TestDnd5eRollUsesComputedAttr(t *testing.T) {
	ctx, msg, stub, stCmd := newDnd5eTestContext(t)

	coreExt, ok := stub.extensions["core"]
	require.True(t, ok, "core extension should be registered")

	setCmd, ok := coreExt.CmdMap["set"]
	require.True(t, ok, "core extension should provide set command")

	rollCmd, ok := coreExt.CmdMap["roll"]
	require.True(t, ok, "core extension should provide roll command")

	_, _ = executeCommandWith(t, stub, ctx, msg, ".set dnd", setCmd, "set")

	executeStCommands(t, stub, ctx, msg, stCmd, ".st 敏捷:20 熟练:2", ".st &命中=敏捷调整值+熟练+2")

	_, reply := executeCommandWith(t, stub, ctx, msg, ".r 命中", rollCmd, "r", "ra", "rh", "rd")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "命中=")
	require.Contains(t, reply, "=9")
	if strings.Count(reply, "=") >= 2 {
		require.True(t, strings.HasSuffix(reply, "=9"), "expected roll reply to end with '=9', got %q", reply)
	}
}

func TestDnd5eRaAbilityDisplaysModifierDetail(t *testing.T) {
	ctx, msg, stub, stCmd := newDnd5eTestContext(t)

	executeStCommands(t, stub, ctx, msg, stCmd, ".st 力量*:18")

	ext, ok := stub.extensions["dnd5e"]
	require.True(t, ok, "dnd5e extension should be registered")

	cmdRc, ok := ext.CmdMap["rc"]
	require.True(t, ok, "dnd5e extension should provide rc command")

	_, reply := executeCommandWith(t, stub, ctx, msg, ".ra 力量", cmdRc, "rc", "ra", "rah", "rch", "drc")
	require.NotEmpty(t, reply)
	require.Containsf(t, reply, "+ 4[力量调整值4]", "expected ability check detail to include ability modifier detail, got %q", reply)
}

func TestDnd5eBuffAffectsAbilityAndSkill(t *testing.T) {
	ctx, msg, stub, stCmd := newDnd5eTestContext(t)

	ext, ok := stub.extensions["dnd5e"]
	require.True(t, ok, "dnd5e extension should be registered")

	cmdBuff, ok := ext.CmdMap["buff"]
	require.True(t, ok, "dnd5e extension should provide buff command")

	strength := "\u529b\u91cf"
	athletics := "\u8fd0\u52a8"
	strengthMod := "\u529b\u91cf\u8c03\u6574\u503c"

	executeStCommands(t, stub, ctx, msg, stCmd, ".st "+strength+":10")

	_, _ = executeCommandWith(t, stub, ctx, msg, ".buff "+strength+":4", cmdBuff, "buff", "dbuff")

	attrsItem, err := ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId)
	require.NoError(t, err)
	abilityBuff, exists := attrsItem.Load("$buff_" + strength)
	require.True(t, exists, "expected strength buff to be stored")
	require.Equal(t, ds.VMTypeInt, abilityBuff.TypeId, "strength buff should store an integer value")

	setDndReadForVM(ctx, false)

	ability := ctx.Eval(strength, nil)
	require.NotNil(t, ability, "expected strength value to be readable")
	require.EqualValues(t, 14, ability.MustReadInt(), "buff should increase ability score")

	modifier := ctx.Eval(strengthMod, nil)
	require.NotNil(t, modifier, "expected strength modifier to be readable")
	require.EqualValues(t, 2, modifier.MustReadInt(), "buff should affect ability modifier")

	_, _ = executeCommandWith(t, stub, ctx, msg, ".buff "+athletics+":3", cmdBuff, "buff", "dbuff")

	skillBuff, exists := attrsItem.Load("$buff_" + athletics)
	require.True(t, exists, "expected athletics buff to be stored")
	require.Equal(t, ds.VMTypeComputedValue, skillBuff.TypeId, "athletics buff should store computed value")

	skill := ctx.Eval(athletics, nil)
	require.NotNil(t, skill, "expected athletics value to be readable")
	require.EqualValues(t, 5, skill.MustReadInt(), "buff should affect skill totals")
}

func TestDnd5eBuffAffectsRollCommand(t *testing.T) {
	ctx, msg, stub, stCmd := newDnd5eTestContext(t)

	coreExt, ok := stub.extensions["core"]
	require.True(t, ok)

	setCmd, ok := coreExt.CmdMap["set"]
	require.True(t, ok)

	rollCmd, ok := coreExt.CmdMap["roll"]
	require.True(t, ok)

	ext, ok := stub.extensions["dnd5e"]
	require.True(t, ok)

	buffCmd, ok := ext.CmdMap["buff"]
	require.True(t, ok)

	_, _ = executeCommandWith(t, stub, ctx, msg, ".set dnd", setCmd, "set")
	executeStCommands(t, stub, ctx, msg, stCmd, ".st 力量:4")

	_, _ = executeCommandWith(t, stub, ctx, msg, ".buff 力量:4", buffCmd, "buff", "dbuff")

	_, reply := executeCommandWith(t, stub, ctx, msg, ".r 力量+1", rollCmd, "r", "ra", "rh", "rd")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "=9", "expected buffed roll result to include total 9")
	require.Contains(t, reply, "力量+1=8", "expected buffed strength value to be 8, got %q", reply)
	require.Contains(t, reply, "buff4", "expected buff contribution in detail, got %q", reply)
}

func TestDnd5eStShowDisplaysBaseAndBuff(t *testing.T) {
	ctx, msg, stub, stCmd := newDnd5eTestContext(t)

	coreExt, ok := stub.extensions["core"]
	require.True(t, ok)

	setCmd, ok := coreExt.CmdMap["set"]
	require.True(t, ok)

	ext, ok := stub.extensions["dnd5e"]
	require.True(t, ok)

	buffCmd, ok := ext.CmdMap["buff"]
	require.True(t, ok)

	stCmdExt, ok := ext.CmdMap["st"]
	require.True(t, ok)

	_, _ = executeCommandWith(t, stub, ctx, msg, ".set dnd", setCmd, "set")
	executeStCommands(t, stub, ctx, msg, stCmd, ".st 力量:4")
	_, _ = executeCommandWith(t, stub, ctx, msg, ".buff 力量:4", buffCmd, "buff", "dbuff")

	_, reply := executeCommandWith(t, stub, ctx, msg, ".st show", stCmdExt, "st")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "力量:8[4]", "expected show output to include buffed and base values, got %q", reply)
}
