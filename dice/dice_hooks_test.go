package dice

import (
	"testing"

	"github.com/sealdice/smallseal/dice/types"
	"github.com/stretchr/testify/assert"
)

func TestRegisterAndUnregisterMessageInHook(t *testing.T) {
	var d Dice
	as := assert.New(t)

	callCount := 0
	hook := func(dl types.DiceLike, adapterID string, msg *types.Message, ctx *types.MsgContext) types.HookResult {
		callCount++
		as.Equal("adapter", adapterID, "unexpected adapterID")
		return types.HookResultContinue
	}

	handle, err := d.RegisterMessageInHook("test", types.HookPriorityNormal, hook)
	as.NoError(err)

	aborted := d.runMessageInHooks("adapter", &types.Message{}, &types.MsgContext{})
	as.False(aborted, "runMessageInHooks should not abort for continue result")
	as.Equal(1, callCount, "hook should run exactly once")

	as.True(d.UnregisterMessageInHook(handle), "expected unregister to succeed")
	as.False(d.UnregisterMessageInHook(handle), "unregistering twice should fail")

	callCount = 0
	d.runMessageInHooks("adapter", &types.Message{}, &types.MsgContext{})
	as.Zero(callCount, "hook should not fire after unregistration")
}

func TestRegisterMessageInHookPriorityOrder(t *testing.T) {
	var d Dice
	as := assert.New(t)
	order := []string{}

	makeHook := func(tag string) types.MessageInHook {
		return func(types.DiceLike, string, *types.Message, *types.MsgContext) types.HookResult {
			order = append(order, tag)
			return types.HookResultContinue
		}
	}

	_, err := d.RegisterMessageInHook("low", types.HookPriorityLow, makeHook("low"))
	as.NoError(err)
	_, err = d.RegisterMessageInHook("high", types.HookPriorityHigh, makeHook("high"))
	as.NoError(err)
	_, err = d.RegisterMessageInHook("normal", types.HookPriorityNormal, makeHook("normal"))
	as.NoError(err)

	d.runMessageInHooks("adapter", &types.Message{}, &types.MsgContext{})

	expected := []string{"high", "normal", "low"}
	as.Equal(expected, order)
}

func TestRegisterAndUnregisterMessageOutHook(t *testing.T) {
	var d Dice
	as := assert.New(t)

	callCount := 0
	hook := func(dl types.DiceLike, adapterID string, reply *types.MsgToReply) types.HookResult {
		callCount++
		as.Equal("adapter", adapterID, "unexpected adapterID")
		return types.HookResultContinue
	}

	handle, err := d.RegisterMessageOutHook("test", types.HookPriorityNormal, hook)
	as.NoError(err)

	aborted := d.runMessageOutHooks("adapter", &types.MsgToReply{})
	as.False(aborted, "runMessageOutHooks should not abort for continue result")
	as.Equal(1, callCount, "hook should run exactly once")

	as.True(d.UnregisterMessageOutHook(handle), "expected unregister to succeed")
	as.False(d.UnregisterMessageOutHook(handle), "unregistering twice should fail")

	callCount = 0
	d.runMessageOutHooks("adapter", &types.MsgToReply{})
	as.Zero(callCount, "hook should not fire after unregistration")
}

func TestRegisterMessageOutHookPriorityOrder(t *testing.T) {
	var d Dice
	as := assert.New(t)
	order := []string{}

	makeHook := func(tag string) types.MessageOutHook {
		return func(types.DiceLike, string, *types.MsgToReply) types.HookResult {
			order = append(order, tag)
			return types.HookResultContinue
		}
	}

	_, err := d.RegisterMessageOutHook("low", types.HookPriorityLow, makeHook("low"))
	as.NoError(err)
	_, err = d.RegisterMessageOutHook("high", types.HookPriorityHigh, makeHook("high"))
	as.NoError(err)
	_, err = d.RegisterMessageOutHook("normal", types.HookPriorityNormal, makeHook("normal"))
	as.NoError(err)

	d.runMessageOutHooks("adapter", &types.MsgToReply{})

	expected := []string{"high", "normal", "low"}
	as.Equal(expected, order)
}
