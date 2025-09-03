package types

import (
	"sync"
)

type CmdExecuteResult struct {
	Matched  bool // 是否是指令
	Solved   bool `jsbind:"solved"` // 是否响应此指令
	ShowHelp bool `jsbind:"showHelp"`
}

type CmdItemInfo struct {
	Name                    string                    `jsbind:"name"`
	ShortHelp               string                    // 短帮助，格式是 .xxx a b // 说明
	Help                    string                    `jsbind:"help"`                    // 长帮助，带换行的较详细说明
	HelpFunc                func(isShort bool) string `jsbind:"helpFunc"`                // 函数形式帮助，存在时优先于其他
	AllowDelegate           bool                      `jsbind:"allowDelegate"`           // 允许代骰
	DisabledInPrivate       bool                      `jsbind:"disabledInPrivate"`       // 私聊不可用
	EnableExecuteTimesParse bool                      `jsbind:"enableExecuteTimesParse"` // 启用执行次数解析，也就是解析3#这样的文本

	IsJsSolveFunc bool
	JSLoopVersion int64                                                                  // Loop版本号
	Solve         func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult `jsbind:"solve"`

	Raw                bool `jsbind:"raw"`                // 高级模式。默认模式下行为是：需要在当前群/私聊开启，或@自己时生效(需要为第一个@目标)
	CheckCurrentBotOn  bool `jsbind:"checkCurrentBotOn"`  // 是否检查当前可用状况，包括群内可用和是私聊两种方式，如失败不进入solve
	CheckMentionOthers bool `jsbind:"checkMentionOthers"` // 是否检查@了别的骰子，如失败不进入solve
}

type CmdMapCls map[string]*CmdItemInfo

type ExtInfo struct {
	Name    string   `jsbind:"name"    json:"name"    yaml:"name"` // 名字
	Aliases []string `jsbind:"aliases" json:"aliases" yaml:"-"`    // 别名
	Version string   `jsbind:"version" json:"version" yaml:"-"`    // 版本
	// 作者
	// 更新时间
	AutoActive      bool      `jsbind:"autoActive" json:"-" yaml:"-"` // 是否自动开启
	CmdMap          CmdMapCls `jsbind:"cmdMap"     json:"-" yaml:"-"` // 指令集合
	Brief           string    `json:"-"            yaml:"-"`
	ActiveOnPrivate bool      `json:"-"            yaml:"-"`

	// DefaultSetting *ExtDefaultSettingItem `json:"-" yaml:"-"` // 默认配置

	Author       string   `jsbind:"author" json:"-" yaml:"-"`
	ConflictWith []string `json:"-"        yaml:"-"`
	Official     bool     `json:"-"        yaml:"-"` // 官方插件

	// dice          *Dice
	IsJsExt       bool  `json:"-"`
	JSLoopVersion int64 `json:"-"`
	// Source        *JsScriptInfo `json:"-" yaml:"-"`
	// Storage *buntdb.DB `json:"-" yaml:"-"`

	// 为Storage使用互斥锁,并根据ID佬的说法修改为合适的名称
	dbMu sync.Mutex `yaml:"-"` // 互斥锁
	init bool       `yaml:"-"` // 标记Storage是否已初始化

	// 定时任务列表，用于避免 task 失去引用
	// taskList []*JsScriptTask `json:"-" yaml:"-"`

	OnNotCommandReceived func(ctx *MsgContext, msg *Message)                        `jsbind:"onNotCommandReceived" json:"-" yaml:"-"` // 指令过滤后剩下的
	OnCommandOverride    func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) bool `json:"-"                      yaml:"-"`          // 覆盖指令行为

	OnCommandReceived   func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) `jsbind:"onCommandReceived"   json:"-" yaml:"-"`
	OnMessageReceived   func(ctx *MsgContext, msg *Message)                   `jsbind:"onMessageReceived"   json:"-" yaml:"-"`
	OnMessageSend       func(ctx *MsgContext, msg *Message, flag string)      `jsbind:"onMessageSend"       json:"-" yaml:"-"`
	OnMessageDeleted    func(ctx *MsgContext, msg *Message)                   `jsbind:"onMessageDeleted"    json:"-" yaml:"-"`
	OnMessageEdit       func(ctx *MsgContext, msg *Message)                   `jsbind:"onMessageEdit"       json:"-" yaml:"-"`
	OnGroupJoined       func(ctx *MsgContext, msg *Message)                   `jsbind:"onGroupJoined"       json:"-" yaml:"-"`
	OnGroupMemberJoined func(ctx *MsgContext, msg *Message)                   `jsbind:"onGroupMemberJoined" json:"-" yaml:"-"`
	OnGuildJoined       func(ctx *MsgContext, msg *Message)                   `jsbind:"onGuildJoined"       json:"-" yaml:"-"`
	OnBecomeFriend      func(ctx *MsgContext, msg *Message)                   `jsbind:"onBecomeFriend"      json:"-" yaml:"-"`
	GetDescText         func(i *ExtInfo) string                               `jsbind:"getDescText"         json:"-" yaml:"-"`
	IsLoaded            bool                                                  `jsbind:"isLoaded"            json:"-" yaml:"-"`
	OnLoad              func()                                                `jsbind:"onLoad"              json:"-" yaml:"-"`

	// OnPoke       func(ctx *MsgContext, event *events.PokeEvent)       `jsbind:"onPoke"              json:"-" yaml:"-"` // 戳一戳
	// OnGroupLeave func(ctx *MsgContext, event *events.GroupLeaveEvent) `jsbind:"onGroupLeave"        json:"-" yaml:"-"` // 群成员被踢出
}
