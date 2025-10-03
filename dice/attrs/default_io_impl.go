package attrs

import (
	"errors"
	"sync"
	"time"

	ds "github.com/sealdice/dicescript"
)

// MemoryAttrsIO 基于内存的AttrsIO实现
type MemoryAttrsIO struct {
	mu       sync.RWMutex
	items    map[string]*AttributesItem   // 存储所有属性项，key为ID
	bindings map[string]map[string]string // 存储绑定关系，key为groupId，value为map[userId]attrsId
}

// NewMemoryAttrsIO 创建新的内存AttrsIO实例
func NewMemoryAttrsIO() *MemoryAttrsIO {
	return &MemoryAttrsIO{
		items:    make(map[string]*AttributesItem),
		bindings: make(map[string]map[string]string),
	}
}

// GetById 根据ID获取角色
func (m *MemoryAttrsIO) GetById(id string) (*AttributesItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, exists := m.items[id]
	if !exists {
		return nil, errors.New("item not found")
	}

	// 更新最后使用时间
	item.LastUsedTime = time.Now().Unix()
	return item, nil
}

// Puts 批量更新插入角色
func (m *MemoryAttrsIO) Puts(items []*AttrsUpsertParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, param := range items {
		if param.Id == "" {
			return errors.New("id cannot be empty")
		}

		// 检查是否已存在
		existingItem, exists := m.items[param.Id]
		if exists {
			// 更新现有项
			existingItem.Data = param.Data
			existingItem.Name = param.Name
			existingItem.SheetType = param.SheetType
			existingItem.OwnerId = param.OwnerId
			existingItem.AttrsType = param.AttrsType
			existingItem.IsHidden = param.IsHidden
			existingItem.LastModifiedTime = time.Now().Unix()
			existingItem.IsSaved = false
			if len(param.Data) > 0 {
				if v, err := ds.VMValueFromJSON(param.Data); err == nil {
					if dict, ok := v.ReadDictData(); ok {
						existingItem.valueMap = dict.Dict
					}
				}
			}
		} else {
			// 创建新项
			newItem := &AttributesItem{
				ID:               param.Id,
				Data:             param.Data,
				Name:             param.Name,
				SheetType:        param.SheetType,
				OwnerId:          param.OwnerId,
				AttrsType:        param.AttrsType,
				IsHidden:         param.IsHidden,
				LastModifiedTime: time.Now().Unix(),
				LastUsedTime:     time.Now().Unix(),
				valueMap:         &ds.ValueMap{},
				IsSaved:          false,
			}
			if len(param.Data) > 0 {
				if v, err := ds.VMValueFromJSON(param.Data); err == nil {
					if dict, ok := v.ReadDictData(); ok {
						newItem.valueMap = dict.Dict
					}
				}
			}
			m.items[param.Id] = newItem
		}
	}

	return nil
}

// DeleteById 根据ID删除角色
func (m *MemoryAttrsIO) DeleteById(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.items[id]; !exists {
		return errors.New("item not found")
	}

	delete(m.items, id)

	// 同时清理所有相关的绑定
	m.unbindAllInternal(id)

	return nil
}

// ListByUid 根据用户ID获取其角色列表
func (m *MemoryAttrsIO) ListByUid(userId string) ([]*AttributesItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*AttributesItem
	for _, item := range m.items {
		if item.OwnerId == userId {
			result = append(result, item)
		}
	}

	return result, nil
}

// GetByUidAndName 根据uid和卡名获取角色
func (m *MemoryAttrsIO) GetByUidAndName(userId string, name string) (*AttributesItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, item := range m.items {
		if item.OwnerId == userId && item.Name == name {
			item.LastUsedTime = time.Now().Unix()
			return item, nil
		}
	}

	return nil, nil // 不存在时返回nil
}

// BindingIdGet 获取群组绑定的卡片ID
func (m *MemoryAttrsIO) BindingIdGet(groupId string, userId string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	groupBindings, exists := m.bindings[groupId]
	if !exists {
		return "", nil
	}

	attrsId, exists := groupBindings[userId]
	if !exists {
		return "", nil
	}

	return attrsId, nil
}

// Bind 绑定角色
func (m *MemoryAttrsIO) Bind(groupId string, userId string, attrsId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查角色是否存在
	if _, exists := m.items[attrsId]; !exists {
		return errors.New("attrs not found")
	}

	// 确保群组绑定映射存在
	if m.bindings[groupId] == nil {
		m.bindings[groupId] = make(map[string]string)
	}

	m.bindings[groupId][userId] = attrsId
	return nil
}

// Unbind 解绑角色
func (m *MemoryAttrsIO) Unbind(groupId string, userId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	groupBindings, exists := m.bindings[groupId]
	if !exists {
		return nil
	}

	delete(groupBindings, userId)

	// 如果群组没有任何绑定了，删除群组映射
	if len(groupBindings) == 0 {
		delete(m.bindings, groupId)
	}

	return nil
}

// UnbindAll 解绑角色在所有群的绑定
func (m *MemoryAttrsIO) UnbindAll(attrsId string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.unbindAllInternal(attrsId), nil
}

// unbindAllInternal 内部方法，解绑指定角色的所有绑定（不加锁）
func (m *MemoryAttrsIO) unbindAllInternal(attrsId string) int64 {
	var count int64

	for groupId, groupBindings := range m.bindings {
		for userId, boundAttrsId := range groupBindings {
			if boundAttrsId == attrsId {
				delete(groupBindings, userId)
				count++
			}
		}

		// 如果群组没有任何绑定了，删除群组映射
		if len(groupBindings) == 0 {
			delete(m.bindings, groupId)
		}
	}

	return count
}

// BindingGroupIdList 获取指定角色的所有绑定的群ID列表
func (m *MemoryAttrsIO) BindingGroupIdList(attrsId string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var groupIds []string

	for groupId, groupBindings := range m.bindings {
		for _, boundAttrsId := range groupBindings {
			if boundAttrsId == attrsId {
				groupIds = append(groupIds, groupId)
				break // 一个群只需要添加一次
			}
		}
	}

	return groupIds, nil
}
