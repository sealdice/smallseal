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
	BuffManager    *BuffManager
}

func newGameState(gameSystem *GameSystemTemplateV2) *GameState {
	vm := ds.NewVM()
	am := NewAttrsManager()
	bm := NewBuffManager()

	gameState := &GameState{
		VM:             vm,
		SystemTemplate: gameSystem,
		AttrsManager:   am,
		BuffManager:    bm,
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
		curVal = doCompute(curVal)

		// 获取影响当前属性的所有buff
		buffs := gameState.BuffManager.GetBuffsForAttr(name)
		if len(buffs) > 0 {
			// 计算buff加成
			totalBuffValue := int64(0)
			for _, buff := range buffs {
				if buff.Add != "" {
					// 执行buff的add表达式
					result, _ := vm.RunExpr(buff.Add, false)
					if result.TypeId == ds.VMTypeInt {
						buffValue := result.MustReadInt()
						totalBuffValue += int64(buffValue)
						fmt.Printf("[Buff] %s 对 %s 增加 %d (来源: %s)\n", buff.Name, name, buffValue, buff.Source)
					}
				}
			}

			// 将buff加成应用到原始值
			if totalBuffValue != 0 && curVal.TypeId == ds.VMTypeInt {
				originalValue := curVal.MustReadInt()
				newValue := int64(originalValue) + totalBuffValue
				curVal = ds.NewIntVal(ds.IntType(newValue))
				fmt.Printf("[Buff] %s 最终值: %d (原始值 %d + buff加成 %d)\n", name, newValue, originalValue, totalBuffValue)

				if detail != nil {
					fmt.Println("???", name, detail)
					detail.Ret = curVal
					detail.Text = "buff+" + strconv.FormatInt(totalBuffValue, 10)
				}
			}
		}

		return curVal
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

		if curVal == nil {
			// 鉴于骰点环境的现实情况，设置为0
			return ds.NewIntVal(0)
		}

		return curVal
	}

	// 添加一些测试buff来验证功能
	// 模拟coc7.yaml中的buff配置
	testBuff1 := &Buff{
		ID:         "strength_boost",
		Keys:       []string{"力量"},
		KeyPattern: "",
		Name:       "力量加值",
		Add:        "d10 + 2", // 这是一个算式，载入vm执行
		Source:     "测试道具",
	}
	gameState.BuffManager.AddBuff(testBuff1)

	// 添加一个使用正则匹配的buff
	testBuff2 := &Buff{
		ID:         "physical_boost",
		Keys:       []string{},
		KeyPattern: "力量|体型", // 匹配力量或体型
		Name:       "体质强化",
		Add:        "50", // 固定加5
		Source:     "魔法药剂",
	}
	gameState.BuffManager.AddBuff(testBuff2)

	fmt.Println("[BuffManager] 已添加测试buff:")
	for _, buff := range gameState.BuffManager.ListAllBuffs() {
		fmt.Printf("  - %s: %s (影响: %v, 模式: %s, 加成: %s)\n",
			buff.ID, buff.Name, buff.Keys, buff.KeyPattern, buff.Add)
	}

	return gameState
}
