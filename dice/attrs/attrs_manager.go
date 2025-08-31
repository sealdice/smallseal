package attrs

import (
	"context"
	"errors"
	"fmt"
	"time"

	ds "github.com/sealdice/dicescript"

	"smallseal/dice/attrs/attrs_io"
	"smallseal/model/do"
	"smallseal/utils"
)

type AttrsManager struct {
	db any
	// db     engine.DatabaseOperator
	// logger *zap.SugaredLogger
	cancel context.CancelFunc
	m      utils.SyncMap[string, *AttributesItem]
}

func (am *AttrsManager) Stop() {
	// logger.M().Info("结束数据库保存程序...")
	am.cancel()
}

// LoadByCtx 获取当前角色，如有绑定，则获取绑定的角色，若无绑定，获取群内默认卡
// func (am *AttrsManager) LoadByCtx(ctx *MsgContext) (*AttributesItem, error) {
// 	// 如果是兼容性测试环境，跳过绑定查询以避免不必要的数据库操作
// 	if ctx.IsCompatibilityTest {
// 		return am.LoadByIdDirect(ctx.Group.GroupID, ctx.Player.UserID)
// 	}
// 	return am.Load(ctx.Group.GroupID, ctx.Player.UserID)
// }

func (am *AttrsManager) Load(groupId string, userId string) (*AttributesItem, error) {
	userId = am.UIDConvert(userId)

	// 组装当前群-用户的id
	gid := fmt.Sprintf("%s-%s", groupId, userId)

	//	1. 首先获取当前群+用户所绑定的卡
	// 绑定卡的id是nanoid
	id, err := attrs_io.AttrsGetBindingSheetIdByGroupId(am.db, gid)
	if err != nil {
		return nil, err
	}

	// 2. 如果取不到，那么取用户在当前群的默认卡
	if id == "" {
		id = gid
	}

	return am.LoadById(id)
}

// LoadByIdDirect 直接使用组合ID加载数据，跳过绑定查询（用于兼容性测试）
func (am *AttrsManager) LoadByIdDirect(groupId string, userId string) (*AttributesItem, error) {
	userId = am.UIDConvert(userId)
	// 直接使用组合ID，跳过绑定查询
	id := fmt.Sprintf("%s-%s", groupId, userId)
	return am.LoadById(id)
}

func (am *AttrsManager) UIDConvert(userId string) string {
	// 如果存在一个虚拟id，那么返回虚拟id，不存在原样返回
	return userId
}

func (am *AttrsManager) GetCharacterList(userId string) ([]*do.AttributesItemDO, error) {
	userId = am.UIDConvert(userId)
	lst, err := attrs_io.AttrsGetCharacterListByUserId(am.db, userId)
	if err != nil {
		return nil, err
	}
	return lst, err
}

func (am *AttrsManager) CharNew(userId string, name string, sheetType string) (*do.AttributesItemDO, error) {
	userId = am.UIDConvert(userId)
	dict := &ds.ValueMap{}
	// dict.Store("$sheetType", ds.NewStrVal(sheetType))
	json, err := ds.NewDictVal(dict).V().ToJSON()
	if err != nil {
		return nil, err
	}

	return attrs_io.AttrsNewItem(am.db, &do.AttributesItemDO{
		Name:      name,
		OwnerId:   userId,
		AttrsType: "character",
		SheetType: sheetType,
		Data:      json,
	})
}

func (am *AttrsManager) CharDelete(id string) error {
	if err := attrs_io.AttrsDeleteById(am.db, id); err != nil {
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
	// 1. 如果当前有缓存，那么从缓存中返回。
	// 但是。。如果有人把这个对象一直持有呢？
	i, exists := am.m.Load(id)
	if exists {
		return i, nil
	}

	// 2. 从新数据库加载
	data, err := attrs_io.AttrsGetById(am.db, id)
	if err == nil {
		if data.IsDataExists() {
			var v *ds.VMValue
			v, err = ds.VMValueFromJSON(data.Data)
			if err != nil {
				return nil, err
			}
			if dd, ok := v.ReadDictData(); ok {
				i = &AttributesItem{
					ID:           id,
					valueMap:     dd.Dict,
					Name:         data.Name,
					SheetType:    data.SheetType,
					LastUsedTime: time.Now().Unix(),
					IsSaved:      true,
				}
				am.m.Store(id, i)
				return i, nil
			} else {
				return nil, errors.New("角色数据类型不正确")
			}
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
	return nil
}

// CheckAndFreeUnused 此函数会被定期调用，释放最近不用的对象
func (am *AttrsManager) CheckAndFreeUnused() error {
	return nil
}

func (am *AttrsManager) CharBind(charId string, groupId string, userId string) error {
	userId = am.UIDConvert(userId)
	id := fmt.Sprintf("%s-%s", groupId, userId)
	return attrs_io.AttrsBindCharacter(am.db, charId, id)
}

// CharGetBindingId 获取当前群绑定的角色ID
func (am *AttrsManager) CharGetBindingId(groupId string, userId string) (string, error) {
	userId = am.UIDConvert(userId)
	id := fmt.Sprintf("%s-%s", groupId, userId)
	return attrs_io.AttrsGetBindingSheetIdByGroupId(am.db, id)
}

func (am *AttrsManager) CharIdGetByName(userId string, name string) (string, error) {
	return attrs_io.AttrsGetIdByUidAndName(am.db, userId, name)
}

func (am *AttrsManager) CharCheckExists(userId string, name string) bool {
	id, _ := am.CharIdGetByName(userId, name)
	return id != ""
}

func (am *AttrsManager) CharGetBindingGroupIdList(id string) []string {
	all, err := attrs_io.AttrsCharGetBindingList(am.db, id)
	if err != nil {
		return []string{}
	}
	// 只要群号
	for i, v := range all {
		a, b, _ := utils.UnpackGroupUserId(v)
		if b != "" {
			all[i] = a
		} else {
			all[i] = b
		}
	}
	return all
}

func (am *AttrsManager) CharUnbindAll(id string) []string {
	all := am.CharGetBindingGroupIdList(id)
	_, err := attrs_io.AttrsCharUnbindAll(am.db, id)
	if err != nil {
		return []string{}
	}
	return all
}
