package attrs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"time"

	ds "github.com/sealdice/dicescript"

	"github.com/sealdice/smallseal/utils"
)

type AttrsManager struct {
	cancel context.CancelFunc
	m      utils.SyncMap[string, *AttributesItem]

	io AttrsIO
}

func (am *AttrsManager) SetIO(io AttrsIO) {
	am.io = io
}

func (am *AttrsManager) Stop() {
	am.cancel()
}

func (am *AttrsManager) Load(groupId string, userId string) (*AttributesItem, error) {
	userId = am.UIDConvert(userId)

	am.ensureIO()

	// 组装当前群-用户的id
	gid := fmt.Sprintf("%s-%s", groupId, userId)

	//	1. 首先获取当前群+用户所绑定的卡
	// 绑定卡的id是nanoid
	id, _ := am.io.BindingIdGet(groupId, userId)
	// if err != nil {
	// 	return nil, err
	// }

	// 2. 如果取不到，那么取用户在当前群的默认卡
	if id == "" {
		id = gid
	}

	return am.LoadById(id)
}

func (am *AttrsManager) UIDConvert(userId string) string {
	// 如果存在一个虚拟id，那么返回虚拟id，不存在原样返回
	return userId
}

func (am *AttrsManager) GetCharacterList(userId string) ([]*AttributesItem, error) {
	userId = am.UIDConvert(userId)
	am.ensureIO()
	lst, err := am.io.ListByUid(userId)
	if err != nil {
		return nil, err
	}
	for _, item := range lst {
		groups, err := am.io.BindingGroupIdList(item.ID)
		if err != nil {
			item.BindingGroupsNum = 0
			continue
		}
		item.BindingGroupsNum = len(groups)
	}
	sort.Slice(lst, func(i, j int) bool {
		if lst[i].LastUsedTime == lst[j].LastUsedTime {
			return lst[i].Name < lst[j].Name
		}
		return lst[i].LastUsedTime > lst[j].LastUsedTime
	})
	return lst, nil
}

func (am *AttrsManager) CharNew(userId string, name string, sheetType string) (*AttributesItem, error) {
	userId = am.UIDConvert(userId)
	am.ensureIO()
	dict := &ds.ValueMap{}
	// dict.Store("$sheetType", ds.NewStrVal(sheetType))
	json, err := ds.NewDictVal(dict).V().ToJSON()
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	item := &AttributesItem{
		ID:               generateAttrID(),
		Name:             name,
		OwnerId:          userId,
		AttrsType:        "character",
		SheetType:        sheetType,
		Data:             json,
		valueMap:         dict,
		LastModifiedTime: now,
		LastUsedTime:     now,
		IsSaved:          true,
	}
	params := []*AttrsUpsertParams{
		{
			Id:        item.ID,
			Data:      item.Data,
			Name:      item.Name,
			SheetType: item.SheetType,
			OwnerId:   item.OwnerId,
			AttrsType: item.AttrsType,
			IsHidden:  item.IsHidden,
		},
	}
	if err := am.io.Puts(params); err != nil {
		return nil, err
	}
	am.m.Store(item.ID, item)
	return item, nil
}

func (am *AttrsManager) CharDelete(id string) error {
	am.ensureIO()
	if err := am.io.DeleteById(id); err != nil {
		return err
	}
	// 从缓存中删除
	am.m.Delete(id)
	return nil
}

// LoadById 数据加载，负责以下数据
// 1. 群内用户的默认卡(id格式为：群id:用户id)
// 2. 用户创建出的角色卡（指定id）
// 3. 群属性(id为群id)
// 4. 用户全局属性
func (am *AttrsManager) LoadById(id string) (*AttributesItem, error) {
	am.ensureIO()
	// 1. 如果当前有缓存，那么从缓存中返回。
	// 但是。。如果有人把这个对象一直持有呢？
	i, exists := am.m.Load(id)
	if exists {
		// cache
		// i.Range(func(key string, value *ds.VMValue) bool {
		// 	fmt.Println("x", key, value.ToString())
		// 	return true
		// })
		return i, nil
	}

	// 2. 从新数据库加载
	data, err := am.io.GetById(id)
	if err == nil && data.IsDataExists() {
		var v *ds.VMValue
		v, err = ds.VMValueFromJSON(data.Data)
		if err != nil {
			return nil, err
		}
		if dd, ok := v.ReadDictData(); ok {
			i = &AttributesItem{
				ID:               id,
				valueMap:         dd.Dict,
				Name:             data.Name,
				SheetType:        data.SheetType,
				OwnerId:          data.OwnerId,
				AttrsType:        data.AttrsType,
				IsHidden:         data.IsHidden,
				BindingGroupsNum: data.BindingGroupsNum,
				LastModifiedTime: data.LastModifiedTime,
				LastUsedTime:     time.Now().Unix(),
				IsSaved:          true,
			}
			am.m.Store(id, i)
			return i, nil
		} else {
			return nil, errors.New("角色数据类型不正确")
		}
	}
	// 之前 else 是读不出时返回报错
	// return nil, err
	// 改为创建新数据集。因为遇到一个特别案例：因为clr前会读取当前角色数据，因为读不出来所以无法st clr
	// 从而永久卡死

	// 3. 创建一个新的
	// 注: 缺 created_at、updated_at、sheet_type、owner_id、is_hidden、nickname等各项
	// 可能需要ctx了
	now := time.Now().Unix()
	i = &AttributesItem{
		ID:               id,
		valueMap:         &ds.ValueMap{},
		LastModifiedTime: now,
		LastUsedTime:     now,
		IsSaved:          false,
	}
	am.m.Store(id, i)
	return i, nil
}

func (am *AttrsManager) Init() {
	// am.db = d.DBOperator
	// am.logger = d.Logger
	// 创建一个 context 用于取消 goroutine
	ctx, cancel := context.WithCancel(context.Background())
	// 启动后台定时任务
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				// 检测到取消信号，执行最后一次保存后退出
				// d.Logger.Info("正在执行最后一次数据保存...")
				if err := am.CheckForSave(); err != nil {
					// d.Logger.Errorf("最终数据保存失败: %v", err)
				}
				return
			case <-ticker.C:
				// 定时执行保存和清理任务
				if err := am.CheckForSave(); err != nil {
					// d.Logger.Errorf("数据库保存程序出错: %v", err)
				}
				if err := am.CheckAndFreeUnused(); err != nil {
					// d.Logger.Errorf("数据库保存-清理程序出错: %v", err)
				}
			}
		}
	}()
	am.cancel = cancel
}

func (am *AttrsManager) CheckForSave() error {
	if am.io == nil {
		// 尚未初始化
		return errors.New("属性IO尚未初始化")
	}

	var resultList []*AttrsUpsertParams
	prepareToSave := map[string]int{}

	am.m.Range(func(key string, value *AttributesItem) bool {
		if !value.IsSaved {
			saveModel, err := value.ToAttrsUpsertParams()
			if err != nil {
				// 打印日志，但继续处理其他项目
				// log.Errorf("定期写入用户数据出错(获取批量保存模型): %v", err)
				return true
			}
			prepareToSave[key] = 1
			resultList = append(resultList, saveModel)
		}
		return true
	})

	// 整体落盘
	if len(resultList) == 0 {
		return nil
	}

	if err := am.io.Puts(resultList); err != nil {
		// log.Errorf("定期写入用户数据出错(批量保存): %v", err)
		return err
	}

	for key := range prepareToSave {
		// 理应不存在这个数据没有的情况
		v, _ := am.m.Load(key)
		v.IsSaved = true
	}

	return nil
}

// CheckAndFreeUnused 此函数会被定期调用，释放最近不用的对象
func (am *AttrsManager) CheckAndFreeUnused() error {
	return nil
}

func (am *AttrsManager) CharBind(charId string, groupId string, userId string) error {
	userId = am.UIDConvert(userId)
	am.ensureIO()
	if charId == "" {
		return am.io.Unbind(groupId, userId)
	}
	return am.io.Bind(groupId, userId, charId)
}

// CharGetBindingId 获取当前群绑定的角色ID
func (am *AttrsManager) CharGetBindingId(groupId string, userId string) (string, error) {
	userId = am.UIDConvert(userId)
	am.ensureIO()
	return am.io.BindingIdGet(groupId, userId)
}

func (am *AttrsManager) CharIdGetByName(userId string, name string) (string, error) {
	userId = am.UIDConvert(userId)
	am.ensureIO()
	item, err := am.io.GetByUidAndName(userId, name)
	if err != nil {
		return "", err
	}
	if item == nil {
		return "", nil
	}
	return item.ID, nil
}

func (am *AttrsManager) CharCheckExists(userId string, name string) bool {
	am.ensureIO()
	id, err := am.CharIdGetByName(userId, name)
	if err != nil {
		return false
	}
	return id != ""
}

func (am *AttrsManager) CharGetBindingGroupIdList(id string) []string {
	am.ensureIO()
	all, err := am.io.BindingGroupIdList(id)
	if err != nil {
		return []string{}
	}
	return all
}

func (am *AttrsManager) CharUnbindAll(id string) []string {
	am.ensureIO()
	all := am.CharGetBindingGroupIdList(id)
	_, err := am.io.UnbindAll(id)
	if err != nil {
		return []string{}
	}
	return all
}

func (am *AttrsManager) Save(item *AttributesItem) error {
	am.ensureIO()
	if item == nil {
		return errors.New("attributes item is nil")
	}
	if item.valueMap == nil {
		item.valueMap = &ds.ValueMap{}
	}
	jsonData, err := ds.NewDictVal(item.valueMap).V().ToJSON()
	if err != nil {
		return err
	}
	item.Data = jsonData
	item.LastModifiedTime = time.Now().Unix()
	params := []*AttrsUpsertParams{
		{
			Id:        item.ID,
			Data:      item.Data,
			Name:      item.Name,
			SheetType: item.SheetType,
			OwnerId:   item.OwnerId,
			AttrsType: item.AttrsType,
			IsHidden:  item.IsHidden,
		},
	}
	if err := am.io.Puts(params); err != nil {
		return err
	}
	item.IsSaved = true
	am.m.Store(item.ID, item)
	return nil
}

func (am *AttrsManager) ensureIO() {
	if am.io == nil {
		am.io = NewMemoryAttrsIO()
	}
}

func generateAttrID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err == nil {
		return "char-" + hex.EncodeToString(buf)
	}
	return fmt.Sprintf("char-%d", time.Now().UnixNano())
}
