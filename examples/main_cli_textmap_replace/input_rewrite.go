package main

import (
	"github.com/sealdice/smallseal/dice"
	"github.com/sealdice/smallseal/dice/types"
)

// 输入重写文本模板集合 - 在消息进入时替换原始模板
// 这些会影响实际的计算和显示逻辑

// 风格0: 原版
func buildInputRewriteStyle0() types.TextTemplateWithWeightDict {
	return cloneTextTemplateDict(dice.DefaultTextMap)
}

// 风格1: 正式书面语
func buildInputRewriteStyle1() types.TextTemplateWithWeightDict {
	cloned := cloneTextTemplateDict(dice.DefaultTextMap)
	core := ensureCategory(cloned, "核心")

	core["骰点"] = []types.TextTemplateItem{
		{"{$t原因句子}{$t玩家}{$t代骰信息}的投掷结果为 {$t结果文本}", 1},
	}
	core["骰点_单项结果文本"] = []types.TextTemplateItem{
		{`{$t表达式文本}{$t计算过程} => {$t计算结果}`, 1},
	}
	core["骰点_多轮"] = []types.TextTemplateItem{
		{`{$t原因句子}{$t玩家}{$t代骰信息}进行了{$t次数}轮投掷:\n{$t结果文本}`, 1},
	}

	return cloned
}

// 风格2: 口语化表达
func buildInputRewriteStyle2() types.TextTemplateWithWeightDict {
	cloned := cloneTextTemplateDict(dice.DefaultTextMap)
	core := ensureCategory(cloned, "核心")

	core["骰点"] = []types.TextTemplateItem{
		{"{$t原因句子}{$t玩家}{$t代骰信息}摇出了 {$t结果文本}", 1},
	}
	core["骰点_单项结果文本"] = []types.TextTemplateItem{
		{`{$t表达式文本}{$t计算过程}，得到{$t计算结果}`, 1},
	}
	core["骰点_多轮"] = []types.TextTemplateItem{
		{`{$t原因句子}{$t玩家}{$t代骰信息}连续摇了{$t次数}次骰子:\n{$t结果文本}`, 1},
	}

	return cloned
}

// 所有输入重写风格
var inputRewriteStyles = []types.TextTemplateWithWeightDict{
	buildInputRewriteStyle0(),
	buildInputRewriteStyle1(),
	buildInputRewriteStyle2(),
}
