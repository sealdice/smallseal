package dice

import (
	"os"
	"testing"

	"github.com/sealdice/smallseal/dice/types"
)

type trackingGroupInfoManager struct {
	inner      GroupInfoManager
	storeCalls int
	lastID     string
	lastInfo   *types.GroupInfo
}

func newTrackingGroupInfoManager() *trackingGroupInfoManager {
	return &trackingGroupInfoManager{inner: NewDefaultGroupInfoManager()}
}

func (t *trackingGroupInfoManager) Load(groupID string) (*types.GroupInfo, bool) {
	return t.inner.Load(groupID)
}

func (t *trackingGroupInfoManager) Store(groupID string, info *types.GroupInfo) {
	t.storeCalls++
	t.lastID = groupID
	t.lastInfo = info
	t.inner.Store(groupID, info)
}

func (t *trackingGroupInfoManager) Delete(groupID string) {
	t.inner.Delete(groupID)
}

func TestBotOffDisablesGroupCommands(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		_ = os.Chdir(cwd)
	}()
	if err := os.Chdir(".."); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	d := NewDice()
	tracker := newTrackingGroupInfoManager()
	d.GroupInfoManager = tracker

	var replies []string
	d.CallbackForSendMsg.Store("test", func(msg *types.MsgToReply) {
		replies = append(replies, msg.Segments.ToText())
	})

	send := func(content string) {
		msg := &types.Message{
			MessageType: "group",
			GroupID:     "QQ-Group:12345",
			Sender: types.SenderBase{
				UserID:   "user",
				Nickname: "tester",
			},
			Platform: "test",
			Segments: types.MessageSegments{&types.TextElement{Content: content}},
		}
		d.Execute("test", msg)
	}

	send(".roll 1d20")
	if len(replies) == 0 {
		t.Fatalf("expected roll to reply before bot off")
	}
	if tracker.storeCalls != 1 {
		t.Fatalf("expected single store on first message, got %d", tracker.storeCalls)
	}

	send(".bot off")

	groupInfo, ok := d.GroupInfoManager.Load("QQ-Group:12345")
	if !ok {
		t.Fatalf("group info not found after bot off")
	}
	if groupInfo.Active {
		t.Fatalf("expected group to be inactive after bot off")
	}
	if tracker.storeCalls < 2 {
		t.Fatalf("expected bot off to persist group state, store calls=%d", tracker.storeCalls)
	}
	if tracker.lastID != "QQ-Group:12345" {
		t.Fatalf("unexpected persisted group id %q", tracker.lastID)
	}
	if tracker.lastInfo == nil || tracker.lastInfo.Active {
		t.Fatalf("expected persisted group state to be inactive")
	}

	before := len(replies)
	send(".roll 1d20")
	if len(replies) != before {
		t.Fatalf("roll should not reply when bot is off")
	}
	if tracker.storeCalls != 2 {
		t.Fatalf("unexpected extra store after blocked command, got %d", tracker.storeCalls)
	}
}
