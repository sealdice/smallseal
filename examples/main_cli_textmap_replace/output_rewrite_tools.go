package main

import (
	"strings"
	"sync"
	"time"

	"github.com/sealdice/smallseal/dice"
	"github.com/sealdice/smallseal/dice/exts"
	"github.com/sealdice/smallseal/dice/types"
)

// 工具函数文件 - 提供通用的重写和克隆功能

func cleanupStaleContexts(store *sync.Map, ttl time.Duration) {
	now := time.Now()
	store.Range(func(key, value any) bool {
		createdAt, ok := value.(time.Time)
		if !ok {
			store.Delete(key)
			return true
		}
		ctx, ok := key.(*types.MsgContext)
		if !ok {
			store.Delete(key)
			return true
		}
		if ctx.CommandId == 0 && now.Sub(createdAt) > ttl {
			store.Delete(key)
		}
		return true
	})
}

func findContextByCommandID(store *sync.Map, commandID int64) *types.MsgContext {
	if commandID == 0 {
		return nil
	}

	var matched *types.MsgContext
	store.Range(func(key, value any) bool {
		ctx, ok := key.(*types.MsgContext)
		if !ok {
			store.Delete(key)
			return true
		}
		if ctx.CommandId == commandID {
			matched = ctx
			store.Delete(key)
			return false
		}
		return true
	})
	return matched
}

func rebuildWithTextMap(ctx *types.MsgContext, reply *types.MsgToReply, textMap types.TextTemplateWithWeightDict) string {
	if ctx == nil || reply == nil {
		return ""
	}

	var parts []string
	for _, info := range reply.CommandFormatInfo {
		if info == nil || info.Key == "" {
			continue
		}

		// 为每个格式化步骤创建独立的上下文副本
		ctxCopy := ctx.Copy()
		ctxCopy.TextTemplateMap = textMap
		ctxCopy.FallbackTextTemplate = dice.DefaultTextMap

		// 还原这一步的所有变量到 VM 中
		vm := ctxCopy.GetVM()
		for _, record := range info.LoadRecords {
			if record == nil || record.Key == "" || record.Val == nil {
				continue
			}
			// 临时变量直接存到 VM local
			if strings.HasPrefix(record.Key, "$t") {
				vm.StoreNameLocal(record.Key, record.Val.Clone())
			} else {
				// 非临时变量使用 exts 的标准设置方法
				exts.VarSetValue(ctxCopy, record.Key, record.Val)
			}
		}

		// 使用新的 textMap 重新渲染这个模板键
		rendered := exts.DiceFormatTmpl(ctxCopy, info.Key)
		if strings.TrimSpace(rendered) != "" {
			// 这里原本是连起来所有的 rendered 但好像之前轮的渲染信息没用，有可能只有最后一轮有用，姑且看看
			parts = append([]string{}, rendered)
		}
	}
	return strings.Join(parts, "\n")
}

func cloneTextTemplateDict(src types.TextTemplateWithWeightDict) types.TextTemplateWithWeightDict {
	dst := make(types.TextTemplateWithWeightDict, len(src))
	for cat, catMap := range src {
		if catMap == nil {
			continue
		}
		newCat := make(types.TextTemplateWithWeight, len(catMap))
		for key, items := range catMap {
			if items == nil {
				continue
			}
			newItems := make([]types.TextTemplateItem, len(items))
			for i, item := range items {
				if item == nil {
					continue
				}
				clone := make(types.TextTemplateItem, len(item))
				copy(clone, item)
				newItems[i] = clone
			}
			newCat[key] = newItems
		}
		dst[cat] = newCat
	}
	return dst
}

func ensureCategory(dict types.TextTemplateWithWeightDict, category string) types.TextTemplateWithWeight {
	cat, ok := dict[category]
	if !ok || cat == nil {
		cat = make(types.TextTemplateWithWeight)
		dict[category] = cat
	}
	return cat
}
