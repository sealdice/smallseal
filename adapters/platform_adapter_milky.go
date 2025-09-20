package adapters

import (
	"fmt"
	"path/filepath"
	"smallseal/dice/types"
	"smallseal/utils"
	"strconv"
	"strings"
	"time"

	milky "github.com/Szzrain/Milky-go-sdk"
	"go.uber.org/zap"
)

// MessageSendCallbackInfo 消息发送回调信息
type MessageSendCallbackInfo struct {
	Sender  *SimpleUserInfo
	Message *types.Message
}

// GroupNameCacheItem 群组名称缓存项
type GroupNameCacheItem struct {
	Name string
	time int64
}

// PlatformAdapterMilky Milky 平台适配器
type PlatformAdapterMilky struct {
	WsGateway     string         `json:"ws_gateway"   yaml:"ws_gateway"`
	RestGateway   string         `json:"rest_gateway" yaml:"rest_gateway"`
	Token         string         `json:"token"        yaml:"token"`
	IntentSession *milky.Session `json:"-"            yaml:"-"`

	// 缓存引用，避免长链式调用
	groupNameCache *utils.SyncMap[string, *GroupNameCacheItem]

	// 回调接口
	callback AdapterCallback

	// 连接状态
	isAlive bool
}

// SetCallback 设置回调接口
func (pa *PlatformAdapterMilky) SetCallback(callback AdapterCallback) {
	pa.callback = callback
}

// IsAlive 检查连接是否存活
func (pa *PlatformAdapterMilky) IsAlive() bool {
	return pa.IntentSession != nil && pa.isAlive
}

// MsgSendToGroup 发送消息到群组
func (pa *PlatformAdapterMilky) MsgSendToGroup(request *MessageSendRequest) (bool, error) {
	log := zap.S().Named("adapter")

	// 提取群组ID
	var groupIDStr string
	switch v := request.TargetId.(type) {
	case string:
		groupIDStr = ExtractQQGroupID(v)
	case int64:
		groupIDStr = strconv.FormatInt(v, 10)
	default:
		err := fmt.Errorf("invalid group ID type: %T", request.TargetId)
		log.Error(err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	id, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		log.Errorf("Invalid group ID %s: %v", groupIDStr, err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	elements := ParseMessageToMilky(request.Segments)
	ret, err := pa.IntentSession.SendGroupMessage(id, &elements)
	if err != nil {
		log.Errorf("Failed to send group message to %s: %v", groupIDStr, err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	// 构造消息对象用于回调
	msg := &types.Message{
		Platform:    "QQ",
		MessageType: "group",
		GroupID:     groupIDStr,
		Segment:     request.Segments,
		RawID:       ret.MessageSeq,
	}

	if request.Sender != nil {
		msg.Sender = types.SenderBase{
			UserID:   request.Sender.UserId,
			Nickname: request.Sender.UserName,
		}
	}

	// 使用回调接口
	if pa.callback != nil {
		info := &MessageSendCallbackInfo{
			Sender:  request.Sender,
			Message: msg,
		}
		pa.callback.OnMessageReceived(info)
	}

	return true, nil
}

// MsgSendToPerson 发送消息到个人
func (pa *PlatformAdapterMilky) MsgSendToPerson(request *MessageSendRequest) (bool, error) {
	log := zap.S().Named("adapter")

	// 提取用户ID
	var userIDStr string
	switch v := request.TargetId.(type) {
	case string:
		userIDStr = ExtractQQUserID(v)
	case int64:
		userIDStr = strconv.FormatInt(v, 10)
	default:
		err := fmt.Errorf("invalid user ID type: %T", request.TargetId)
		log.Error(err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	id, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Errorf("Invalid user ID %s: %v", userIDStr, err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	elements := ParseMessageToMilky(request.Segments)
	ret, err := pa.IntentSession.SendPrivateMessage(id, &elements)
	if err != nil {
		log.Errorf("Failed to send private message to %s: %v", userIDStr, err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	// 构造消息对象用于回调
	msg := &types.Message{
		Platform:    "QQ",
		MessageType: "private",
		Segment:     request.Segments,
		RawID:       ret.MessageSeq,
	}

	if request.Sender != nil {
		msg.Sender = types.SenderBase{
			UserID:   request.Sender.UserId,
			Nickname: request.Sender.UserName,
		}
	}

	// 使用回调接口
	if pa.callback != nil {
		info := &MessageSendCallbackInfo{
			Sender:  request.Sender,
			Message: msg,
		}
		pa.callback.OnMessageReceived(info)
	}

	return true, nil
}

// GetGroupInfoAsync 异步获取群组信息
func (pa *PlatformAdapterMilky) GetGroupInfoAsync(groupID string) {
	go func() {
		info, err := pa.GroupInfoGet(groupID)
		if err != nil {
			if pa.callback != nil {
				pa.callback.OnError(err)
			}
			return
		}

		// 缓存群组信息
		if pa.groupNameCache == nil {
			pa.groupNameCache = &utils.SyncMap[string, *GroupNameCacheItem]{}
		}
		pa.groupNameCache.Store(groupID, &GroupNameCacheItem{
			Name: info.GroupName,
			time: time.Now().Unix(),
		})
	}()
}

// GroupInfoGet 获取群组信息
func (pa *PlatformAdapterMilky) GroupInfoGet(groupID any) (*GroupInfo, error) {
	log := zap.S().Named("adapter")

	// 提取群组ID
	var groupIDStr string
	switch v := groupID.(type) {
	case string:
		groupIDStr = ExtractQQGroupID(v)
	case int64:
		groupIDStr = strconv.FormatInt(v, 10)
	default:
		err := fmt.Errorf("invalid group ID type: %T", groupID)
		log.Error(err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return nil, err
	}

	// 先检查缓存
	if pa.groupNameCache != nil {
		if cached, ok := pa.groupNameCache.Load(groupIDStr); ok {
			// 检查缓存是否过期（5分钟）
			if time.Now().Unix()-cached.time < 300 {
				return &GroupInfo{
					GroupID:   groupIDStr,
					GroupName: cached.Name,
				}, nil
			}
		}
	}

	id, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		log.Errorf("Invalid group ID %s: %v", groupIDStr, err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return nil, err
	}

	groupInfo, err := pa.IntentSession.GetGroupInfo(id, false)
	if err != nil {
		log.Errorf("Failed to get group info for %s: %v", groupIDStr, err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return nil, err
	}

	result := &GroupInfo{
		GroupID:   groupIDStr,
		GroupName: groupInfo.Name,
	}

	// 更新缓存
	if pa.groupNameCache == nil {
		pa.groupNameCache = &utils.SyncMap[string, *GroupNameCacheItem]{}
	}
	pa.groupNameCache.Store(groupIDStr, &GroupNameCacheItem{
		Name: groupInfo.Name,
		time: time.Now().Unix(),
	})

	return result, nil
}

// Serve 启动服务
func (pa *PlatformAdapterMilky) Serve() int {
	log := zap.S().Named("adapter")
	pa.isAlive = false

	// 初始化缓存引用
	if pa.RestGateway[len(pa.RestGateway)-1] == '/' {
		pa.RestGateway = pa.RestGateway[:len(pa.RestGateway)-1] // 去掉末尾的斜杠
	}
	if pa.WsGateway[len(pa.WsGateway)-1] == '/' {
		pa.WsGateway = pa.WsGateway[:len(pa.WsGateway)-1]
	}

	session, err := milky.New(pa.WsGateway, pa.RestGateway, pa.Token, log.Named("adapter"))
	if err != nil {
		log.Errorf("Milky SDK initialization failed: %v", err)
		return 1
	}
	pa.IntentSession = session

	// 添加消息处理器
	session.AddHandler(func(session2 *milky.Session, m *milky.ReceiveMessage) {
		if m == nil {
			return
		}
		log.Debugf("Received message: Sender %d", m.SenderId)

		msg := &types.Message{
			Platform: "QQ",
			Time:     m.Time,
			RawID:    m.MessageSeq,
			Sender: types.SenderBase{
				UserID: FormatDiceIDQQ(strconv.FormatInt(m.SenderId, 10)),
			},
		}

		switch m.MessageScene {
		case "group":
			if m.Group != nil && m.GroupMember != nil {
				msg.GroupID = FormatDiceIDQQGroup(strconv.FormatInt(m.Group.GroupId, 10))
				msg.MessageType = "group"
				msg.GroupName = m.Group.Name
				msg.Sender.GroupRole = m.GroupMember.Role
				msg.Sender.Nickname = m.GroupMember.Nickname
			} else {
				log.Warnf("Received group message without group info: %v", m)
				return
			}
		case "friend":
			if m.Friend != nil {
				msg.MessageType = "private"
				msg.Sender.Nickname = m.Friend.Nickname
			} else {
				log.Warnf("Received friend message without friend info: %v", m)
				return
			}
		default:
			return // 临时对话消息，不处理
		}

		if m.Segments != nil {
			for _, segment := range m.Segments {
				switch seg := segment.(type) {
				case *milky.TextElement:
					log.Debugf(" Text: %s", seg.Text)
					msg.Segment = append(msg.Segment, &types.TextElement{
						Content: seg.Text,
					})
				case *milky.ImageElement:
					log.Debugf(" Image: %s", seg.TempURL)
					msg.Segment = append(msg.Segment, &types.ImageElement{
						URL: seg.TempURL,
					})
				case *milky.AtElement:
					log.Debugf(" At: %d", seg.UserID)
					msg.Segment = append(msg.Segment, &types.AtElement{
						Target: strconv.FormatInt(seg.UserID, 10),
					})
				case *milky.ReplyElement:
					log.Debugf(" Reply to message ID: %d", seg.MessageSeq)
					msg.Segment = append(msg.Segment, &types.ReplyElement{
						ReplySeq: strconv.FormatInt(seg.MessageSeq, 10),
					})
				default:
					log.Debugf("Unknown segment type: %T", segment)
				}
			}
		}

		if len(msg.Segment) == 0 {
			return // 如果没有消息内容，忽略
		}

		// 通过回调处理消息
		if pa.callback != nil {
			info := &MessageSendCallbackInfo{
				Sender: &SimpleUserInfo{
					UserId:   msg.Sender.UserID,
					UserName: msg.Sender.Nickname,
				},
				Message: msg,
			}
			pa.callback.OnMessageReceived(info)
		}
	})

	// 添加戳一戳处理器
	session.AddHandler(func(session2 *milky.Session, m *milky.GroupNudge) {
		if m == nil {
			return
		}
		log.Debugf("Received group nudge: %d -> %d in group %d", m.SenderID, m.ReceiverID, m.GroupID)

		msg := &types.Message{
			Platform:    "QQ",
			MessageType: "group",
			GroupID:     FormatDiceIDQQGroup(strconv.FormatInt(m.GroupID, 10)),
			Time:        time.Now().Unix(),
			Sender: types.SenderBase{
				UserID: FormatDiceIDQQ(strconv.FormatInt(m.SenderID, 10)),
			},
			Segment: []types.IMessageElement{
				&types.PokeElement{
					Target: strconv.FormatInt(m.ReceiverID, 10),
				},
			},
		}

		fmt.Println("戳一戳", msg)
		// 通过回调处理戳一戳
		// TODO: 没有回调，给dice一个单独的回调函数
		// if pa.callback != nil {
		// 	info := &MessageSendCallbackInfo{
		// 		Sender: &SimpleUserInfo{
		// 			UserId:   msg.Sender.UserID,
		// 			UserName: msg.Sender.Nickname,
		// 		},
		// 		Message: msg,
		// 	}
		// 	pa.callback.OnMessageReceived(info)
		// }
	})

	err = session.Open()
	if err != nil {
		log.Errorf("Milky Connect Error:%s", err.Error())
		fmt.Printf("Milky 连接失败: %s", err.Error())
		return 1
	}

	fmt.Println("Milky 连接成功 1")
	pa.isAlive = true
	fmt.Println("Milky 连接成功")
	return 0
}

// DoRelogin 重新登录
func (pa *PlatformAdapterMilky) DoRelogin() bool {
	log := zap.S().Named("adapter")
	if pa.IntentSession == nil {
		success := pa.Serve()
		return success == 0
	}
	_ = pa.IntentSession.Close()
	err := pa.IntentSession.Open()
	if err != nil {
		log.Errorf("Milky Connect Error:%s", err.Error())
		pa.isAlive = false
		return false
	}
	pa.isAlive = true
	return true
}

// SetEnable 设置启用状态
func (pa *PlatformAdapterMilky) SetEnable(enable bool) {
	if enable {
		if !pa.IsAlive() {
			pa.DoRelogin()
		}
	} else {
		if pa.IntentSession != nil {
			_ = pa.IntentSession.Close()
		}
		pa.isAlive = false
	}
}

// ParseMessageToMilky 将消息段转换为 Milky 格式
func ParseMessageToMilky(send []types.IMessageElement) []milky.IMessageElement {
	log := zap.S().Named("adapter")
	var elements []milky.IMessageElement
	for _, elem := range send {
		switch e := elem.(type) {
		case *types.TextElement:
			elements = append(elements, &milky.TextElement{Text: e.Content})
		case *types.ImageElement:
			log.Debugf("Image: %s", e.URL)
			elements = append(elements, &milky.ImageElement{URI: e.URL, Summary: e.File.File, SubType: "normal"})
		case *types.AtElement:
			log.Debugf("At user: %s", e.Target)
			if uid, err := strconv.ParseInt(e.Target, 10, 64); err == nil {
				elements = append(elements, &milky.AtElement{UserID: uid})
			}
		case *types.ReplyElement:
			if seq, err := strconv.ParseInt(e.ReplySeq, 10, 64); err == nil {
				elements = append(elements, &milky.ReplyElement{MessageSeq: seq})
			}
		case *types.RecordElement:
			log.Debugf("Record: %s", e.File.URL)
			elements = append(elements, &milky.RecordElement{URI: e.File.URL})
		case *types.PokeElement:
			continue
		default:
			log.Warnf("Unsupported message element type: %T", elem)
		}
	}
	return elements
}

// MsgSendFileToPerson 发送文件到个人
func (pa *PlatformAdapterMilky) MsgSendFileToPerson(request *MessageSendFileRequest) (bool, error) {
	log := zap.S().Named("adapter")

	// 提取用户ID
	var userIDStr string
	switch v := request.TargetId.(type) {
	case string:
		userIDStr = ExtractQQUserID(v)
	case int64:
		userIDStr = strconv.FormatInt(v, 10)
	default:
		err := fmt.Errorf("invalid user ID type: %T", request.TargetId)
		log.Error(err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	id, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Errorf("Invalid user ID %s: %v", userIDStr, err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	fileName := filepath.Base(request.FilePath)
	ret, err := pa.IntentSession.SendPrivateMessage(id, &[]milky.IMessageElement{
		// 假装是文件吧
		&milky.TextElement{
			Text: "发送文件" + fileName,
		},
	})
	if err != nil {
		log.Errorf("Failed to send file to %s: %v", userIDStr, err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}
	_ = ret
	return true, nil
}

// MsgSendFileToGroup 发送文件到群组
func (pa *PlatformAdapterMilky) MsgSendFileToGroup(request *MessageSendFileRequest) (bool, error) {
	log := zap.S().Named("adapter")

	// 提取群组ID
	var groupIDStr string
	switch v := request.TargetId.(type) {
	case string:
		groupIDStr = ExtractQQGroupID(v)
	case int64:
		groupIDStr = strconv.FormatInt(v, 10)
	default:
		err := fmt.Errorf("invalid group ID type: %T", request.TargetId)
		log.Error(err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	id, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		log.Errorf("Invalid group ID %s: %v", groupIDStr, err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	filename := filepath.Base(request.FilePath)
	path := request.FilePath
	if strings.HasPrefix(path, "files://") {
		path = "file://" + path[len("files://"):]
	}
	_, err = pa.IntentSession.UploadGroupFile(id, path, filename, "")
	if err != nil {
		log.Errorf("Failed to send file to group %s: %v", groupIDStr, err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	// 构造消息对象用于回调
	msg := &types.Message{
		Platform:    "QQ",
		MessageType: "group",
		GroupID:     groupIDStr,
		Message:     fmt.Sprintf("[文件: %s]", filename),
		Sender: types.SenderBase{
			UserID:   request.Sender.UserId,
			Nickname: request.Sender.UserName,
		},
	}

	// 使用回调接口
	if pa.callback != nil {
		info := &MessageSendCallbackInfo{
			Sender:  request.Sender,
			Message: msg,
		}
		pa.callback.OnMessageReceived(info)
	}

	return true, nil
}

// GroupQuit 退出群组
func (pa *PlatformAdapterMilky) GroupQuit(request *GroupOperationQuitRequest) (bool, error) {
	log := zap.S().Named("adapter")

	// 提取群组ID
	var groupIDStr string
	switch v := request.GroupID.(type) {
	case string:
		groupIDStr = ExtractQQGroupID(v)
	case int64:
		groupIDStr = strconv.FormatInt(v, 10)
	default:
		err := fmt.Errorf("invalid group ID type: %T", request.GroupID)
		log.Error(err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		log.Errorf("Invalid group ID %s: %v", groupIDStr, err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	err = pa.IntentSession.QuitGroup(groupID)
	if err != nil {
		log.Errorf("Failed to quit group %s: %v", groupIDStr, err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	log.Infof("Successfully quit group %s", groupIDStr)
	return true, nil
}

// GroupCardNameSet 设置群名片
func (pa *PlatformAdapterMilky) GroupCardNameSet(request *GroupOperationCardNameSetRequest) (bool, error) {
	log := zap.S().Named("adapter")

	// 提取群组ID
	var groupIDStr string
	switch v := request.GroupID.(type) {
	case string:
		groupIDStr = ExtractQQGroupID(v)
	case int64:
		groupIDStr = strconv.FormatInt(v, 10)
	default:
		err := fmt.Errorf("invalid group ID type: %T", request.GroupID)
		log.Error(err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		log.Errorf("Invalid group ID %s: %v", groupIDStr, err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	// 提取用户ID
	var userIDStr string
	switch v := request.UserID.(type) {
	case string:
		userIDStr = ExtractQQUserID(v)
	case int64:
		userIDStr = strconv.FormatInt(v, 10)
	default:
		err := fmt.Errorf("invalid user ID type: %T", request.UserID)
		log.Error(err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Errorf("Invalid user ID %s: %v", userIDStr, err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	err = pa.IntentSession.SetGroupMemberCard(groupID, userID, request.Name)
	if err != nil {
		log.Errorf("Failed to set group card name for %s in group %s: %v", userIDStr, groupIDStr, err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	log.Infof("Successfully set group card name for %s in group %s to %s", userIDStr, groupIDStr, request.Name)
	return true, nil
}

// GroupMemberBan 禁言群成员
func (pa *PlatformAdapterMilky) GroupMemberBan(request *GroupOperationBanRequest) (bool, error) {
	log := zap.S().Named("adapter")

	// 提取群组ID
	var groupIDStr string
	switch v := request.GroupID.(type) {
	case string:
		groupIDStr = ExtractQQGroupID(v)
	case int64:
		groupIDStr = strconv.FormatInt(v, 10)
	default:
		err := fmt.Errorf("invalid group ID type: %T", request.GroupID)
		log.Error(err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		log.Errorf("Invalid group ID %s: %v", groupIDStr, err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	// 提取用户ID
	var userIDStr string
	switch v := request.UserID.(type) {
	case string:
		userIDStr = ExtractQQUserID(v)
	case int64:
		userIDStr = strconv.FormatInt(v, 10)
	default:
		err := fmt.Errorf("invalid user ID type: %T", request.UserID)
		log.Error(err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Errorf("Invalid user ID %s: %v", userIDStr, err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	err = pa.IntentSession.SetGroupMemberMute(groupID, userID, request.Duration)
	if err != nil {
		log.Errorf("Failed to ban group member %s in group %s: %v", userIDStr, groupIDStr, err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	log.Infof("Successfully banned group member %s in group %s for %d seconds", userIDStr, groupIDStr, request.Duration)
	return true, nil
}

// GroupMemberKick 踢出群成员
func (pa *PlatformAdapterMilky) GroupMemberKick(request *GroupOperationKickRequest) (bool, error) {
	log := zap.S().Named("adapter")

	// 提取群组ID
	var groupIDStr string
	switch v := request.GroupID.(type) {
	case string:
		groupIDStr = ExtractQQGroupID(v)
	case int64:
		groupIDStr = strconv.FormatInt(v, 10)
	default:
		err := fmt.Errorf("invalid group ID type: %T", request.GroupID)
		log.Error(err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		log.Errorf("Invalid group ID %s: %v", groupIDStr, err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	// 提取用户ID
	var userIDStr string
	switch v := request.UserID.(type) {
	case string:
		userIDStr = ExtractQQUserID(v)
	case int64:
		userIDStr = strconv.FormatInt(v, 10)
	default:
		err := fmt.Errorf("invalid user ID type: %T", request.UserID)
		log.Error(err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Errorf("Invalid user ID %s: %v", userIDStr, err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	err = pa.IntentSession.KickGroupMember(groupID, userID, false)
	if err != nil {
		log.Errorf("Failed to kick group member %s from group %s: %v", userIDStr, groupIDStr, err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	log.Infof("Successfully kicked group member %s from group %s", userIDStr, groupIDStr)
	return true, nil
}

// MsgEdit 编辑消息
func (pa *PlatformAdapterMilky) MsgEdit(request *MessageOperationRequest) (bool, error) {
	log := zap.S().Named("adapter")

	// Milky SDK 可能不支持编辑消息功能
	err := fmt.Errorf("message editing not supported by Milky SDK")
	log.Warn(err.Error())
	if pa.callback != nil {
		pa.callback.OnError(err)
	}
	return false, err
}

// MsgRecall 撤回消息
func (pa *PlatformAdapterMilky) MsgRecall(request *MessageOperationRequest) (bool, error) {
	log := zap.S().Named("adapter")

	// 解析消息ID
	var messageSeq int64
	switch v := request.MessageID.(type) {
	case int32:
		messageSeq = int64(v)
	case int64:
		messageSeq = v
	case int:
		messageSeq = int64(v)
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			log.Errorf("Invalid message ID %s: %v", v, err)
			if pa.callback != nil {
				pa.callback.OnError(err)
			}
			return false, err
		}
		messageSeq = parsed
	default:
		err := fmt.Errorf("unsupported message ID type: %T", v)
		log.Error(err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	// 注意：这里需要根据具体的消息上下文来判断是群组消息还是私聊消息
	// 由于新接口没有明确区分，这里暂时返回不支持
	err := fmt.Errorf("message recall requires additional context information: messageSeq=%d", messageSeq)
	log.Warn(err.Error())
	if pa.callback != nil {
		pa.callback.OnError(err)
	}
	return false, err
}

// ExtractQQUserID 提取QQ用户ID
func ExtractQQUserID(id string) string {
	if strings.HasPrefix(id, "QQ:") {
		return id[3:]
	}
	return id
}

// ExtractQQGroupID 提取QQ群组ID
func ExtractQQGroupID(id string) string {
	if strings.HasPrefix(id, "QQ-Group:") {
		return id[9:]
	}
	return id
}

// FriendDelete 删除好友
func (pa *PlatformAdapterMilky) FriendDelete(request *FriendOperationRequest) (bool, error) {
	log := zap.S().Named("adapter")

	// 提取用户ID
	var userIDStr string
	switch v := request.UserID.(type) {
	case string:
		userIDStr = ExtractQQUserID(v)
	case int64:
		userIDStr = strconv.FormatInt(v, 10)
	default:
		err := fmt.Errorf("invalid user ID type: %T", request.UserID)
		log.Error(err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	_, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Errorf("Invalid user ID %s: %v", userIDStr, err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	// TODO: 实现删除好友功能
	// Milky SDK 可能不支持删除好友功能
	err = fmt.Errorf("friend deletion not supported by Milky SDK")
	log.Warn(err.Error())
	if pa.callback != nil {
		pa.callback.OnError(err)
	}
	return false, err
}

// FriendAdd 添加好友
func (pa *PlatformAdapterMilky) FriendAdd(request *FriendOperationRequest) (bool, error) {
	log := zap.S().Named("adapter")

	// 提取用户ID
	var userIDStr string
	switch v := request.UserID.(type) {
	case string:
		userIDStr = ExtractQQUserID(v)
	case int64:
		userIDStr = strconv.FormatInt(v, 10)
	default:
		err := fmt.Errorf("invalid user ID type: %T", request.UserID)
		log.Error(err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	_, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Errorf("Invalid user ID %s: %v", userIDStr, err)
		if pa.callback != nil {
			pa.callback.OnError(err)
		}
		return false, err
	}

	// TODO: 实现添加好友功能
	// Milky SDK 可能不支持添加好友功能
	err = fmt.Errorf("friend addition not supported by Milky SDK")
	log.Warn(err.Error())
	if pa.callback != nil {
		pa.callback.OnError(err)
	}
	return false, err
}
