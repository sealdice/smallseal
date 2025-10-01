package adapters

import "github.com/sealdice/smallseal/dice/types"

type RequestBase struct {
	MessageId string // 消息ID
}

type SimpleUserInfo struct {
	UserId   string // 用户ID
	UserName string // 用户名
}

// AdapterCallback 适配器回调接口
type AdapterCallback interface {
	OnError(err error)
	OnMessageReceived(info *MessageSendCallbackInfo)
	OnEvent(evt *types.AdapterEvent)
}

// AdapterEvent captures non-message events pushed from platform adapters.
// 定义移动至 dice/types 以便核心逻辑复用。
// MessageSendRequest 发送消息请求
type MessageSendRequest struct {
	RequestBase
	Sender   *SimpleUserInfo         // 发送者 可不填
	Segments []types.IMessageElement // 消息段
	TargetId any                     // 目标用户ID/群组ID, string或int64
}

// MessageSendFileRequest 发送文件请求
type MessageSendFileRequest struct {
	RequestBase
	Sender   *SimpleUserInfo // 发送者 可不填
	FilePath string          // 文件路径
	TargetId any             // 目标用户ID/群组ID, string或int64
}

// GroupOperationRequest 群组操作请求
type GroupOperationKickRequest struct {
	RequestBase
	GroupID any // 群组ID
	UserID  any // 被操作用户ID（如果涉及用户操作）
}

// GroupOperationRequest 群组操作请求
type GroupOperationBanRequest struct {
	RequestBase
	GroupID  any   // 群组ID
	UserID   any   // 被操作用户ID（如果涉及用户操作）
	Duration int64 // 持续时间（如禁言时长）
}

// GroupOperationQuitRequest 群组操作请求
type GroupOperationQuitRequest struct {
	RequestBase
	GroupID any // 群组ID
}

// GroupOperationCardNameSetRequest 群组操作请求
type GroupOperationCardNameSetRequest struct {
	RequestBase
	GroupID any    // 群组ID
	UserID  any    // 用户ID（如果涉及用户操作）
	Name    string // 名称（如群名片）
}

// GroupInfo 群组信息
type GroupInfo struct {
	GroupID        any    `json:"group_id"`         // 群组ID
	GroupName      string `json:"group_name"`       // 群组名称
	MemberCount    int64  `json:"member_count"`     // 成员数量
	MaxMemberCount uint   `json:"max_member_count"` // 最大成员数量
}

// MessageOperationRequest 消息操作请求
type MessageOperationRequest struct {
	RequestBase
	MessageID any                     // 消息ID
	Segments  []types.IMessageElement // 消息段
}

// FriendOperationRequest 好友操作请求
type FriendOperationRequest struct {
	RequestBase
	UserID any // 用户ID
}

// GroupFileListRequest 获取群文件列表请求
type GroupFileListRequest struct {
	RequestBase
	GroupID  any    // 群组ID
	FolderID string // 文件夹ID，空字符串表示根目录
}

// GroupFileDownloadRequest 下载群文件请求
type GroupFileDownloadRequest struct {
	RequestBase
	GroupID any    // 群组ID
	FileID  string // 文件ID
	BusID   int32  // 文件类型
}

// GroupFileUploadRequest 上传群文件请求
type GroupFileUploadRequest struct {
	RequestBase
	GroupID  any    // 群组ID
	FilePath string // 本地文件路径
	FileName string // 文件名（可选，默认使用文件路径中的文件名）
	FolderID string // 目标文件夹ID，空字符串表示根目录
}

// GroupFileInfo 群文件信息
type GroupFileInfo struct {
	FileID        string `json:"file_id"`        // 文件ID
	FileName      string `json:"file_name"`      // 文件名
	BusID         int32  `json:"busid"`          // 文件类型
	FileSize      int64  `json:"file_size"`      // 文件大小
	UploadTime    int64  `json:"upload_time"`    // 上传时间
	DeadTime      int64  `json:"dead_time"`      // 过期时间
	ModifyTime    int64  `json:"modify_time"`    // 修改时间
	DownloadTimes int32  `json:"download_times"` // 下载次数
	Uploader      int64  `json:"uploader"`       // 上传者QQ号
	UploaderName  string `json:"uploader_name"`  // 上传者名称
	FolderPath    string `json:"folder_path"`    // 文件夹路径
}

// GroupFolderInfo 群文件夹信息
type GroupFolderInfo struct {
	FolderID       string `json:"folder_id"`        // 文件夹ID
	FolderName     string `json:"folder_name"`      // 文件夹名
	CreateTime     int64  `json:"create_time"`      // 创建时间
	Creator        int64  `json:"creator"`          // 创建者QQ号
	CreatorName    string `json:"creator_name"`     // 创建者名称
	TotalFileCount int32  `json:"total_file_count"` // 文件总数
}

// GroupFileListResponse 群文件列表响应
type GroupFileListResponse struct {
	Files   []GroupFileInfo   `json:"files"`   // 文件列表
	Folders []GroupFolderInfo `json:"folders"` // 文件夹列表
}

// GroupFileDownloadResponse 群文件下载响应
type GroupFileDownloadResponse struct {
	URL string `json:"url"` // 文件下载链接
}
