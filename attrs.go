package main

import (
	"encoding/json"
	"os"
	"sort"
	"strconv"

	ds "github.com/sealdice/dicescript"
)

// TODO: 和原本的attrs manager区别太大，准备还是用原始写法

// AttributesItem 表示一个角色的人物卡数据
// 这里提供最简单的内存实现，键值对为字符串->VMValue
type AttributesItem struct {
	UserID  string
	GroupID string
	Data    *ds.ValueMap
}

func NewAttributesItem(userID, groupID string) *AttributesItem {
	return &AttributesItem{
		UserID:  userID,
		GroupID: groupID,
		Data:    &ds.ValueMap{},
	}
}

func (a *AttributesItem) Load(name string) (*ds.VMValue, bool) { return a.Data.Load(name) }
func (a *AttributesItem) Store(name string, v *ds.VMValue)     { a.Data.Store(name, v) }
func (a *AttributesItem) Delete(name string)                   { a.Data.Delete(name) }

// AttrsManager 管理用户和群组的人物卡（内存+简易持久化）
// 覆盖层：临时(VM) -> 个人(全局) -> 群组(全局) -> 群组内个人
// 注意：临时层在VM中实现，这里提供其余三层

type AttrsManager struct {
	// key: groupID -> userID -> AttributesItem
	items map[string]map[string]*AttributesItem
}

func NewAttrsManager() *AttrsManager {
	return &AttrsManager{items: map[string]map[string]*AttributesItem{}}
}

// Load 返回群组内个人的人物卡（覆盖层最低一层）
func (m *AttrsManager) Load(groupID, userID string) *AttributesItem {
	if _, ok := m.items[groupID]; !ok {
		m.items[groupID] = map[string]*AttributesItem{}
	}
	if it, ok := m.items[groupID][userID]; ok {
		return it
	}
	it := NewAttributesItem(userID, groupID)
	m.items[groupID][userID] = it
	return it
}

// LoadUserGlobal 返回个人全局层（不依赖群组）
func (m *AttrsManager) LoadUserGlobal(userID string) *AttributesItem {
	return m.Load("", userID)
}

// LoadGroupGlobal 返回群组全局层（不依赖用户）
func (m *AttrsManager) LoadGroupGlobal(groupID string) *AttributesItem {
	return m.Load(groupID, "")
}

func (m *AttrsManager) Delete(groupID, userID string) {
	if _, ok := m.items[groupID]; ok {
		delete(m.items[groupID], userID)
	}
}

// ResolveByLayers 根据覆盖层顺序解析变量
// 顺序：个人全局 -> 群组全局 -> 群组个人
func (m *AttrsManager) ResolveByLayers(groupID, userID, name string) (*ds.VMValue, bool) {
	// 个人全局
	if it := m.LoadUserGlobal(userID); it != nil {
		if v, ok := it.Load(name); ok {
			return v, true
		}
	}
	// 群组全局
	if groupID != "" {
		if it := m.LoadGroupGlobal(groupID); it != nil {
			if v, ok := it.Load(name); ok {
				return v, true
			}
		}
	}
	// 群组个人
	if groupID != "" && userID != "" {
		if it := m.Load(groupID, userID); it != nil {
			if v, ok := it.Load(name); ok {
				return v, true
			}
		}
	}
	return nil, false
}

// -------- 简易持久化（JSON）---------
// 存储为： groupID -> userID -> attrName -> stringValue
// 仅保存值的字符串表示，加载时尝试转为int，否则为string

type dumpData = map[string]map[string]map[string]string

func (m *AttrsManager) SaveToFile(path string) error {
	data := dumpData{}
	for gid, users := range m.items {
		if _, ok := data[gid]; !ok {
			data[gid] = map[string]map[string]string{}
		}
		for uid, item := range users {
			// 提取并排序键，保证结果稳定
			keys := []string{}
			item.Data.Range(func(k string, _ *ds.VMValue) bool { keys = append(keys, k); return true })
			sort.Strings(keys)
			kv := map[string]string{}
			for _, k := range keys {
				if v, ok := item.Load(k); ok {
					kv[k] = v.ToString()
				}
			}
			data[gid][uid] = kv
		}
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func (m *AttrsManager) LoadFromFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	data := dumpData{}
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	m.items = map[string]map[string]*AttributesItem{}
	for gid, users := range data {
		for uid, kv := range users {
			it := m.Load(gid, uid)
			for k, s := range kv {
				// 尝试转为整数
				if iv, e := strconv.ParseInt(s, 10, 64); e == nil {
					it.Store(k, ds.NewIntVal(ds.IntType(iv)))
				} else {
					it.Store(k, ds.NewStrVal(s))
				}
			}
		}
	}
	return nil
}
