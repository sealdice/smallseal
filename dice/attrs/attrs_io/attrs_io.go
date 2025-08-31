package attrs_io

import (
	"fmt"
	"smallseal/model/do"
)

// 这一层参考 core 当前版本的抽象情况，其实不排除有点过于抽象，后面再改吧
// 应该还能做一个更底层的抽象，只把读写部分做几个接口

func AttrsGetBindingSheetIdByGroupId(db any, groupId string) (string, error) {
	// return fmt.Sprintf("binding_%s", groupId)
	return fmt.Sprintf("binding_%s", groupId), nil
}

func AttrsGetCharacterListByUserId(operator any, userId string) ([]*do.AttributesItemDO, error) {
	return nil, nil
}

func AttrsNewItem(db any, item *do.AttributesItemDO) (*do.AttributesItemDO, error) {
	return nil, nil
}

func AttrsDeleteById(db any, id string) error {
	return nil
}

func AttrsGetById(db any, id string) (*do.AttributesItemDO, error) {
	return &do.AttributesItemDO{}, nil
}

type AttributesBatchUpsertModel struct {
	Id        string `json:"id"`
	Data      []byte `json:"data"`
	Name      string `json:"name"`
	SheetType string `json:"sheetType"`
}

// AttrsPutsByIDBatch 特殊入库函数
// https://github.com/go-gorm/gorm/issues/6047 混合的更新疑似仍然有问题，所以手动拆分逻辑
func AttrsPutsByIDBatch(operator any, saveList []*AttributesBatchUpsertModel) error {
	return nil
}

func AttrsBindCharacter(operator any, charId string, id string) error {
	return nil
}

func AttrsPutById(operator any, id string, data []byte, name, sheetType string) error {
	return nil
}

func AttrsCharUnbindAll(operator any, id string) (int64, error) {
	return 0, nil
}

func AttrsCharGetBindingList(operator any, id string) ([]string, error) {
	return nil, nil
}

func AttrsGetIdByUidAndName(operator any, userId string, name string) (string, error) {
	return "", nil
}
