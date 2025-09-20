package dice

import (
	"smallseal/dice/types"
	"smallseal/utils"
)

// GroupInfoManager 群组信息管理器接口
type GroupInfoManager interface {
	// Load 加载群组信息
	Load(groupId string) (*types.GroupInfo, bool)
	// Store 存储群组信息
	Store(groupId string, groupInfo *types.GroupInfo)
	// Delete 删除群组信息
	Delete(groupId string)
}

// DefaultGroupInfoManager 默认的群组信息管理器实现
type DefaultGroupInfoManager struct {
	groupMap *utils.SyncMap[string, *types.GroupInfo]
}

// NewDefaultGroupInfoManager 创建默认的群组信息管理器
func NewDefaultGroupInfoManager() *DefaultGroupInfoManager {
	return &DefaultGroupInfoManager{
		groupMap: &utils.SyncMap[string, *types.GroupInfo]{},
	}
}

// Load 加载群组信息
func (m *DefaultGroupInfoManager) Load(groupId string) (*types.GroupInfo, bool) {
	return m.groupMap.Load(groupId)
}

// Store 存储群组信息
func (m *DefaultGroupInfoManager) Store(groupId string, groupInfo *types.GroupInfo) {
	m.groupMap.Store(groupId, groupInfo)
}

// Delete 删除群组信息
func (m *DefaultGroupInfoManager) Delete(groupId string) {
	m.groupMap.Delete(groupId)
}
