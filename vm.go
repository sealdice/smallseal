package main

import (
	"fmt"
	"regexp"
	"strconv"

	ds "github.com/sealdice/dicescript"
)

type GameState struct {
	VM             *ds.Context
	SystemTemplate *GameSystemTemplateV2
	AttrsManager   *AttrsManager
}

func newGameState(gameSystem *GameSystemTemplateV2) *GameState {
	vm := ds.NewVM()
	am := NewAttrsManager()

	gameState := &GameState{
		VM:             vm,
		SystemTemplate: gameSystem,
		AttrsManager:   am,
	}

	attrs := am.Load("Group:1000", "User:1001")

	vm.Config.EnableDiceWoD = true
	vm.Config.EnableDiceCoC = true
	vm.Config.EnableDiceFate = true
	vm.Config.EnableDiceDoubleCross = true

	vm.Config.DisableStmts = false

	vm.Config.PrintBytecode = false
	vm.Config.CallbackSt = func(_type string, name string, val *ds.VMValue, extra *ds.VMValue, op string, detail string) {
		fmt.Println("st:", _type, name, val.ToString(), extra.ToString(), op, detail)
	}

	vm.Config.IgnoreDiv0 = false
	vm.Config.DefaultDiceSideExpr = "面数 ?? 50"
	vm.Config.OpCountLimit = 30000

	getAttrByName := func(nameOrigin string) *ds.VMValue {
		// 尝试变量读取
		name := nameOrigin

		// 别名转换
		if gameState.SystemTemplate != nil {
			name = gameState.SystemTemplate.GetAlias(name)
		}

		// 属性读取
		if val, ok := attrs.Load(name); ok {
			return val
		}

		// 系统默认值
		if gameState.SystemTemplate != nil {
			v2n, _, _, exists2 := gameState.SystemTemplate.GetDefaultValue(name)
			if exists2 {
				return v2n
			}
		}

		return nil
	}

	setAttrByName := func(nameOrigin string, v *ds.VMValue) {
		name := nameOrigin
		if gameState.SystemTemplate != nil {
			name = gameState.SystemTemplate.GetAlias(name)
		}
		attrs.Store(name, v)
	}

	// 主要考虑别名的问题，转一道手存入
	od := &ds.NativeObjectData{
		Name: "player",
		AttrSet: func(ctx *ds.Context, name string, v *ds.VMValue) {
			setAttrByName(name, v)
		},
		AttrGet: func(ctx *ds.Context, name string) *ds.VMValue {
			return getAttrByName(name)
		},
		ItemGet: func(ctx *ds.Context, index *ds.VMValue) *ds.VMValue {
			name2 := index.ToString()
			return getAttrByName(name2)
		},
		ItemSet: func(ctx *ds.Context, index *ds.VMValue, v *ds.VMValue) {
			setAttrByName(index.ToString(), v)
		},
		DirFunc: func(ctx *ds.Context) []*ds.VMValue {
			var keys []*ds.VMValue
			attrs.Data.Range(func(key string, value *ds.VMValue) bool {
				keys = append(keys, ds.NewStrVal(key))
				return true
			})
			return keys
		},
		// TODO: Extends: keys values
	}

	v := ds.NewNativeObjectVal(od)
	vm.StoreName("player", v, false)

	// 注册几个调试函数
	RegisterDebugFuncs(gameState)

	vm.Config.HookFuncValueLoad = func(ctx *ds.Context, name string) (string, *ds.VMValue) {
		re := regexp.MustCompile(`^(困难|极难|大成功|常规|失败|困難|極難|常規|失敗)?([^\d]+)(\d+)?$`)
		m := re.FindStringSubmatch(name)
		var cocFlagVarPrefix string

		if len(m) > 0 {
			if m[1] != "" {
				cocFlagVarPrefix = m[1]
				name = name[len(m[1]):]
			}

			// 有末值时覆盖，有初值时
			if m[3] != "" {
				v, _ := strconv.ParseInt(m[3], 10, 64)
				fmt.Println("COC值:", name, cocFlagVarPrefix)
				return name, ds.NewIntVal(ds.IntType(v))
			}
		}

		return name, nil
	}

	// _ = vm.RegCustomDice(`E(\d+)`, func(ctx *ds.Context, groups []string) *ds.VMValue {
	// 	return ds.NewIntVal(2)
	// })

	// 原生值，目前不干涉
	// vm.GlobalValueLoadFunc = func(varname string) *ds.VMValue {
	// 	nameForLoad := varname
	// 	// 别名转换
	// 	if gameState.SystemTemplate != nil {
	// 		nameForLoad = gameState.SystemTemplate.GetAlias(varname)
	// 	}

	// 	if val, ok := attrs.Load(nameForLoad); ok {
	// 		return val
	// 	}
	// 	return nil
	// }

	vm.GlobalValueLoadOverwriteFunc = func(nameOrigin string, curVal *ds.VMValue) *ds.VMValue {
		// 如果已经拿到，直接返回
		if curVal != nil {
			return curVal
		}

		// 尝试变量读取
		name := nameOrigin

		// 别名转换
		if gameState.SystemTemplate != nil {
			name = gameState.SystemTemplate.GetAlias(name)
		}

		// 属性读取
		if val, ok := attrs.Load(name); ok {
			return val
		}

		// 系统默认值
		if gameState.SystemTemplate != nil {
			v2n, _, _, exists2 := gameState.SystemTemplate.GetDefaultValue(name)
			if exists2 {
				return v2n
			}
		}

		// TODO: 规则模板中的读取拦截
		// TODO: buff机制怎么实现？

		return curVal
	}

	return gameState
}
