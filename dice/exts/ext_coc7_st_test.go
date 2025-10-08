package exts

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"
	ds "github.com/sealdice/dicescript"
	"github.com/stretchr/testify/require"

	"github.com/sealdice/smallseal/dice/attrs"
	"github.com/sealdice/smallseal/dice/types"
	"github.com/sealdice/smallseal/utils"
)

type stubDice struct {
	templates  *utils.SyncMap[string, *types.GameSystemTemplateV2]
	replies    []*types.MsgToReply
	extensions map[string]*types.ExtInfo
}

func newStubDice(tmpl *types.GameSystemTemplateV2) *stubDice {
	tmplMap := &utils.SyncMap[string, *types.GameSystemTemplateV2]{}
	tmplMap.Store(tmpl.Name, tmpl)

	return &stubDice{
		templates:  tmplMap,
		extensions: make(map[string]*types.ExtInfo),
	}
}

func (s *stubDice) GameSystemMapGet() *utils.SyncMap[string, *types.GameSystemTemplateV2] {
	return s.templates
}

func (s *stubDice) RegisterExtension(info *types.ExtInfo) {
	if info == nil {
		return
	}
	s.extensions[info.Name] = info
}

func (s *stubDice) GameSystemMapLoad(name string) (*types.GameSystemTemplateV2, error) {
	if tmpl, ok := s.templates.Load(name); ok {
		return tmpl, nil
	}
	return nil, fmt.Errorf("template %s not found", name)
}

func (s *stubDice) ExtFind(string, bool) *types.ExtInfo { return nil }

func (s *stubDice) GetExtList() []*types.ExtInfo { return nil }

func (s *stubDice) SendReply(msg *types.MsgToReply) {
	s.replies = append(s.replies, msg)
}

func (s *stubDice) RegisterMessageInHook(string, types.HookPriority, types.MessageInHook) (types.HookHandle, error) {
	return "", nil
}

func (s *stubDice) UnregisterMessageInHook(types.HookHandle) bool { return true }

func (s *stubDice) RegisterMessageOutHook(string, types.HookPriority, types.MessageOutHook) (types.HookHandle, error) {
	return "", nil
}

func (s *stubDice) UnregisterMessageOutHook(types.HookHandle) bool { return true }

func (s *stubDice) RegisterEventHook(string, types.HookPriority, types.EventHook) (types.HookHandle, error) {
	return "", nil
}

func (s *stubDice) UnregisterEventHook(types.HookHandle) bool { return true }

func (s *stubDice) DispatchEvent(string, *types.AdapterEvent) {}

func (s *stubDice) MasterAdd(string) {}

func (s *stubDice) MasterRemove(string) bool { return false }

func (s *stubDice) ListMasters() []string { return nil }

func (s *stubDice) IsMaster(string) bool { return false }

func (s *stubDice) PersistGroupInfo(string, *types.GroupInfo) {}

func minimalTextMap() types.TextTemplateWithWeightDict {
	toItem := func(text string) types.TextTemplateItem {
		return types.TextTemplateItem{text, 1}
	}

	return types.TextTemplateWithWeightDict{
		"COC": types.TextTemplateWithWeight{
			"属性设置": []types.TextTemplateItem{
				toItem("SET {$t玩家} {$t规则模板} {$t有效数量}"),
			},
			"属性设置_列出": []types.TextTemplateItem{
				toItem("LIST {$t属性信息}"),
			},
			"属性设置_列出_未发现记录": []types.TextTemplateItem{
				toItem("LIST_EMPTY"),
			},
			"属性设置_列出_隐藏提示": []types.TextTemplateItem{
				toItem("HIDDEN {$t数量} {$t判定值}"),
			},
			"属性设置_删除": []types.TextTemplateItem{
				toItem("DEL {$t删除列表} FAIL {$t失败数量}"),
			},
			"属性设置_清除": []types.TextTemplateItem{
				toItem("CLEAR {$t玩家}"),
			},
			"属性设置_增减": []types.TextTemplateItem{
				toItem("MOD {$t变更列表}"),
			},
			"属性设置_增减_单项": []types.TextTemplateItem{
				toItem("ITEM {$t属性}: {$t旧值}->{$t新值} ({$t增加或减少}{$t表达式}={$t变化值})"),
			},
		},
		"DND": types.TextTemplateWithWeight{
			"检定": []types.TextTemplateItem{
				toItem("{$t玩家}的\"{$t技能}\"检定（DND5E）结果为: {$t检定过程文本} = {$t检定结果}"),
			},
			"检定_单项结果文本": []types.TextTemplateItem{
				toItem("{$t检定过程文本} = {$t检定结果}"),
			},
			"检定_多轮": []types.TextTemplateItem{
				toItem("对{$t玩家}的\"{$t技能}\"进行了{$t次数}次检定（DND5E），结果为:\n{$t结果文本}"),
			},
		},
		"核心": types.TextTemplateWithWeight{
			"骰点": []types.TextTemplateItem{
				toItem("{$t结果文本}"),
			},
			"骰点_单项结果文本": []types.TextTemplateItem{
				toItem("{$t表达式文本}{$t计算过程}={$t计算结果}"),
			},
			"骰点_原因": []types.TextTemplateItem{
				toItem("{$t原因句子}"),
			},
		},
	}
}

func newCoc7TestContext(t *testing.T) (*types.MsgContext, *types.Message, *stubDice) {
	t.Helper()

	tmplPath := filepath.Join("..", "..", "coc7.yaml")
	tmpl, err := types.LoadGameSystemTemplate(tmplPath)
	require.NoError(t, err)

	stub := newStubDice(tmpl)

	am := &attrs.AttrsManager{}
	am.SetIO(attrs.NewMemoryAttrsIO())

	group := &types.GroupInfo{
		GroupId: "group-1",
		System:  "coc7",
		BotList: &utils.SyncMap[string, bool]{},
	}

	player := &types.GroupPlayerInfo{
		UserId: "user-1",
		Name:   "调查员A",
	}

	ctx := &types.MsgContext{
		Dice:            stub,
		AttrsManager:    am,
		Group:           group,
		Player:          player,
		TextTemplateMap: minimalTextMap(),
		GameSystem:      tmpl,
	}

	_, err = am.Load(group.GroupId, player.UserId)
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

	return ctx, msg, stub
}

func executeCommandWith(t *testing.T, stub *stubDice, ctx *types.MsgContext, msg *types.Message, raw string, cmd *types.CmdItemInfo, names ...string) (types.CmdExecuteResult, string) {
	t.Helper()

	cmdArgs := types.CommandParse(raw, names, []string{".", "。"}, "", false)
	require.NotNilf(t, cmdArgs, "failed to parse command %q", raw)

	ctx.CommandId++

	prev := len(stub.replies)
	result := cmd.Solve(ctx, msg, cmdArgs)

	require.Truef(t, result.Matched, "command %q should match", raw)
	require.Truef(t, result.Solved, "command %q should be solved", raw)

	if len(stub.replies) > prev {
		reply := stub.replies[len(stub.replies)-1]
		return result, reply.Segments.ToText()
	}

	return result, ""
}

func executeStCommand(t *testing.T, stub *stubDice, ctx *types.MsgContext, msg *types.Message, raw string, cmd *types.CmdItemInfo) (types.CmdExecuteResult, string) {
	return executeCommandWith(t, stub, ctx, msg, raw, cmd, "st")
}

func executeStCommands(t *testing.T, stub *stubDice, ctx *types.MsgContext, msg *types.Message, cmd *types.CmdItemInfo, commands ...string) {
	for _, raw := range commands {
		executeStCommand(t, stub, ctx, msg, raw, cmd)
	}
}

func attrIntValue(t *testing.T, ctx *types.MsgContext, item *attrs.AttributesItem, name string) int64 {
	t.Helper()

	keys := []string{name}
	trimmed := strings.Trim(name, "'")
	if trimmed != name {
		keys = append(keys, trimmed)
	} else {
		keys = append(keys, "'"+name+"'")
	}
	if ctx != nil && ctx.GameSystem != nil {
		alias := ctx.GameSystem.GetAlias(name)
		keys = append(keys, alias)
	}

	seen := map[string]struct{}{}
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if val, ok := item.Load(key); ok {
			return int64(val.MustReadInt())
		}
	}

	t.Fatalf("attribute %s not found (tried %v)", name, keys)
	return 0
}

func collectAttrInts(item *attrs.AttributesItem) map[string]int64 {
	result := make(map[string]int64)
	item.GetValueMap().Range(func(key string, value *ds.VMValue) bool {
		if value != nil && value.TypeId == ds.VMTypeInt {
			result[key] = int64(value.MustReadInt())
		}
		return true
	})
	return result
}

func TestCoc7StHelp(t *testing.T) {
	ctx, msg, stub := newCoc7TestContext(t)
	cmd := getCmdStBase(CmdStOverrideInfo{})

	result, reply := executeStCommand(t, stub, ctx, msg, ".st help", cmd)
	require.True(t, result.ShowHelp)
	require.Empty(t, reply)
}

func TestCoc7StSetCommands(t *testing.T) {
	ctx, msg, stub := newCoc7TestContext(t)
	cmd := getCmdStBase(CmdStOverrideInfo{})

	executeStCommands(t, stub, ctx, msg, cmd, ".sthp13", ".stsan60dex60", ".stsan60dex60hp13")

	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))

	require.Equal(t, "coc7", attrsItem.SheetType)
	require.Equal(t, int64(13), attrIntValue(t, ctx, attrsItem, "生命值"))
	require.Equal(t, int64(60), attrIntValue(t, ctx, attrsItem, "理智"))
	require.Equal(t, int64(60), attrIntValue(t, ctx, attrsItem, "敏捷"))
}

func TestCoc7StSetWithExplicitName(t *testing.T) {
	ctx, msg, stub := newCoc7TestContext(t)
	cmd := getCmdStBase(CmdStOverrideInfo{})

	_, reply := executeStCommand(t, stub, ctx, msg, ".st 张三力量70", cmd)
	require.Contains(t, reply, "SET")

	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))
	attrs := collectAttrInts(attrsItem)

	found := false
	for key, val := range attrs {
		if strings.HasSuffix(key, "力量") {
			require.Equal(t, int64(70), val)
			found = true
		}
	}
	require.Truef(t, found, "expected attribute ending with 力量, got %v", attrs)
}

func TestCoc7StShowEmpty(t *testing.T) {
	ctx, msg, stub := newCoc7TestContext(t)
	cmd := getCmdStBase(CmdStOverrideInfo{})

	_, reply := executeStCommand(t, stub, ctx, msg, ".st show", cmd)
	require.Contains(t, reply, "LIST_EMPTY")
}

func TestCoc7StShowPickAndLimit(t *testing.T) {
	ctx, msg, stub := newCoc7TestContext(t)
	cmd := getCmdStBase(CmdStOverrideInfo{})

	executeStCommands(t, stub, ctx, msg, cmd, ".st摄影20", ".st跳跃70")

	_, replyPick := executeStCommand(t, stub, ctx, msg, ".st show 摄影", cmd)
	require.Contains(t, replyPick, "摄影")
	require.NotContains(t, replyPick, "跳跃")

	_, replyLimit := executeStCommand(t, stub, ctx, msg, ".st show 30", cmd)
	require.Contains(t, replyLimit, "跳跃")
	require.NotContains(t, replyLimit, "摄影")
	require.Contains(t, replyLimit, "HIDDEN")
}

func TestCoc7StShowComputedExpression(t *testing.T) {
	ctx, msg, stub := newCoc7TestContext(t)
	cmd := getCmdStBase(CmdStOverrideInfo{})

	executeStCommands(t, stub, ctx, msg, cmd, ".st &手枪=1d3+1d5")

	_, reply := executeStCommand(t, stub, ctx, msg, ".st show 手枪", cmd)
	require.Contains(t, reply, "手枪:&(1d3+1d5)")
}

func TestCoc7StShowDbKeepsExpression(t *testing.T) {
	ctx, msg, stub := newCoc7TestContext(t)
	cmd := getCmdStBase(CmdStOverrideInfo{})

	executeStCommands(t, stub, ctx, msg, cmd, ".st力量90", ".st体型80")

	_, reply := executeStCommand(t, stub, ctx, msg, ".st show db", cmd)
	require.Contains(t, reply, "db")
	require.Contains(t, reply, "1d6")
}

func TestCoc7StShowManualStrengthAndBuild(t *testing.T) {
	ctx, msg, stub := newCoc7TestContext(t)
	cmd := getCmdStBase(CmdStOverrideInfo{})

	executeStCommands(t, stub, ctx, msg, cmd, ".st力量200", ".st体格=3", ".st db=5")

	_, reply := executeStCommand(t, stub, ctx, msg, ".st show 力量 体格 db", cmd)
	require.Contains(t, reply, "力量:200")
	require.Contains(t, reply, "体格:3")
	require.Contains(t, reply, "db:5")
}

func TestCoc7StShowAliasAndChinesePrefix(t *testing.T) {
	ctx, msg, stub := newCoc7TestContext(t)
	cmd := getCmdStBase(CmdStOverrideInfo{})

	executeStCommands(t, stub, ctx, msg, cmd, ".st体型55")

	_, reply := executeStCommand(t, stub, ctx, msg, "。st show siz", cmd)
	require.Contains(t, reply, "体型")
}

func TestCoc7StExport(t *testing.T) {
	ctx, msg, stub := newCoc7TestContext(t)
	cmd := getCmdStBase(CmdStOverrideInfo{})

	executeStCommands(t, stub, ctx, msg, cmd, ".st敏捷60", ".st图书馆45")

	_, reply := executeStCommand(t, stub, ctx, msg, ".st export", cmd)
	require.Contains(t, reply, ".st clr")
	require.Contains(t, reply, "敏捷:60")
	require.Contains(t, reply, "图书馆使用:45")
	require.Contains(t, reply, "调查员A")
	require.NotContains(t, reply, "&db:")
}

func TestCoc7StExportEmpty(t *testing.T) {
	ctx, msg, stub := newCoc7TestContext(t)
	cmd := getCmdStBase(CmdStOverrideInfo{})

	_, reply := executeStCommand(t, stub, ctx, msg, ".st export", cmd)
	require.Contains(t, reply, "LIST_EMPTY")
}

func TestCoc7StDeleteAndClear(t *testing.T) {
	ctx, msg, stub := newCoc7TestContext(t)
	cmd := getCmdStBase(CmdStOverrideInfo{})

	executeStCommands(t, stub, ctx, msg, cmd, ".st敏捷60", ".st图书馆45")

	_, delReply := executeStCommand(t, stub, ctx, msg, ".st del 敏捷", cmd)
	require.Contains(t, delReply, "DEL")
	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))
	if _, exists := attrsItem.Load("敏捷"); exists {
		t.Fatalf("attribute 敏捷 should be deleted")
	}

	_, rmReply := executeStCommand(t, stub, ctx, msg, ".st rm 图书馆", cmd)
	require.Contains(t, rmReply, "DEL")
	if _, exists := attrsItem.Load("图书馆使用"); exists {
		t.Fatalf("attribute 图书馆使用 should be deleted")
	}

	executeStCommands(t, stub, ctx, msg, cmd, ".st敏捷60", ".st摄影30")
	_, clrReply := executeStCommand(t, stub, ctx, msg, ".st clr", cmd)
	require.Contains(t, clrReply, "CLEAR")
	require.Equal(t, 0, attrsItem.Len())
}

func TestCoc7StIncreaseAndDecrease(t *testing.T) {
	ctx, msg, stub := newCoc7TestContext(t)
	cmd := getCmdStBase(CmdStOverrideInfo{})

	executeStCommands(t, stub, ctx, msg, cmd, ".st敏捷60")

	_, incReply := executeStCommand(t, stub, ctx, msg, ".st敏捷+5", cmd)
	require.Contains(t, incReply, "MOD")
	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))
	require.Equal(t, int64(65), attrIntValue(t, ctx, attrsItem, "敏捷"))

	_, decReply := executeStCommand(t, stub, ctx, msg, ".st敏捷-3", cmd)
	require.Contains(t, decReply, "MOD")
	require.Equal(t, int64(62), attrIntValue(t, ctx, attrsItem, "敏捷"))
}

func TestCoc7StFormatForcesCardType(t *testing.T) {
	ctx, msg, stub := newCoc7TestContext(t)
	cmd := getCmdStBase(CmdStOverrideInfo{})

	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))
	attrsItem.SetSheetType("dnd5e")

	_, reply := executeStCommand(t, stub, ctx, msg, ".st fmt", cmd)
	require.Contains(t, reply, "强制修改")
	require.Equal(t, "coc7", attrsItem.SheetType)
}

func TestCoc7StShowAfterSet(t *testing.T) {
	ctx, msg, stub := newCoc7TestContext(t)
	cmd := getCmdStBase(CmdStOverrideInfo{})

	executeStCommands(t, stub, ctx, msg, cmd, ".stsan60dex60hp13")

	_, reply := executeStCommand(t, stub, ctx, msg, ".st show", cmd)
	require.Contains(t, reply, "LIST")
	require.Contains(t, reply, "理智:60")
	require.Contains(t, reply, "敏捷:60")
}

func TestCoc7StComplexExpressions(t *testing.T) {
	ctx, msg, stub := newCoc7TestContext(t)
	cmd := getCmdStBase(CmdStOverrideInfo{})

	complexCmd := ".st 力量+2 敏捷+1d3 理智+4d1+2 '测试2'+10 测试测试+10 测试测试+=10 '测试测试2'+4d1 测试新-10 测试新-4d1-2 体型-=4 '测试测试3'-1"
	executeStCommands(t, stub, ctx, msg, cmd, complexCmd)

	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))

	require.Equal(t, int64(2), attrIntValue(t, ctx, attrsItem, "力量"))

	dex := attrIntValue(t, ctx, attrsItem, "敏捷")
	require.GreaterOrEqual(t, dex, int64(1))
	require.LessOrEqual(t, dex, int64(3))

	require.Equal(t, int64(6), attrIntValue(t, ctx, attrsItem, "理智"))
	require.Equal(t, int64(10), attrIntValue(t, ctx, attrsItem, "测试2"))
	require.Equal(t, int64(20), attrIntValue(t, ctx, attrsItem, "测试测试"))
	require.Equal(t, int64(4), attrIntValue(t, ctx, attrsItem, "测试测试2"))
	require.Equal(t, int64(-16), attrIntValue(t, ctx, attrsItem, "测试新"))
	require.Equal(t, int64(-4), attrIntValue(t, ctx, attrsItem, "体型"))
	require.Equal(t, int64(-1), attrIntValue(t, ctx, attrsItem, "测试测试3"))
}

func setupCoc7RaTest(t *testing.T) (*types.MsgContext, *types.Message, *stubDice, *types.CmdItemInfo) {
	ctx, msg, stub := newCoc7TestContext(t)

	RegisterBuiltinExtCoc7(stub)

	ext, ok := stub.extensions["coc7"]
	require.True(t, ok)

	ctx.Group.ExtActive(ext)

	cmd, ok := ext.CmdMap["ra"]
	require.True(t, ok)

	return ctx, msg, stub, cmd
}

func TestCoc7RaBonusDiceShorthand(t *testing.T) {
	ctx, msg, stub, cmd := setupCoc7RaTest(t)

	ctx.CommandInfo = nil
	stub.replies = nil

	_, reply := executeCommandWith(t, stub, ctx, msg, ".rab力量50", cmd, "rc", "ra", "rah", "rch")
	require.NotEmpty(t, reply)

	info := ctx.CommandInfo
	require.NotNil(t, info)

	itemsAny, ok := info["items"]
	require.True(t, ok)

	items, ok := itemsAny.([]interface{})
	require.True(t, ok)
	require.Len(t, items, 1)

	item, ok := items[0].(map[string]interface{})
	require.True(t, ok)

	expr1, ok := item["expr1"].(string)
	require.True(t, ok)
	require.NotEmpty(t, expr1)
	require.Equal(t, "b", expr1)

	expr2, ok := item["expr2"].(string)
	require.True(t, ok)
	require.NotEmpty(t, expr2)
	require.Equal(t, "力量50", expr2)
}

func TestCoc7RaMultipleBonusDiceShorthand(t *testing.T) {
	ctx, msg, stub, cmd := setupCoc7RaTest(t)

	ctx.CommandInfo = nil
	stub.replies = nil

	_, reply := executeCommandWith(t, stub, ctx, msg, ".rab3力量50", cmd, "rc", "ra", "rah", "rch")
	require.NotEmpty(t, reply)

	info := ctx.CommandInfo
	require.NotNil(t, info)

	itemsAny, ok := info["items"]
	require.True(t, ok)

	items, ok := itemsAny.([]interface{})
	require.True(t, ok)
	require.Len(t, items, 1)

	item, ok := items[0].(map[string]interface{})
	require.True(t, ok)

	expr1, ok := item["expr1"].(string)
	require.True(t, ok)
	require.NotEmpty(t, expr1)
	require.Equal(t, "b3", expr1)

	expr2, ok := item["expr2"].(string)
	require.True(t, ok)
	require.NotEmpty(t, expr2)
	require.Equal(t, "力量50", expr2)
}

func TestCoc7RaInlineBonusDiceParenthesis(t *testing.T) {
	ctx, msg, stub, cmd := setupCoc7RaTest(t)

	ctx.CommandInfo = nil
	stub.replies = nil

	_, reply := executeCommandWith(t, stub, ctx, msg, ".ra b3(1)50", cmd, "rc", "ra", "rah", "rch")
	require.NotEmpty(t, reply)

	info := ctx.CommandInfo
	require.NotNil(t, info)

	itemsAny, ok := info["items"]
	require.True(t, ok)

	items, ok := itemsAny.([]interface{})
	require.True(t, ok)
	require.Len(t, items, 1)

	item, ok := items[0].(map[string]interface{})
	require.True(t, ok)

	expr1, ok := item["expr1"].(string)
	require.True(t, ok)
	require.NotEmpty(t, expr1)

	expr2, ok := item["expr2"].(string)
	require.True(t, ok)
	require.NotEmpty(t, expr2)

	require.Equal(t, "b3", expr1)

}

func TestCoc7RaInlineBonusDiceShortcutParenthesis(t *testing.T) {
	ctx, msg, stub, cmd := setupCoc7RaTest(t)

	ctx.CommandInfo = nil
	stub.replies = nil

	_, reply := executeCommandWith(t, stub, ctx, msg, ".rab3(1)50", cmd, "rc", "ra", "rah", "rch")
	require.NotEmpty(t, reply)

	info := ctx.CommandInfo
	require.NotNil(t, info)

	itemsAny, ok := info["items"]
	require.True(t, ok)

	items, ok := itemsAny.([]interface{})
	require.True(t, ok)
	require.Len(t, items, 1)

	item, ok := items[0].(map[string]interface{})
	require.True(t, ok)

	expr1, ok := item["expr1"].(string)
	require.True(t, ok)
	require.NotEmpty(t, expr1)

	expr2, ok := item["expr2"].(string)
	require.True(t, ok)
	require.NotEmpty(t, expr2)

	require.Equal(t, "b3", expr1)

}

func TestCoc7RaParenthesisArgument(t *testing.T) {
	ctx, msg, stub, cmd := setupCoc7RaTest(t)

	ctx.CommandInfo = nil
	stub.replies = nil

	result, reply := executeCommandWith(t, stub, ctx, msg, ".ra(1)50", cmd, "rc", "ra", "rah", "rch")

	require.True(t, result.Matched)
	require.True(t, result.Solved)
	require.NotEmpty(t, reply)

	info := ctx.CommandInfo
	require.NotNil(t, info)

	itemsAny, ok := info["items"]
	require.True(t, ok)

	items, ok := itemsAny.([]interface{})
	require.True(t, ok)
	require.Len(t, items, 1)

	item, ok := items[0].(map[string]interface{})
	require.True(t, ok)

	expr1, ok := item["expr1"].(string)
	require.True(t, ok)
	require.NotEmpty(t, expr1)
	require.Equal(t, "(1)", expr1)

	expr2, ok := item["expr2"].(string)
	require.True(t, ok)
	require.NotEmpty(t, expr2)
	require.Equal(t, "50", expr2)
}
func TestCoc7SanCheckPersistsToAttributes(t *testing.T) {
	ctx, msg, stub := newCoc7TestContext(t)

	RegisterBuiltinExtCoc7(stub)
	ext, ok := stub.extensions["coc7"]
	require.True(t, ok)

	ctx.Group.ExtActive(ext)

	cmdSc, ok := ext.CmdMap["sc"]
	require.True(t, ok)

	cmdSt := getCmdStBase(CmdStOverrideInfo{})

	executeStCommands(t, stub, ctx, msg, cmdSt, ".stsan60")

	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))
	require.Equal(t, int64(60), attrIntValue(t, ctx, attrsItem, "san"))

	// 第一次检定 52，因此扣除6点
	_, _ = executeCommandWith(t, stub, ctx, msg, ".sc 52 6/11", cmdSc, "sc")
	require.Equal(t, int64(54), attrIntValue(t, ctx, attrsItem, "san"))

	// 第二次检定 51，扣除5点。之前bug为sc不能扣除理智
	_, _ = executeCommandWith(t, stub, ctx, msg, ".sc 51 5/9", cmdSc, "sc")
	require.Equal(t, int64(49), attrIntValue(t, ctx, attrsItem, "san"))
}
