package main

import (
	"fmt"

	ds "github.com/sealdice/dicescript"
)

func RegisterDebugFuncs(game *GameState) {
	vm := game.VM
	vm.StoreName("toJSON", ds.NewNativeFunctionVal(&ds.NativeFunctionData{
		Name:   "toJSON",
		Params: []string{"value"},
		NativeFunc: func(ctx *ds.Context, this *ds.VMValue, params []*ds.VMValue) *ds.VMValue {
			if len(params) > 0 {
				jsonBytes, err := params[0].ToJSON()
				if err != nil {
					ctx.Error = err
					return nil
				}
				return ds.NewStrVal(string(jsonBytes))
			}
			return nil
		},
	}), false)

	vm.StoreName("parseJSON", ds.NewNativeFunctionVal(&ds.NativeFunctionData{
		Name:   "parseJSON",
		Params: []string{"jsonStr"},
		NativeFunc: func(ctx *ds.Context, this *ds.VMValue, params []*ds.VMValue) *ds.VMValue {
			if len(params) > 0 {
				jsonStr := params[0].ToString()
				jsonBytes := []byte(jsonStr)

				v := &ds.VMValue{}
				err := v.UnmarshalJSON(jsonBytes)
				if err != nil {
					ctx.Error = err
					return nil
				}

				return v
			}
			return nil
		},
	}), false)

	vm.StoreName("configShow", ds.NewNativeFunctionVal(&ds.NativeFunctionData{
		Name:   "configShow",
		Params: []string{},
		NativeFunc: func(ctx *ds.Context, this *ds.VMValue, params []*ds.VMValue) *ds.VMValue {
			fmt.Println("当前设置:")
			c := vm.Config
			fmt.Printf("%-40s: %v - XaYmZkNqM, X个数,Y加骰线,Z面数,N阈值(>=),M阈值(<=)\n", "启用WOD语法(EnableDiceWoD)", c.EnableDiceWoD)
			fmt.Printf("%-40s: %v - bX/pX奖惩骰\n", "启用CoC语法(EnableDiceCoC)", c.EnableDiceCoC)
			fmt.Printf("%-40s: %v - fX\n", "启用Fate语法(EnableDiceFate)", c.EnableDiceFate)
			fmt.Printf("%-40s: %v - XcY\n", "启用双重十字语法(EnableDiceDoubleCross)", c.EnableDiceDoubleCross)
			fmt.Println()
			fmt.Printf("%-40s: %v - 用于st，如 &a=1d4\n", "禁用位运算(DisableBitwiseOp)", c.DisableBitwiseOp)
			fmt.Printf("%-40s: %v - 仅允许表达式,禁用if/while等\n", "禁用语句语法(DisableStmts)", c.DisableStmts)
			fmt.Printf("%-40s: %v - 只能2d6这样写,不能写2d\n", "禁用Nd语法(DisableNDice)", c.DisableNDice)
			fmt.Println()
			fmt.Printf("%-40s: %v\n", "解析算力限制", c.ParseExprLimit)
			fmt.Printf("%-40s: %v\n", "运算算力限制", c.OpCountLimit)
			fmt.Printf("%-40s: %v\n", "默认骰子面数", c.DefaultDiceSideExpr)
			fmt.Printf("%-40s: %v\n", "打印字节码", c.PrintBytecode)
			fmt.Printf("%-40s: %v\n", "忽略除零", c.IgnoreDiv0)
			fmt.Println()
			return nil
		},
	}), false)

	vm.StoreName("configSet", ds.NewNativeFunctionVal(&ds.NativeFunctionData{
		Name:   "configSet",
		Params: []string{"key", "value"},
		NativeFunc: func(ctx *ds.Context, this *ds.VMValue, params []*ds.VMValue) *ds.VMValue {
			if len(params) < 2 {
				fmt.Println("用法: configSet(key, value)")
				return nil
			}

			key := params[0].ToString()
			value := params[1]
			c := &vm.Config

			switch key {
			case "EnableDiceWoD":
				c.EnableDiceWoD = value.AsBool()
				fmt.Printf("设置 %s = %v\n", key, c.EnableDiceWoD)
				return ds.NewIntVal(1)
			case "EnableDiceCoC":
				c.EnableDiceCoC = value.AsBool()
				fmt.Printf("设置 %s = %v\n", key, c.EnableDiceCoC)
				return ds.NewIntVal(1)
			case "EnableDiceFate":
				c.EnableDiceFate = value.AsBool()
				fmt.Printf("设置 %s = %v\n", key, c.EnableDiceFate)
				return ds.NewIntVal(1)
			case "EnableDiceDoubleCross":
				c.EnableDiceDoubleCross = value.AsBool()
				fmt.Printf("设置 %s = %v\n", key, c.EnableDiceDoubleCross)
				return ds.NewIntVal(1)
			case "DisableBitwiseOp":
				c.DisableBitwiseOp = value.AsBool()
				fmt.Printf("设置 %s = %v\n", key, c.DisableBitwiseOp)
				return ds.NewIntVal(1)
			case "DisableStmts":
				c.DisableStmts = value.AsBool()
				fmt.Printf("设置 %s = %v\n", key, c.DisableStmts)
				return ds.NewIntVal(1)
			case "DisableNDice":
				c.DisableNDice = value.AsBool()
				fmt.Printf("设置 %s = %v\n", key, c.DisableNDice)
				return ds.NewIntVal(1)
			case "ParseExprLimit":
				c.ParseExprLimit = uint64(value.MustReadInt())
				fmt.Printf("设置 %s = %v\n", key, c.ParseExprLimit)
				return ds.NewIntVal(1)
			case "OpCountLimit":
				c.OpCountLimit = value.MustReadInt()
				fmt.Printf("设置 %s = %v\n", key, c.OpCountLimit)
				return ds.NewIntVal(1)
			case "DefaultDiceSideExpr":
				c.DefaultDiceSideExpr = value.ToString()
				fmt.Printf("设置 %s = %s\n", key, c.DefaultDiceSideExpr)
				return ds.NewIntVal(1)
			case "PrintBytecode":
				c.PrintBytecode = value.AsBool()
				fmt.Printf("设置 %s = %v\n", key, c.PrintBytecode)
				return ds.NewIntVal(1)
			case "IgnoreDiv0":
				c.IgnoreDiv0 = value.AsBool()
				fmt.Printf("设置 %s = %v\n", key, c.IgnoreDiv0)
				return ds.NewIntVal(1)
			default:
				fmt.Printf("未知配置项: %s\n", key)
				return nil
			}
		},
	}), false)

}
