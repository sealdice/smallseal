package exts

// 用户变量相关
// 这两个helper文件基本上出于可移植性上的考虑，来兼容旧的，其实很多写法可能有点过时了

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	ds "github.com/sealdice/dicescript"
	"golang.org/x/exp/rand"

	"github.com/sealdice/smallseal/dice/types"
)

func VarSetValueStr(ctx *types.MsgContext, s string, v string) {
	VarSetValue(ctx, s, ds.NewStrVal(v))
}

func VarSetValueInt64(ctx *types.MsgContext, s string, v int64) {
	VarSetValue(ctx, s, ds.NewIntVal(ds.IntType(v)))
}

func VarSetValueComputed(ctx *types.MsgContext, s string, v string) {
	VarSetValue(ctx, s, ds.NewComputedVal(v))
}

func VarSetValue(ctx *types.MsgContext, s string, v *ds.VMValue) {
	vm := ctx.GetVM()
	name := s
	// name := ctx.Player.GetValueNameByAlias(s, nil)
	vClone := v.Clone()

	// 临时变量
	if strings.HasPrefix(s, "$t") {
		// 如果是内部设置的临时变量，不需要长期存活
		//  if ctx.Player.ValueMapTemp == nil {
		//  	ctx.Player.ValueMapTemp = &ds.ValueMap{}
		//  }
		//  ctx.Player.ValueMapTemp.Store(s, vClone)
		// 用这个替代 temp 吗？我不太确定
		vm.StoreNameLocal(name, vClone)
		return
	}

	// 个人变量
	if strings.HasPrefix(name, "$m") {
		playerAttrs := ctx.LoadAttrsForCurUser()
		if playerAttrs != nil {
			playerAttrs.Store(name, vClone)
		}
		return
	}

	// 群变量
	if ctx.Group != nil && strings.HasPrefix(s, "$g") {
		groupAttrs := ctx.LoadAttrsForCurGroup()
		if groupAttrs != nil {
			groupAttrs.Store(name, vClone)
		}
		return
	}

	// 个人群变量
	curAttrs := ctx.LoadAttrsForCurGroupUser()
	if curAttrs != nil {
		curAttrs.Store(name, vClone)
	}
}

func VarDelValue(ctx *types.MsgContext, s string) {
	// name := ctx.Player.GetValueNameByAlias(s, nil)
	name := s

	// 临时变量
	if strings.HasPrefix(s, "$t") {
		vm := ctx.GetVM()
		vm.Attrs.Delete(s)
		return
	}

	// 个人变量
	if strings.HasPrefix(s, "$m") {
		playerAttrs := ctx.LoadAttrsForCurUser()
		if playerAttrs != nil {
			playerAttrs.Delete(name)
		}
		return
	}

	// 群变量
	if ctx.Group != nil && strings.HasPrefix(s, "$g") {
		groupAttrs := ctx.LoadAttrsForCurGroup()
		if groupAttrs != nil {
			groupAttrs.Delete(s)
		}
		return
	}

	curAttrs := ctx.LoadAttrsForCurGroupUser()
	if curAttrs != nil {
		curAttrs.Delete(name)
	}
}

func VarGetValueInt64(ctx *types.MsgContext, s string) (int64, bool) {
	v, exists := VarGetValue(ctx, s)
	if exists && v.TypeId == ds.VMTypeInt {
		return int64(v.MustReadInt()), true
	}
	return 0, false
}

func VarGetValueComputed(ctx *types.MsgContext, s string) (string, bool) {
	v, exists := VarGetValue(ctx, s)
	if exists && v.TypeId == ds.VMTypeComputedValue {
		return v.Value.(*ds.ComputedData).Expr, true
	}
	return "", false
}

func VarGetValue(ctx *types.MsgContext, s string) (*ds.VMValue, bool) {
	// name := ctx.Player.GetValueNameByAlias(s, nil)
	vm := ctx.GetVM()
	name := s

	// 临时变量
	if strings.HasPrefix(s, "$t") {
		if vm != nil {
			v, ok := vm.Attrs.Load(s)
			if ok {
				return v, ok
			}
		}
		// 跟入群致辞闪退的一个bug有关，当时是报 _v, exists := ctx.Player.ValueMapTemp.Get(s) 这一行 nil pointer
		if ctx.Player != nil && ctx.Player.ValueMapTemp != nil {
			if v, ok := ctx.Player.ValueMapTemp.Load(s); ok {
				return v, ok
			}
		}
		return nil, false
	}

	// 个人变量
	if strings.HasPrefix(s, "$m") {
		playerAttrs := ctx.LoadAttrsForCurUser()
		if playerAttrs != nil {
			return playerAttrs.Load(name)
		}
		return nil, false
	}

	// 群变量
	if ctx.Group != nil && strings.HasPrefix(s, "$g") {
		groupAttrs := ctx.LoadAttrsForCurGroup()
		if groupAttrs != nil {
			return groupAttrs.Load(s)
		}
		return nil, false
	}

	// 个人群变量
	curAttrs := ctx.LoadAttrsForCurGroupUser()
	if curAttrs != nil {
		return curAttrs.Load(name)
	}

	return nil, false
}

func GetValueNameByAlias(s string, alias map[string][]string) string {
	name := s

	for k, v := range alias {
		if strings.EqualFold(s, k) {
			name = k // 防止一手大小写不一致
			break    // 名字本身就是确定值，不用修改
		}
		for _, i := range v {
			if strings.EqualFold(s, i) {
				name = k
				break
			}
		}
	}

	return name
}

func SetTempVars(ctx *types.MsgContext, qqNickname string) {
	// 设置临时变量
	if ctx.Player != nil {
		pcName := ctx.Player.Name
		pcName = strings.ReplaceAll(pcName, "\n", "")
		pcName = strings.ReplaceAll(pcName, "\r", "")
		pcName = strings.ReplaceAll(pcName, `\n`, "")
		pcName = strings.ReplaceAll(pcName, `\r`, "")
		pcName = strings.ReplaceAll(pcName, `\f`, "")

		VarSetValueStr(ctx, "$t玩家", fmt.Sprintf("<%s>", pcName))
		// if ctx.Dice != nil && !ctx.Dice.Config.PlayerNameWrapEnable {
		// 	VarSetValueStr(ctx, "$t玩家", pcName)
		// }
		VarSetValueStr(ctx, "$t玩家_RAW", pcName)
		VarSetValueStr(ctx, "$tQQ昵称", fmt.Sprintf("<%s>", qqNickname))
		VarSetValueStr(ctx, "$t帐号昵称", fmt.Sprintf("<%s>", qqNickname))
		VarSetValueStr(ctx, "$t账号昵称", fmt.Sprintf("<%s>", qqNickname))
		VarSetValueStr(ctx, "$t帐号ID", ctx.Player.UserId)
		VarSetValueStr(ctx, "$t账号ID", ctx.Player.UserId)
		VarSetValueStr(ctx, "$t个人骰子", ctx.Player.DiceSideExpr)
		VarSetValueStr(ctx, "$tQQ", ctx.Player.UserId)
		// VarSetValueStr(ctx, "$t骰子帐号", ctx.EndPoint.UserID)
		// VarSetValueStr(ctx, "$t骰子账号", ctx.EndPoint.UserID)
		// VarSetValueStr(ctx, "$t骰子昵称", ctx.EndPoint.Nickname)
		// VarSetValueStr(ctx, "$t帐号ID_RAW", UserIDExtract(ctx.Player.UserID))
		// VarSetValueStr(ctx, "$t账号ID_RAW", UserIDExtract(ctx.Player.UserID))

		// if ctx.EndPoint.ProtocolType != "official" {
		// 	VarSetValueStr(ctx, "$t平台", ctx.EndPoint.Platform)
		// } else {
		// 	VarSetValueStr(ctx, "$t平台", "QQ-official")
		// }

		// rpSeed := (time.Now().Unix() + (8 * 60 * 60)) / (24 * 60 * 60)
		// rpSeed += int64(fingerprint(ctx.EndPoint.UserID))
		// rpSeed += int64(fingerprint(ctx.Player.UserID))
		// randItem := rand.NewSource(rpSeed)
		// rp := randItem.Int63()%100 + 1
		// VarSetValueInt64(ctx, "$t人品", rp)

		now := time.Now()
		t, _ := strconv.ParseInt(now.Format("20060102"), 10, 64)
		VarSetValueInt64(ctx, "$tDate", t)
		wd := int64(now.Weekday())
		if wd == 0 {
			wd = 7
		}
		VarSetValueInt64(ctx, "$tWeekday", wd)

		t, _ = strconv.ParseInt(now.Format("2006"), 10, 64)
		VarSetValueInt64(ctx, "$tYear", t)
		t, _ = strconv.ParseInt(now.Format("01"), 10, 64)
		VarSetValueInt64(ctx, "$tMonth", t)
		t, _ = strconv.ParseInt(now.Format("02"), 10, 64)
		VarSetValueInt64(ctx, "$tDay", t)
		t, _ = strconv.ParseInt(now.Format("15"), 10, 64)
		VarSetValueInt64(ctx, "$tHour", t)
		t, _ = strconv.ParseInt(now.Format("04"), 10, 64)
		VarSetValueInt64(ctx, "$tMinute", t)
		t, _ = strconv.ParseInt(now.Format("05"), 10, 64)
		VarSetValueInt64(ctx, "$tSecond", t)
		VarSetValueInt64(ctx, "$tTimestamp", now.Unix())
	}

	if ctx.Group != nil {
		// 	if ctx.MessageType == "group" {
		// 		VarSetValueStr(ctx, "$t群号", ctx.Group.GroupID)
		// 		VarSetValueStr(ctx, "$t群名", ctx.Group.GroupName)
		// 		VarSetValueStr(ctx, "$t群号_RAW", UserIDExtract(ctx.Group.GroupID))
		// 	}
		VarSetValueStr(ctx, "$t群组骰子", ctx.Group.DiceSideExpr)
		// VarSetValueInt64(ctx, "$t当前骰子面数", getDefaultDicePoints(ctx))
		// 	if ctx.Group.System == "" {
		// 		ctx.Group.System = "coc7"
		// 		ctx.Group.UpdatedAtTime = time.Now().Unix()
		// 	}
		VarSetValueStr(ctx, "$t游戏模式", ctx.Group.System)
		VarSetValueStr(ctx, "$t规则模板", ctx.Group.System)
		VarSetValueStr(ctx, "$tSystem", ctx.Group.System)
		// 	VarSetValueStr(ctx, "$t当前记录", ctx.Group.LogCurName)
		// 	VarSetValueInt64(ctx, "$t权限等级", int64(ctx.PrivilegeLevel))

		// 	var isLogOn int64
		// 	if ctx.Group.LogOn {
		// 		isLogOn = 1
		// 	}
		// 	VarSetValueInt64(ctx, "$t日志开启", isLogOn)
		// }

		// if ctx.MessageType == "group" {
		// 	VarSetValueStr(ctx, "$t消息类型", "group")
		// } else {
		// 	VarSetValueStr(ctx, "$t消息类型", "private")
	}
}

func DiceFormatTmpl(ctx *types.MsgContext, s string) string {
	// 注: 这里有问题，格式化取值应该是变量重载的一环，这会导致模板中载入模板变量失败
	// 后记: 这段代码与前面重复，但是暂时没想明白怎么改
	var text string

	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return "<%未知项-" + s + "%>"
	}

	items := ctx.TextTemplateMap[parts[0]][parts[1]]

	if items == nil {
		return "<%未知项-" + s + "%>"
	} else {
		rand.Seed(uint64(time.Now().UnixNano()))
		idx := rand.Intn(len(items))
		text = items[idx][0].(string)
	}

	_ = ctx.LoadRecordFetchAndClear()
	ret, _ := DiceFormat(ctx, text)
	rs := ctx.LoadRecordFetchAndClear()
	ctx.CommandFormatInfo = append(ctx.CommandFormatInfo, &types.CommandFormatInfo{
		Key:         s,
		Text:        text,
		LoadRecords: rs,
	})
	return ret
}

func DiceFormat(ctx *types.MsgContext, s string) (string, error) { //nolint:revive
	// NOTE: 可以考虑实现一个机制，varSet的时候把变量名记在里面，这样可以传到外面再渲染
	vm := ctx.GetVM()
	vm.Ret = nil
	vm.Error = nil
	vm.Config.DisableStmts = false
	SetTempVars(ctx, "")

	s = CompatibleReplace(ctx, s)

	// 隐藏的内置字符串符号 \x1e
	// err := ctx.vm.Run("\x1e" + s + "\x1e")
	v, err := vm.RunExpr("\x1e"+s+"\x1e", true)
	if err != nil || v == nil {
		// fmt.Println("脚本执行出错V2f: ", s, "->", err)
		errText := "格式化错误:" + strconv.Quote(s)
		return errText, err
	} else {
		return v.ToString(), nil
	}
}
