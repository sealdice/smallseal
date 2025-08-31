package utils

import "strings"

// UnpackGroupUserId 分离群组ID和用户ID的连写
func UnpackGroupUserId(id string) (groupIdPart, userIdPart string, ok bool) {
	prefixMap := map[string]string{
		"QQ-Group:":         "QQ:",
		"QQ-CH-Group:":      "QQ-CH:", // QQ-CH-Group:33845581638279358-3047284-QQ-CH:144115218744389299
		"KOOK-CH-Group:":    "KOOK:",
		"DODO-Group:":       "DODO:",
		"TG-Group:":         "TG:",
		"DISCORD-CH-Group:": "DISCORD:",
		"SEALCHAT-Group:":   "SEALCHAT:",
		"SLACK-CH-Group":    "SLACK:",
		"DINGTALK-Group":    "DINGTALK:",
		"OpenQQ-Group-T:":   "OpenQQ-Member-T:",
		"UI-Group:":         "UI:",
	}

	// 我们是否能做这样的假设：XX-Group:123 的用户ID形式是 XX:456 ？
	// 但是 KOOK 就例外了
	for groupPrefix, userPrefix := range prefixMap {
		if strings.HasPrefix(id, groupPrefix) {
			lenX := len(groupPrefix)

			idx := strings.Index(id[lenX:], "-"+userPrefix)
			if idx != -1 {
				idx += lenX
				return id[:idx], id[idx+1:], true
			}
		}
	}

	prefixMap2 := map[string]string{
		"PG-QQ:":       "QQ:",
		"PG-KOOK:":     "KOOK:",
		"PG-SEALCHAT:": "SEALCHAT:", // PG-SEALCHAT:LJMmncRk4Kh_yPNjCNIhX:xK7iDQ13ypAw7GHiGBzHM
		"PG-UI:":       "UI:",
		"PG-DISCORD:":  "DISCORD:",
		"PG-DODO:":     "DODO:",
		"PG-SLACK:":    "SLACK:",
		"PG-TG:":       "TG:",
		"PG-DINGTALK:": "DINGTALK:",
	}
	for userPrivateGroupPrefix, userPrefix := range prefixMap2 {
		if strings.HasPrefix(id, userPrivateGroupPrefix) {
			lenX := len(userPrivateGroupPrefix)
			idx := strings.Index(id[lenX:], "-"+userPrefix)
			if idx != -1 {
				idx += lenX
				return id[:idx], id[idx+1:], true
			}
		}
	}

	return "", "", false
}
