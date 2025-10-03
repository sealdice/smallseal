package attrs

type AttrsUpsertParams struct {
	Id        string `json:"id"`
	Data      []byte `json:"data"`
	Name      string `json:"name"`
	SheetType string `json:"sheetType"`
	OwnerId   string `json:"ownerId"`
	AttrsType string `json:"attrsType"`
	IsHidden  bool   `json:"isHidden"`
}

// AttrsIO 定义了属性数据访问层的接口
// 这个接口抽象了所有与属性相关的数据库操作
type AttrsIO interface {
	// 根据ID获取角色
	GetById(id string) (*AttributesItem, error)
	// 批量更新插入角色
	Puts(items []*AttrsUpsertParams) error
	// 根据ID删除角色
	DeleteById(id string) error

	// 个人卡: 根据用户ID获取其角色列表
	ListByUid(userId string) ([]*AttributesItem, error)
	// 个人卡: 根据 uid 和 卡名 获取角色，如果不存在，第一个参数返回nil
	GetByUidAndName(userId string, name string) (*AttributesItem, error)

	// 绑卡机制
	// 绑卡: 获取群组绑定的卡片ID
	BindingIdGet(groupId string, userId string) (string, error)
	// 绑卡: 绑定角色
	Bind(groupId string, userId string, attrsId string) error
	// 绑卡: 解绑绝色
	Unbind(groupId string, userId string) error
	// 绑卡: 解绑角色的在所有群的绑定
	UnbindAll(attrsId string) (int64, error)
	// 绑卡: 获取指定角色的所有绑定的群ID列表
	BindingGroupIdList(attrsId string) ([]string, error)
}
