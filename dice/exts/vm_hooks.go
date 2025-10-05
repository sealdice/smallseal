package exts

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/lascape/sat"
	ds "github.com/sealdice/dicescript"
	"github.com/sealdice/smallseal/dice/types"
)

var (
	cocPrefixPattern = regexp.MustCompile(`^(困难|极难|大成功|常规|失败|困難|極難|常規|失敗)?([^\d]+)(\d+)?$`)
	cocPrefixDict    sat.Dicter
	cocPrefixOnce    sync.Once
)

func setCocPrefixReadForVM(ctx *MsgContext, cb func(string)) {
	vm := ctx.GetVM()
	if vm == nil {
		return
	}

	vm.Config.HookValueLoadPre = func(vmCtx *ds.Context, name string) (string, *ds.VMValue) {
		matches := cocPrefixPattern.FindStringSubmatch(name)
		if len(matches) > 0 {
			prefix := matches[1]
			if prefix != "" {
				if cb != nil {
					cb(normalizePrefix(prefix))
				}
				name = name[len(prefix):]
			}

			if matches[3] != "" && !strings.HasPrefix(name, "$") {
				if v, err := strconv.ParseInt(matches[3], 10, 64); err == nil {
					return name, ds.NewIntVal(ds.IntType(v))
				}
			}
		}

		return name, nil
	}
}

func normalizePrefix(prefix string) string {
	dict := getCocPrefixDict()
	if dict == nil {
		return prefix
	}
	if converted := dict.Read(prefix); converted != "" {
		return converted
	}
	return prefix
}

func getCocPrefixDict() sat.Dicter {
	cocPrefixOnce.Do(func() {
		cocPrefixDict = sat.DefaultDict()
	})
	return cocPrefixDict
}

func tryMergeBuff(ctx *MsgContext, vm *ds.Context, varname string, curVal *ds.VMValue, computedOnly bool, detail *ds.BufferSpan) (*ds.VMValue, bool) {
	if ctx == nil || ctx.AttrsManager == nil || ctx.Group == nil || ctx.Player == nil || curVal == nil {
		return curVal, false
	}

	attrs, err := ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId)
	if err != nil || attrs == nil {
		return curVal, false
	}

	buffName := "$buff_" + varname
	buffVal, exists := attrs.Load(buffName)
	if !exists || buffVal == nil {
		return curVal, false
	}

	if computedOnly {
		if curVal.TypeId == ds.VMTypeComputedValue && buffVal.TypeId == ds.VMTypeComputedValue {
			data, err := curVal.ToJSON()
			if err != nil {
				return curVal, false
			}
			newVal := curVal.Clone()
			if err := newVal.UnmarshalJSON(data); err != nil {
				return curVal, false
			}
			cdCur, okCur := newVal.ReadComputed()
			cdBuff, okBuff := buffVal.ReadComputed()
			if !okCur || !okBuff {
				return curVal, false
			}

			cdBuff.Attrs.Range(func(key string, value *ds.VMValue) bool {
				if existing, ok := cdCur.Attrs.Load(key); ok {
					if merged := existing.OpAdd(vm, value); merged != nil {
						vm.Error = nil
						cdCur.Attrs.Store(key, merged)
					} else {
						vm.Error = nil
					}
				} else {
					cdCur.Attrs.Store(key, value.Clone())
				}
				return true
			})

			return newVal, true
		}
		return curVal, false
	}

	if detail != nil {
		detail.Text += fmt.Sprintf("%s+buff%s", curVal.ToString(), buffVal.ToString())
	}

	merged := curVal.OpAdd(vm, buffVal)
	if merged != nil {
		vm.Error = nil
		if detail != nil {
			detail.Ret = merged
		}
		return merged, true
	}
	vm.Error = nil
	return curVal, false
}

func setDndReadForVM(ctx *MsgContext, rcMode bool) {
	if ctx == nil {
		return
	}

	vm := ctx.GetVM()
	if vm == nil {
		return
	}

	var skip bool
	loadBuff := true

	vm.Config.HookValueLoadPost = func(inner *ds.Context, varname string, curVal *ds.VMValue, doCompute func(*ds.VMValue) *ds.VMValue, detail *ds.BufferSpan) *ds.VMValue {
		tmpl := ctx.GameSystem
		if tmpl == nil && ctx.Group != nil {
			tmpl = ctx.GetCharTemplate()
		}

		working := curVal
		if loadBuff && working != nil {
			working, _ = tryMergeBuff(ctx, inner, varname, working, true, detail)
		}

		doComputeWrapped := func(val *ds.VMValue) *ds.VMValue {
			result := doCompute(val)
			if loadBuff && result != nil {
				result, _ = tryMergeBuff(ctx, inner, varname, result, false, detail)
			}
			return result
		}

		var value *ds.VMValue
		if tmpl != nil && tmpl.HookValueLoadPost != nil {
			value = tmpl.HookValueLoadPost(inner, varname, working, doComputeWrapped, detail)
		} else {
			value = doComputeWrapped(working)
		}

		if value == nil {
			ctx.LoadRecords = append(ctx.LoadRecords, &types.LoadRecord{Key: varname, Val: value})
			return nil
		}

		if !skip && rcMode {
			if isAbilityScores(varname) && inner.Depth() == 0 && inner.UpCtx == nil && value.TypeId == ds.VMTypeInt {
				mod := value.MustReadInt()/2 - 5
				v := ds.NewIntVal(mod)
				if detail != nil {
					detail.Tag = "dnd-rc"
					detail.Text = fmt.Sprintf("%s调整值%d", varname, mod)
					detail.Ret = v
				}
				value = v
			} else if tmpl != nil {
				realVarName := tmpl.GetAlias(varname)
				if parent := dndAttrParent[realVarName]; parent != "" && value.TypeId == ds.VMTypeInt {
					base, err := tmpl.GetRealValue(ctx, parent)
					if err == nil && base != nil && base.TypeId == ds.VMTypeInt {
						mod := base.MustReadInt()/2 - 5
						if detail != nil {
							detail.Tag = "dnd-rc"
							detail.Text = fmt.Sprintf("%s调整值%d", parent, mod)
						}

						v := value.MustReadInt() - mod
						expr := fmt.Sprintf("floor(&%s.factor * 熟练)", varname)
						skip = true
						if ret2, _ := inner.RunExpr(expr, false); ret2 != nil {
							if detail != nil {
								detail.Text += fmt.Sprintf("+熟练%s", ret2.ToString())
							}
							if ret2.TypeId == ds.VMTypeInt {
								v -= ret2.MustReadInt()
							}
						}
						inner.Error = nil
						skip = false
						if detail != nil {
							detail.Text += fmt.Sprintf("+%s%d", varname, v)
							detail.Ret = value
						}
					}
				}
			}
		}

		ctx.LoadRecords = append(ctx.LoadRecords, &types.LoadRecord{Key: varname, Val: value})
		return value
	}
}
