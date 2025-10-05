package attrs

import (
	"time"

	ds "github.com/sealdice/dicescript"
)

// AttributesItem 这是一个人物卡对象
type AttributesItem struct {
	ID        string // 如果是群内，那么是类似 QQ-Group:12345-QQ:678910，群外是nanoid
	AttrsType string `json:"attrsType"` // 分为: 角色(character)、组内用户(group_user)、群组(group)、用户(user)
	IsHidden  bool   `json:"isHidden"`  // 隐藏的卡片不出现在 pc list 中

	Data     []byte       `json:"data"` // 序列化后的卡数据
	valueMap *ds.ValueMap // SyncMap[string, *ds.VMValue] 这种类型更好吗？我得确认一下js兼容性

	// 角色卡专用
	Name      string `json:"name"`      // 卡片名称
	SheetType string `json:"sheetType"` // 卡片类型，如dnd5e coc7
	OwnerId   string `json:"ownerId"`   // 若有明确归属，就是对应的UniformID

	BindingGroupsNum int `json:"bindingGroupsNum"` // 绑定的群数量，仅供统计展示

	// 其他
	LastModifiedTime int64 // 上次修改时间
	LastUsedTime     int64 // 上次使用时间
	IsSaved          bool
}

// func (i *AttributesItem) Load(name string) *ds.VMValue {
// 	v, _ := i.valueMap.Load(name)
// 	i.LastUsedTime = time.Now().Unix()
// 	return v
// }

func (i *AttributesItem) Load(name string) (*ds.VMValue, bool) {
	v, exists := i.valueMap.Load(name)
	i.LastUsedTime = time.Now().Unix()
	return v, exists
}

func (i *AttributesItem) Delete(name string) {
	i.valueMap.Delete(name)
	i.LastModifiedTime = time.Now().Unix()
}

func (i *AttributesItem) SetModified() {
	i.LastModifiedTime = time.Now().Unix()
	i.IsSaved = false
}

func (i *AttributesItem) Store(name string, value *ds.VMValue) {
	now := time.Now().Unix()
	i.valueMap.Store(name, value)
	i.LastModifiedTime = now
	i.LastUsedTime = now
	i.IsSaved = false
}

func (i *AttributesItem) Clear() int {
	size := i.valueMap.Length()
	i.valueMap.Clear()
	i.LastModifiedTime = time.Now().Unix()
	i.IsSaved = false
	return size
}

func (i *AttributesItem) ToArrayKeys() []*ds.VMValue {
	var items []*ds.VMValue
	i.valueMap.Range(func(key string, value *ds.VMValue) bool {
		items = append(items, ds.NewStrVal(key))
		return true
	})
	return items
}

func (i *AttributesItem) ToArrayValues() []*ds.VMValue {
	var items []*ds.VMValue
	i.valueMap.Range(func(key string, value *ds.VMValue) bool {
		items = append(items, value)
		return true
	})
	return items
}

func (i *AttributesItem) ToArrayItems() []*ds.VMValue {
	var items []*ds.VMValue
	i.valueMap.Range(func(key string, value *ds.VMValue) bool {
		items = append(
			items,
			ds.NewArrayVal(ds.NewStrVal(key), value),
		)
		return true
	})
	return items
}

func (i *AttributesItem) Range(f func(key string, value *ds.VMValue) bool) {
	i.valueMap.Range(f)
}

func (i *AttributesItem) SetSheetType(system string) {
	i.SheetType = system
	i.LastModifiedTime = time.Now().Unix()
	i.IsSaved = false
}

func (i *AttributesItem) Len() int {
	return i.valueMap.Length()
}

func (i *AttributesItem) IsDataExists() bool {
	return len(i.Data) > 0
}

func (i *AttributesItem) GetValueMap() *ds.ValueMap {
	return i.valueMap
}

// ToAttrsUpsertParams 获取批量保存模型
func (i *AttributesItem) ToAttrsUpsertParams() (*AttrsUpsertParams, error) {
	// 序列化数据
	if i.valueMap == nil {
		i.valueMap = &ds.ValueMap{}
	}
	jsonData, err := ds.NewDictVal(i.valueMap).V().ToJSON()
	if err != nil {
		return nil, err
	}

	i.Data = jsonData
	i.LastModifiedTime = time.Now().Unix()

	return &AttrsUpsertParams{
		Id:        i.ID,
		Data:      i.Data,
		Name:      i.Name,
		SheetType: i.SheetType,
		OwnerId:   i.OwnerId,
		AttrsType: i.AttrsType,
		IsHidden:  i.IsHidden,
	}, nil
}
