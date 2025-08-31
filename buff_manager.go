package main

import (
	"regexp"
)

// Buff 表示一个属性增益效果
type Buff struct {
	ID         string   `json:"id"`         // buff的唯一标识
	Keys       []string `json:"keys"`       // 影响的属性名列表
	KeyPattern string   `json:"keyPattern"` // 属性名匹配模式（正则表达式）
	Type       []string `json:"type"`       // buff类型
	Name       string   `json:"name"`       // buff名称
	Add        string   `json:"add"`        // 增加的值表达式
	Source     string   `json:"source"`     // buff来源
}

// BuffManager 管理所有的buff效果
type BuffManager struct {
	buffs *SyncMap[string, *Buff] // 使用SyncMap存储buff
}

// NewBuffManager 创建新的buff管理器
func NewBuffManager() *BuffManager {
	return &BuffManager{
		buffs: &SyncMap[string, *Buff]{},
	}
}

// AddBuff 添加一个buff
func (bm *BuffManager) AddBuff(buff *Buff) {
	bm.buffs.Store(buff.ID, buff)
}

// RemoveBuff 移除指定ID的buff
func (bm *BuffManager) RemoveBuff(id string) {
	bm.buffs.Delete(id)
}

// GetBuff 获取指定ID的buff
func (bm *BuffManager) GetBuff(id string) *Buff {
	if buff, ok := bm.buffs.Load(id); ok {
		return buff
	}
	return nil
}

// GetBuffsForAttr 获取影响指定属性的所有buff
func (bm *BuffManager) GetBuffsForAttr(attrName string) []*Buff {
	var result []*Buff

	bm.buffs.Range(func(id string, buff *Buff) bool {
		// 检查是否在keys列表中
		for _, key := range buff.Keys {
			if key == attrName {
				result = append(result, buff)
				return true // 继续遍历
			}
		}

		// 检查是否匹配keyPattern
		if buff.KeyPattern != "" {
			if matched, _ := regexp.MatchString(buff.KeyPattern, attrName); matched {
				result = append(result, buff)
			}
		}

		return true // 继续遍历
	})

	return result
}

// ListAllBuffs 列出所有buff
func (bm *BuffManager) ListAllBuffs() []*Buff {
	var result []*Buff

	bm.buffs.Range(func(id string, buff *Buff) bool {
		result = append(result, buff)
		return true // 继续遍历
	})

	return result
}
