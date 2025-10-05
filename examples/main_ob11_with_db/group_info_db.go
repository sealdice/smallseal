package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/sealdice/smallseal/dice/types"
	"github.com/sealdice/smallseal/utils"
	"github.com/tidwall/buntdb"
)

type buntGroupInfoManager struct {
	db    *buntdb.DB
	cache utils.SyncMap[string, *types.GroupInfo]
}

func newBuntGroupInfoManager(db *buntdb.DB) *buntGroupInfoManager {
	return &buntGroupInfoManager{db: db}
}

type storedGroupInfo struct {
	GroupId             string
	GuildID             string
	ChannelID           string
	GroupName           string
	Active              bool
	System              string
	DiceSideExpr        string
	HelpPackages        []string
	ExtListSnapshot     []string
	ExtActiveStates     map[string]bool
	DiceIDActiveMap     map[string]bool
	DiceIDExistsMap     map[string]bool
	BotList             map[string]bool
	PlayerGroups        map[string][]string
	CocRuleIndex        int
	LogCurName          string
	LogOn               bool
	ShowGroupWelcome    bool
	GroupWelcomeMessage string
	EnteredTime         int64
	InviteUserID        string
	DefaultHelpGroup    string
	RecentDiceSendTime  int64
	UpdatedAtTime       int64
}

func groupKey(groupId string) string {
	return "group:" + encodeKeyPart(groupId)
}

func groupInfoToStored(info *types.GroupInfo) *storedGroupInfo {
	if info == nil {
		return nil
	}
	stored := &storedGroupInfo{
		GroupId:             info.GroupId,
		GuildID:             info.GuildID,
		ChannelID:           info.ChannelID,
		GroupName:           info.GroupName,
		Active:              info.Active,
		System:              info.System,
		DiceSideExpr:        info.DiceSideExpr,
		HelpPackages:        append([]string{}, info.HelpPackages...),
		ExtListSnapshot:     append([]string{}, info.ExtListSnapshot...),
		CocRuleIndex:        info.CocRuleIndex,
		LogCurName:          info.LogCurName,
		LogOn:               info.LogOn,
		ShowGroupWelcome:    info.ShowGroupWelcome,
		GroupWelcomeMessage: info.GroupWelcomeMessage,
		EnteredTime:         info.EnteredTime,
		InviteUserID:        info.InviteUserID,
		DefaultHelpGroup:    info.DefaultHelpGroup,
		RecentDiceSendTime:  info.RecentDiceSendTime,
		UpdatedAtTime:       time.Now().Unix(),
	}
	stored.ExtActiveStates = syncMapToBoolMap(info.ExtActiveStates)
	stored.DiceIDActiveMap = syncMapToBoolMap(info.DiceIDActiveMap)
	stored.DiceIDExistsMap = syncMapToBoolMap(info.DiceIDExistsMap)
	stored.BotList = syncMapToBoolMap(info.BotList)
	stored.PlayerGroups = syncMapToSliceMap(info.PlayerGroups)
	return stored
}

func storedToGroupInfo(stored *storedGroupInfo) *types.GroupInfo {
	if stored == nil {
		return nil
	}
	info := &types.GroupInfo{
		GroupId:             stored.GroupId,
		GuildID:             stored.GuildID,
		ChannelID:           stored.ChannelID,
		GroupName:           stored.GroupName,
		Active:              stored.Active,
		System:              stored.System,
		DiceSideExpr:        stored.DiceSideExpr,
		HelpPackages:        append([]string{}, stored.HelpPackages...),
		ExtListSnapshot:     append([]string{}, stored.ExtListSnapshot...),
		CocRuleIndex:        stored.CocRuleIndex,
		LogCurName:          stored.LogCurName,
		LogOn:               stored.LogOn,
		ShowGroupWelcome:    stored.ShowGroupWelcome,
		GroupWelcomeMessage: stored.GroupWelcomeMessage,
		EnteredTime:         stored.EnteredTime,
		InviteUserID:        stored.InviteUserID,
		DefaultHelpGroup:    stored.DefaultHelpGroup,
		RecentDiceSendTime:  stored.RecentDiceSendTime,
		UpdatedAtTime:       stored.UpdatedAtTime,
		ExtActiveStates:     &utils.SyncMap[string, bool]{},
		DiceIDActiveMap:     &utils.SyncMap[string, bool]{},
		DiceIDExistsMap:     &utils.SyncMap[string, bool]{},
		BotList:             &utils.SyncMap[string, bool]{},
		Players:             &utils.SyncMap[string, *types.GroupPlayerInfo]{},
		PlayerGroups:        &utils.SyncMap[string, []string]{},
		ActivatedExtList:    []*types.ExtInfo{},
	}
	restoreBoolSyncMap(info.ExtActiveStates, stored.ExtActiveStates)
	restoreBoolSyncMap(info.DiceIDActiveMap, stored.DiceIDActiveMap)
	restoreBoolSyncMap(info.DiceIDExistsMap, stored.DiceIDExistsMap)
	restoreBoolSyncMap(info.BotList, stored.BotList)
	restoreSliceSyncMap(info.PlayerGroups, stored.PlayerGroups)
	return info
}

func syncMapToBoolMap(m *utils.SyncMap[string, bool]) map[string]bool {
	if m == nil {
		return nil
	}
	out := make(map[string]bool)
	m.Range(func(key string, value bool) bool {
		out[key] = value
		return true
	})
	return out
}

func syncMapToSliceMap(m *utils.SyncMap[string, []string]) map[string][]string {
	if m == nil {
		return nil
	}
	out := make(map[string][]string)
	m.Range(func(key string, value []string) bool {
		out[key] = append([]string{}, value...)
		return true
	})
	return out
}

func restoreBoolSyncMap(m *utils.SyncMap[string, bool], data map[string]bool) {
	if m == nil || data == nil {
		return
	}
	for key, value := range data {
		m.Store(key, value)
	}
}

func restoreSliceSyncMap(m *utils.SyncMap[string, []string], data map[string][]string) {
	if m == nil || data == nil {
		return
	}
	for key, value := range data {
		m.Store(key, append([]string{}, value...))
	}
}

func (m *buntGroupInfoManager) Load(groupId string) (*types.GroupInfo, bool) {
	if groupId == "" {
		return nil, false
	}
	if cached, ok := m.cache.Load(groupId); ok {
		return cached, true
	}
	var loaded *types.GroupInfo
	err := m.db.View(func(tx *buntdb.Tx) error {
		value, err := tx.Get(groupKey(groupId))
		if err != nil {
			if errors.Is(err, buntdb.ErrNotFound) {
				return nil
			}
			return err
		}
		var stored storedGroupInfo
		if err := json.Unmarshal([]byte(value), &stored); err != nil {
			return err
		}
		loaded = storedToGroupInfo(&stored)
		return nil
	})
	if err != nil {
		fmt.Printf("GroupInfo load error: %v\n", err)
		return nil, false
	}
	if loaded == nil {
		return nil, false
	}
	m.cache.Store(groupId, loaded)
	return loaded, true
}

func (m *buntGroupInfoManager) Store(groupId string, info *types.GroupInfo) {
	if groupId == "" || info == nil {
		return
	}
	stored := groupInfoToStored(info)
	if stored == nil {
		return
	}
	payload, err := json.Marshal(stored)
	if err != nil {
		fmt.Printf("GroupInfo marshal error: %v\n", err)
		return
	}
	err = m.db.Update(func(tx *buntdb.Tx) error {
		_, _, e := tx.Set(groupKey(groupId), string(payload), nil)
		return e
	})
	if err != nil {
		fmt.Printf("GroupInfo store error: %v\n", err)
		return
	}
	m.cache.Store(groupId, info)
}

func (m *buntGroupInfoManager) Delete(groupId string) {
	if groupId == "" {
		return
	}
	m.cache.Delete(groupId)
	err := m.db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(groupKey(groupId))
		if err != nil && !errors.Is(err, buntdb.ErrNotFound) {
			return err
		}
		return nil
	})
	if err != nil {
		fmt.Printf("GroupInfo delete error: %v\n", err)
	}
}
