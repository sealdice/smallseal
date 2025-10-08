package types

// 这可不妙，后续想想怎么拆这玩意

import (
	"bytes"
	"regexp"
	"sort"
	"strings"
	"time"

	"golang.org/x/exp/rand"

	"github.com/samber/lo"
	ds "github.com/sealdice/dicescript"

	"github.com/sealdice/smallseal/dice/attrs"
)

type spanByEnd []ds.BufferSpan

func (a spanByEnd) Len() int           { return len(a) }
func (a spanByEnd) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a spanByEnd) Less(i, j int) bool { return a[i].End < a[j].End }

func newVM(mctx *MsgContext) *ds.Context {
	// 搞了半天，终于还是把ctx传进来了
	vm := ds.NewVM()

	groupId := mctx.Group.GroupId
	userId := mctx.Player.UserId
	am := mctx.AttrsManager
	gameSystem := mctx.GameSystem
	textTemplate := mctx.TextTemplateMap
	defaultTextTemplate := mctx.FallbackTextTemplate

	vm.Config.EnableDiceWoD = true
	vm.Config.EnableDiceCoC = true
	vm.Config.EnableDiceFate = true
	vm.Config.EnableDiceDoubleCross = true

	vm.Config.DisableStmts = false

	vm.Config.PrintBytecode = false
	vm.Config.CallbackSt = func(_type string, name string, val *ds.VMValue, extra *ds.VMValue, op string, detail string) {
		// fmt.Println("st:", _type, name, val.ToString(), extra.ToString(), op, detail)
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
			if val, ok := attr.Load(name); ok {
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

	// 取值后hack
	vm.Config.HookValueLoadPost = func(ctx *ds.Context, name string, curVal *ds.VMValue, doCompute func(curVal *ds.VMValue) *ds.VMValue, detail *ds.BufferSpan) *ds.VMValue {
		if gameSystem.HookValueLoadPost != nil {
			curVal = gameSystem.HookValueLoadPost(ctx, name, curVal, doCompute, detail)
		} else {
			curVal = doCompute(curVal)
		}

		mctx.LoadRecords = append(mctx.LoadRecords, &LoadRecord{
			Key: name,
			Val: curVal,
		})
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

		// 检查textMap变量，但这个可能与format有点重复，有些犹豫
		if curVal == nil {
			parts := strings.SplitN(nameOrigin, ":", 2)
			if len(parts) == 2 {
				var items []TextTemplateItem
				if cat, ok := textTemplate[parts[0]]; ok {
					items = cat[parts[1]]
				}
				if len(items) == 0 && defaultTextTemplate != nil {
					if cat, ok := defaultTextTemplate[parts[0]]; ok {
						items = cat[parts[1]]
					}
				}
				if len(items) > 0 {
					rand.Seed(uint64(time.Now().UnixNano()))
					idx := rand.Intn(len(items))
					text := items[idx][0].(string)
					curVal = ds.NewComputedVal("\x1e" + text + "\x1e")
				}
			}
		}

		if curVal == nil {
			// 鉴于骰点环境的现实情况，设置为0
			return ds.NewIntVal(0)
		}

		return curVal
	}

	reSimpleBP := regexp.MustCompile(`^[bpBP]\d*$`)

	vm.Config.CustomMakeDetailFunc = func(ctx *ds.Context, details []ds.BufferSpan, dataBuffer []byte, parsedOffset int) string {
		detailResult := dataBuffer[:parsedOffset]

		// 特殊机制: 从模板读取detail进行覆盖
		for index, i := range details {
			// && ctx.UpCtx == nil
			if (i.Tag == "load" || i.Tag == "load.computed") && mctx.GameSystem != nil {
				expr := strings.TrimSpace(string(detailResult[i.Begin:i.End]))
				detailExpr, exists := mctx.GameSystem.Attrs.DetailOverwrite[expr]
				if !exists {
					// 如果没有，尝试使用通配
					detailExpr = mctx.GameSystem.Attrs.DetailOverwrite["*"]
					if detailExpr != "" {
						// key 应该是等于expr的
						ctx.StoreNameLocal("name", ds.NewStrVal(expr))
					}
				}
				skip := false
				if detailExpr != "" {
					v, err := ctx.RunExpr(detailExpr, true)
					if v != nil {
						details[index].Text = v.ToString()
					}
					if err != nil {
						details[index].Text = err.Error()
					}
					if v == nil || v.TypeId == ds.VMTypeNull {
						skip = true
					}
				}

				if !skip {
					// 如果存在且为空，那么很明显意图就是置空
					if exists && detailExpr == "" {
						details[index].Text = ""
					}
				}
			}
		}

		var curPoint ds.IntType
		lastEnd := ds.IntType(-1) //nolint:ineffassign

		type Group struct {
			begin ds.IntType
			end   ds.IntType
			tag   string
			spans []ds.BufferSpan
			val   *ds.VMValue
		}

		var m []Group
		for _, i := range details {
			// fmt.Println("?", i, lastEnd)
			if i.Begin > lastEnd {
				curPoint = i.Begin
				m = append(m, Group{begin: curPoint, end: i.End, tag: i.Tag, spans: []ds.BufferSpan{i}, val: i.Ret})
			} else {
				m[len(m)-1].spans = append(m[len(m)-1].spans, i)
				if i.End > m[len(m)-1].end {
					m[len(m)-1].end = i.End
				}
			}

			if i.End > lastEnd {
				lastEnd = i.End
			}
		}

		var detailArr []*ds.VMValue
		for i := len(m) - 1; i >= 0; i-- {
			buf := bytes.Buffer{}
			writeBuf := func(p []byte) {
				buf.Write(p)
			}
			writeBufStr := func(s string) {
				buf.WriteString(s)
			}

			item := m[i]
			size := len(item.spans)
			sort.Sort(spanByEnd(item.spans))
			last := item.spans[size-1]

			subDetailsText := ""
			if size > 1 {
				// 次级结果，如 (10d3)d5 中，此处为10d3的结果
				// 例如 (10d3)d5=63[(10d3)d5=...,10d3=19]
				for j := range len(item.spans) - 1 {
					span := item.spans[j]
					subDetailsText += "," + string(detailResult[span.Begin:span.End]) + "=" + span.Ret.ToString()
				}
			}

			exprText := last.Expr
			baseExprText := string(detailResult[item.begin:item.end])
			if last.Expr == "" {
				exprText = baseExprText
			}

			writeBuf(detailResult[:item.begin])

			// 主体结果部分，如 (10d3)d5=63[(10d3)d5=2+2+2+5+2+5+5+4+1+3+4+1+4+5+4+3+4+5+2,10d3=19]
			partRet := last.Ret.ToString()

			detail := "[" + exprText
			if last.Text != "" && partRet != last.Text { // 规则1.1
				detail += "=" + last.Text
			}

			switch item.tag {
			case "dnd-rc":
				detail = "[" + last.Text
			case "load":
				detail = "[" + exprText
				if last.Text != "" {
					detail += "," + last.Text
				}
			case "dice-coc-bonus", "dice-coc-penalty":
				// 对简单式子进行结果简化，未来或许可以做成通配规则(给左式加个规则进行消除)
				if reSimpleBP.MatchString(exprText) {
					detail = "[" + last.Text[1:len(last.Text)-1]
				}
			case "load.computed":
				detail += "=" + partRet
			}

			detail += subDetailsText + "]"
			if len(m) == 1 && detail == "["+baseExprText+"]" {
				detail = "" // 规则1.3
			}
			if len(detail) > 400 {
				detail = "[略]"
			}
			writeBufStr(partRet + detail)
			writeBuf(detailResult[item.end:])
			detailResult = buf.Bytes()

			d := ds.NewDictValWithArrayMust(
				ds.NewStrVal("tag"), ds.NewStrVal(item.tag),
				ds.NewStrVal("expr"), ds.NewStrVal(exprText),
				ds.NewStrVal("val"), item.val,
			)
			detailArr = append(detailArr, d.V())
		}

		// TODO: 此时加了TrimSpace表现正常，但深层原因是ds在处理"d3 x"这个表达式时多吃了一个空格，修复后取消trim
		detailStr := strings.TrimSpace(string(detailResult))
		if detailStr == ctx.Ret.ToString() {
			detailStr = "" // 如果detail和结果值完全一致，那么将其置空
		}
		ctx.StoreNameLocal("details", ds.NewArrayValRaw(lo.Reverse(detailArr))) //nolint:staticcheck // old code
		return detailStr
	}

	return vm
}
