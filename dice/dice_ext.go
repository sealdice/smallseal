package dice

import (
	"fmt"
	"slices"
	"strings"

	"github.com/sealdice/smallseal/dice/types"
)

func (d *Dice) RegisterExtension(extInfo *types.ExtInfo) {
	for _, name := range append(extInfo.Aliases, extInfo.Name) {
		if collide := d.ExtFind(name, false); collide != nil {
			panicMsg := fmt.Sprintf("扩展<%s>的名字%q与现存扩展<%s>冲突", extInfo.Name, name, collide.Name)
			panic(panicMsg)
		}
	}

	// extInfo.dice = d
	d.ExtList = append(d.ExtList, extInfo)
}

// ExtFind 根据名称或别名查找扩展
func (d *Dice) ExtFind(s string, fromJS bool) *types.ExtInfo {
	find := func(_ string) *types.ExtInfo {
		for _, i := range d.ExtList {
			// 名字匹配，优先级最高
			if i.Name == s {
				return i
			}
		}
		for _, i := range d.ExtList {
			// 别名匹配，优先级次之
			if slices.Contains(i.Aliases, s) {
				return i
			}
		}
		for _, i := range d.ExtList {
			// 忽略大小写匹配，优先级最低
			if strings.EqualFold(i.Name, s) || slices.Contains(i.Aliases, strings.ToLower(s)) {
				return i
			}
		}
		return nil
	}
	ext := find(s)
	if ext != nil && ext.Official && fromJS {
		// return a copy of the official extension
		cmdMap := make(types.CmdMapCls, len(ext.CmdMap))
		for s2, info := range ext.CmdMap {
			cmdMap[s2] = &types.CmdItemInfo{
				Name:                    info.Name,
				ShortHelp:               info.ShortHelp,
				Help:                    info.Help,
				HelpFunc:                info.HelpFunc,
				AllowDelegate:           info.AllowDelegate,
				DisabledInPrivate:       info.DisabledInPrivate,
				EnableExecuteTimesParse: info.EnableExecuteTimesParse,
				IsJsSolveFunc:           info.IsJsSolveFunc,
				Solve:                   info.Solve,
				Raw:                     info.Raw,
				CheckCurrentBotOn:       info.CheckCurrentBotOn,
				CheckMentionOthers:      info.CheckMentionOthers,
			}
		}
		return &types.ExtInfo{
			Name:       ext.Name,
			Aliases:    ext.Aliases,
			Author:     ext.Author,
			Version:    ext.Version,
			AutoActive: ext.AutoActive,
			CmdMap:     cmdMap,
			Brief:      ext.Brief,
			Official:   ext.Official,
		}
	}
	return ext
}

func (d *Dice) GetExtList() []*types.ExtInfo {
	return d.ExtList
}
