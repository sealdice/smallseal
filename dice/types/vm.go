package types

// 这可不妙，后续想想怎么拆这玩意

import (
	"fmt"
	"regexp"
	"strconv"

	ds "github.com/sealdice/dicescript"

	"smallseal/dice/attrs"
)

func newVM(groupId string, userId string, am *attrs.AttrsManager, gameSystem *GameSystemTemplateV2) *ds.Context {
	vm := ds.NewVM()

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

	getAttr := func() (*attrs.AttributesItem, error) {
		return am.Load(groupId, userId)
	}

	getAttrByName := func(nameOrigin string) *ds.VMValue {
		// 尝试变量读取
		name := nameOrigin

		// 别名转换
		if gameSystem != nil {
			name = gameSystem.GetAlias(name)
		}

		// 属性读取
		attr, err := getAttr()
		if err == nil {
			if val, ok := attr.LoadX(name); ok {
				return val
			}
		}

		// 系统默认值
		if gameSystem != nil {
			v2n, _, _, exists2 := gameSystem.GetDefaultValue(name)
			if exists2 {
				return v2n
			}
		}

		return nil
	}

	setAttrByName := func(nameOrigin string, v *ds.VMValue) {
		name := nameOrigin
		if gameSystem != nil {
			name = gameSystem.GetAlias(name)
		}
		attr, err := getAttr()
		if err == nil {
			attr.Store(name, v)
		}
	}

	// 主要考虑别名的问题，转一道手存入
	od := &ds.NativeObjectData{
		Name: "actor",
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
			attr, err := getAttr()
			if err == nil {
				attr.Range(func(key string, value *ds.VMValue) bool {
					keys = append(keys, ds.NewStrVal(key))
					return true
				})
			}
			return keys
		},
		// TODO: Extends: keys values
	}

	v := ds.NewNativeObjectVal(od)
	vm.StoreName("actor", v, false)

	// 注册几个调试函数
	RegisterDebugFuncs(vm)

	// 取值hack
	vm.Config.HookValueLoadPre = func(ctx *ds.Context, name string) (string, *ds.VMValue) {
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

	// 取值后hack
	vm.Config.HookValueLoadPost = func(ctx *ds.Context, name string, curVal *ds.VMValue, doCompute func(curVal *ds.VMValue) *ds.VMValue, detail *ds.BufferSpan) *ds.VMValue {
		if gameSystem.HookValueLoadPost != nil {
			curVal = gameSystem.HookValueLoadPost(ctx, name, curVal, doCompute, detail)
		} else {
			curVal = doCompute(curVal)
		}

		return curVal
	}

	// _ = vm.RegCustomDice(`E(\d+)`, func(ctx *ds.Context, groups []string) *ds.VMValue {
	// 	return ds.NewIntVal(2)
	// })

	vm.GlobalValueLoadOverwriteFunc = func(nameOrigin string, curVal *ds.VMValue) *ds.VMValue {
		// 如果已经拿到，直接返回
		if curVal != nil {
			return curVal
		}

		curVal = getAttrByName(nameOrigin)

		// TODO: 规则模板中的读取拦截

		if curVal == nil {
			// 鉴于骰点环境的现实情况，设置为0
			return ds.NewIntVal(0)
		}

		return curVal
	}

	return vm
}
