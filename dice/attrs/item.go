package attrs

import (
	"time"

	ds "github.com/sealdice/dicescript"

	"smallseal/dice/attrs/attrs_io"
)

// AttributesItem 这是一个人物卡对象
type AttributesItem struct {
	ID               string
	valueMap         *ds.ValueMap // SyncMap[string, *ds.VMValue] 这种类型更好吗？我得确认一下js兼容性
	LastModifiedTime int64        // 上次修改时间
	LastUsedTime     int64        // 上次使用时间
	IsSaved          bool
	Name             string
	SheetType        string
}

func (i *AttributesItem) SaveToDB(db any) {
	// 使用事务写入
	rawData, err := ds.NewDictVal(i.valueMap).V().ToJSON()
	if err != nil {
		return
	}
	err = attrs_io.AttrsPutById(db, i.ID, rawData, i.Name, i.SheetType)
	if err != nil {
		// logger.M().Error("保存数据失败", err.Error())
		return
	}
	i.IsSaved = true
}

func (i *AttributesItem) Load(name string) *ds.VMValue {
	v, _ := i.valueMap.Load(name)
	i.LastUsedTime = time.Now().Unix()
	return v
}

func (i *AttributesItem) LoadX(name string) (*ds.VMValue, bool) {
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
