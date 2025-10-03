package types

import (
	"strings"

	"github.com/sealdice/dicescript"
	"golang.org/x/time/rate"

	"github.com/sealdice/smallseal/utils"
)

// GroupPlayerInfoBase 群内玩家信息
type GroupPlayerInfo struct {
	// 补充这个字段，从而保证包含主键ID
	ID     uint   `gorm:"column:id;primaryKey;autoIncrement"                                                               jsbind:"-"      yaml:"-"`    // 主键ID字段，自增
	Name   string `gorm:"column:name"                                                                                      jsbind:"name"   yaml:"name"` // 玩家昵称
	UserId string `gorm:"column:user_id;index:idx_group_player_info_user_id; uniqueIndex:idx_group_player_info_group_user" jsbind:"userId" yaml:"userId"`
	// 非数据库信息：是否在群内
	InGroup         bool  `gorm:"-"                        yaml:"inGroup"`                                  // 是否在群内，有时一个人走了，信息还暂时残留
	LastCommandTime int64 `gorm:"column:last_command_time" jsbind:"lastCommandTime" yaml:"lastCommandTime"` // 上次发送指令时间
	// 非数据库信息
	RateLimiter *rate.Limiter `gorm:"-" yaml:"-"` // 限速器
	// 非数据库信息
	RateLimitWarned     bool   `gorm:"-"                             yaml:"-"`                                                // 是否已经警告过限速
	AutoSetNameTemplate string `gorm:"column:auto_set_name_template" jsbind:"autoSetNameTemplate" yaml:"autoSetNameTemplate"` // 名片模板

	// level int 权限
	DiceSideExpr string `gorm:"column:dice_side_expr" yaml:"diceSideExpr"` // 面数表达式，为空时等同于d100
	// 非数据库信息
	ValueMapTemp *dicescript.ValueMap `gorm:"-" yaml:"-"` // 玩家的群内临时变量
	// ValueMapTemp map[string]*VMValue  `yaml:"-"`           // 玩家的群内临时变量

	// 非数据库信息
	// TempValueAlias *map[string][]string `gorm:"-" yaml:"-"` // 群内临时变量别名 - 其实这个有点怪的，为什么在这里？

	// 非数据库信息
	UpdatedAtTime int64 `gorm:"-" json:"-" yaml:"-"`
	// 非数据库信息
	RecentUsedTime int64 `gorm:"-" json:"-" yaml:"-"`
	// 缺少信息 -> 这边原来就是int吗？
	CreatedAt int    `gorm:"column:created_at"                                                                                  json:"-" yaml:"-"` // 创建时间
	UpdatedAt int    `gorm:"column:updated_at"                                                                                  json:"-" yaml:"-"` // 更新时间
	GroupID   string `gorm:"column:group_id;index:idx_group_player_info_group_id; uniqueIndex:idx_group_player_info_group_user" json:"-" yaml:"-"`
}

type GroupInfo struct {
	Active           bool                                     `jsbind:"active"         json:"active"                yaml:"active"` // 是否在群内开启 - 过渡为象征意义
	ActivatedExtList []*ExtInfo                               `json:"activatedExtList" yaml:"activatedExtList,flow"`               // 当前群开启的扩展列表
	ExtListSnapshot  []string                                 `json:"-"                yaml:"-"`                                   // 存放当前激活的扩展表，无论其是否存在，用于处理插件重载后优先级混乱的问题
	ExtActiveStates  *utils.SyncMap[string, bool]             `json:"extActiveStates"  yaml:"extActiveStates,flow"`                // 扩展激活状态映射表，key为扩展名，value为是否激活
	Players          *utils.SyncMap[string, *GroupPlayerInfo] `json:"-"                yaml:"-"`                                   // 群员角色数据

	GroupId   string `jsbind:"groupId"       json:"groupId"      yaml:"groupId"`
	GuildID   string `jsbind:"guildId"       json:"guildId"      yaml:"guildId"`
	ChannelID string `jsbind:"channelId"     json:"channelId"    yaml:"channelId"`
	GroupName string `jsbind:"groupName"     json:"groupName"    yaml:"groupName"`

	DiceIDActiveMap *utils.SyncMap[string, bool] `json:"diceIdActiveMap" yaml:"diceIds,flow"` // 对应的骰子ID(格式 平台:ID)，对应单骰多号情况，例如骰A B都加了群Z，A退群不会影响B在群内服务
	DiceIDExistsMap *utils.SyncMap[string, bool] `json:"diceIdExistsMap" yaml:"-"`            // 对应的骰子ID(格式 平台:ID)是否存在于群内
	BotList         *utils.SyncMap[string, bool] `json:"botList"         yaml:"botList,flow"` // 其他骰子列表
	DiceSideExpr    string                       `json:"diceSideExpr"    yaml:"diceSideExpr"` //
	System          string                       `json:"system"          yaml:"system"`       // 规则系统，概念同bcdice的gamesystem，距离如dnd5e coc7
	// DiceSideNum     int64                        `json:"diceSideNum"     yaml:"diceSideNum"`  // 以后可能会支持 1d4 这种默认面数，暂不开放给js

	HelpPackages []string `json:"helpPackages"   yaml:"-"`
	CocRuleIndex int      `jsbind:"cocRuleIndex" json:"cocRuleIndex" yaml:"cocRuleIndex"`
	LogCurName   string   `jsbind:"logCurName"   json:"logCurName"   yaml:"logCurFile"`
	LogOn        bool     `jsbind:"logOn"        json:"logOn"        yaml:"logOn"`

	QuitMarkAutoClean   bool   `json:"-"                     yaml:"-"` // 自动清群 - 播报，即将自动退出群组
	QuitMarkMaster      bool   `json:"-"                     yaml:"-"` // 骰主命令退群 - 播报，即将自动退出群组
	RecentDiceSendTime  int64  `jsbind:"recentDiceSendTime"  json:"recentDiceSendTime"`
	ShowGroupWelcome    bool   `jsbind:"showGroupWelcome"    json:"showGroupWelcome"    yaml:"showGroupWelcome"` // 是否迎新
	GroupWelcomeMessage string `jsbind:"groupWelcomeMessage" json:"groupWelcomeMessage" yaml:"groupWelcomeMessage"`
	// FirstSpeechMade     bool   `yaml:"firstSpeechMade"` // 是否做过进群发言
	LastCustomReplyTime float64 `json:"-" yaml:"-"` // 上次自定义回复时间

	RateLimiter     *rate.Limiter `json:"-" yaml:"-"`
	RateLimitWarned bool          `json:"-" yaml:"-"`

	EnteredTime  int64  `jsbind:"enteredTime"  json:"enteredTime"  yaml:"enteredTime"`  // 入群时间
	InviteUserID string `jsbind:"inviteUserId" json:"inviteUserId" yaml:"inviteUserId"` // 邀请人
	// 仅用于http接口
	TmpPlayerNum int64    `json:"tmpPlayerNum" yaml:"-"`
	TmpExtList   []string `json:"tmpExtList"   yaml:"-"`

	UpdatedAtTime int64 `json:"-" yaml:"-"`

	DefaultHelpGroup string `json:"defaultHelpGroup" yaml:"defaultHelpGroup"` // 当前群默认的帮助文档分组

	PlayerGroups *utils.SyncMap[string, []string] `json:"playerGroups" yaml:"playerGroups"` // 供team指令使用并由其管理，与Players不同步
}

// IsExtensionActive 检查指定扩展是否在当前群组中激活
func (g *GroupInfo) IsExtensionActive(extName string) bool {
	if g.ExtActiveStates == nil {
		return false
	}
	active, exists := g.ExtActiveStates.Load(extName)
	return exists && active
}

// SetExtensionActive 设置指定扩展在当前群组中的激活状态
func (g *GroupInfo) SetExtensionActive(extName string, active bool) {
	if g.ExtActiveStates == nil {
		g.ExtActiveStates = &utils.SyncMap[string, bool]{}
	}
	g.ExtActiveStates.Store(extName, active)
}

// GetActiveExtensions 获取当前群组中所有激活的扩展列表
func (g *GroupInfo) GetActiveExtensions(allExtensions []*ExtInfo) []*ExtInfo {
	var activeExts []*ExtInfo
	for _, ext := range allExtensions {
		// 检查扩展是否在激活状态映射表中被明确设置
		if g.ExtActiveStates != nil {
			if active, exists := g.ExtActiveStates.Load(ext.Name); exists {
				if active {
					activeExts = append(activeExts, ext)
				}
				continue
			}
		}
		// 如果没有明确设置，则使用扩展的AutoActive属性
		if ext.AutoActive {
			activeExts = append(activeExts, ext)
			// 同时将其状态记录到映射表中
			g.SetExtensionActive(ext.Name, true)
		}
	}
	return activeExts
}

func (g *GroupInfo) ensureExtStates() {
	if g.ExtActiveStates == nil {
		g.ExtActiveStates = &utils.SyncMap[string, bool]{}
	}
	if g.ActivatedExtList == nil {
		g.ActivatedExtList = []*ExtInfo{}
	}
}

// ExtActive 标记扩展为启用状态，并维护 ActivatedExtList
func (g *GroupInfo) ExtActive(ext *ExtInfo) {
	if ext == nil {
		return
	}
	g.ensureExtStates()
	g.ExtActiveStates.Store(ext.Name, true)
	found := false
	for idx, existing := range g.ActivatedExtList {
		if existing != nil && existing.Name == ext.Name {
			g.ActivatedExtList[idx] = ext
			found = true
			break
		}
	}
	if !found {
		g.ActivatedExtList = append(g.ActivatedExtList, ext)
	}
}

// ExtInactiveByName 关闭指定扩展，并返回被关闭的扩展信息
func (g *GroupInfo) ExtInactiveByName(name string) *ExtInfo {
	if name == "" {
		return nil
	}
	g.ensureExtStates()
	g.ExtActiveStates.Store(name, false)
	var removed *ExtInfo
	newList := make([]*ExtInfo, 0, len(g.ActivatedExtList))
	for _, ext := range g.ActivatedExtList {
		if ext == nil {
			continue
		}
		if strings.EqualFold(ext.Name, name) {
			removed = ext
			continue
		}
		newList = append(newList, ext)
	}
	g.ActivatedExtList = newList
	return removed
}

// ExtGetActive 返回已启用的扩展信息
func (g *GroupInfo) ExtGetActive(name string) *ExtInfo {
	for _, ext := range g.ActivatedExtList {
		if ext != nil && strings.EqualFold(ext.Name, name) {
			return ext
		}
	}
	return nil
}

// EnsureBotList 确保机器人列表被初始化
func (g *GroupInfo) EnsureBotList() {
	if g.BotList == nil {
		g.BotList = &utils.SyncMap[string, bool]{}
	}
}
