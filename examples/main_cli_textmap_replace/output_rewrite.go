package main

import (
	"github.com/sealdice/smallseal/dice"
	"github.com/sealdice/smallseal/dice/types"
)

// 输出重写文本模板集合 - 在输出时通过 CommandFormatInfo 还原变量后重新渲染
// 这些不影响计算，只是用不同方式呈现相同的数据

// 角色1: 极简输出 - 只保留核心数据
func buildOutputRewriteMinimal() types.TextTemplateWithWeightDict {
	cloned := cloneTextTemplateDict(dice.DefaultTextMap)
	core := ensureCategory(cloned, "核心")
	coc := ensureCategory(cloned, "COC")

	// 核心骰点
	core["骰点"] = []types.TextTemplateItem{
		{"{$t原因句子}{$t玩家}{$t代骰信息}投掷结果：{$t结果文本}", 1},
	}
	core["骰点_单项结果文本"] = []types.TextTemplateItem{
		{`{$t表达式文本}{$t计算过程}={$t计算结果}`, 1},
	}
	core["骰点_多轮"] = []types.TextTemplateItem{
		{`{$t原因句子}{$t玩家}{$t代骰信息}投掷{$t次数}次:\n{$t结果文本}`, 1},
	}

	// COC检定
	coc["检定"] = []types.TextTemplateItem{
		{`{$t原因 ? '因' + $t原因 + '，'}{$t玩家}的"{$t属性表达式文本}"检定：{$t结果文本}`, 1},
	}
	coc["检定_单项结果文本"] = []types.TextTemplateItem{
		{`{$t检定表达式文本}={$t骰子出目}/{$t判定值}{$t检定计算过程} {$t判定结果}`, 1},
	}
	coc["检定_多轮"] = []types.TextTemplateItem{
		{`{$t玩家}的"{$t属性表达式文本}"共检定{$t次数}次:\n{$t结果文本}`, 1},
	}

	// COC判定结果 - 简短版
	coc["判定_大失败"] = []types.TextTemplateItem{
		{"大失败", 1},
	}
	coc["判定_失败"] = []types.TextTemplateItem{
		{"失败", 1},
	}
	coc["判定_成功"] = []types.TextTemplateItem{
		{"成功", 1},
	}
	coc["判定_成功_困难"] = []types.TextTemplateItem{
		{"困难成功", 1},
	}
	coc["判定_成功_极难"] = []types.TextTemplateItem{
		{"极难成功", 1},
	}
	coc["判定_大成功"] = []types.TextTemplateItem{
		{"大成功", 1},
	}

	return cloned
}

// 角色2: 女仆 - 礼貌简洁
func buildOutputRewriteMaid() types.TextTemplateWithWeightDict {
	cloned := cloneTextTemplateDict(dice.DefaultTextMap)
	core := ensureCategory(cloned, "核心")
	coc := ensureCategory(cloned, "COC")

	// 核心骰点
	core["骰点"] = []types.TextTemplateItem{
		{"{$t原因句子}{$t玩家}{$t代骰信息}掷出的结果为 {$t结果文本}，请您过目", 1},
		{"{$t原因句子}{$t玩家}{$t代骰信息}的投骰结果是 {$t结果文本}，已为您准备好了", 1},
	}
	core["骰点_单项结果文本"] = []types.TextTemplateItem{
		{`{$t表达式文本}{$t计算过程}，经计算得出{$t计算结果}`, 1},
	}
	core["骰点_多轮"] = []types.TextTemplateItem{
		{`{$t原因句子}{$t玩家}{$t代骰信息}投掷{$t次数}次的结果如下：\n{$t结果文本}\n谨此呈上，请您过目`, 1},
	}

	// COC检定
	coc["检定"] = []types.TextTemplateItem{
		{`{$t原因 ? '由于' + $t原因 + '，'}{$t玩家}的"{$t属性表达式文本}"检定结果为：{$t结果文本}，请您确认`, 1},
		{`为{$t玩家}进行了{$t原因 ? $t原因 : ''}"{$t属性表达式文本}"检定：{$t结果文本}，已妥善记录`, 1},
	}
	coc["检定_单项结果文本"] = []types.TextTemplateItem{
		{`{$t检定表达式文本}={$t骰子出目}/{$t判定值}{$t检定计算过程}，判定为{$t判定结果}`, 1},
	}
	coc["检定_多轮"] = []types.TextTemplateItem{
		{`对{$t玩家}的"{$t属性表达式文本}"进行了{$t次数}次检定，结果如下：\n{$t结果文本}\n以上是全部检定结果`, 1},
	}

	// COC判定结果 - 优雅版
	coc["判定_大失败"] = []types.TextTemplateItem{
		{"大失败...请您节哀", 1},
		{"很遗憾，这是大失败", 1},
	}
	coc["判定_失败"] = []types.TextTemplateItem{
		{"失败了，不过请不要气馁", 1},
		{"未能成功，还请见谅", 1},
	}
	coc["判定_成功"] = []types.TextTemplateItem{
		{"成功了，恭喜您", 1},
		{"判定成功，做得很好呢", 1},
	}
	coc["判定_成功_困难"] = []types.TextTemplateItem{
		{"困难成功！相当出色", 1},
		{"成功跨越了困难难度，令人钦佩", 1},
	}
	coc["判定_成功_极难"] = []types.TextTemplateItem{
		{"极难成功！这是非凡的表现", 1},
		{"突破极难难度，实在是太精彩了", 1},
	}
	coc["判定_大成功"] = []types.TextTemplateItem{
		{"大成功！请容我向您致以敬意", 1},
		{"大成功！这是命运的眷顾呢", 1},
	}

	return cloned
}

// 角色3: 猫娘 - 可爱简短
func buildOutputRewriteNekomimi() types.TextTemplateWithWeightDict {
	cloned := cloneTextTemplateDict(dice.DefaultTextMap)
	core := ensureCategory(cloned, "核心")
	coc := ensureCategory(cloned, "COC")

	// 核心骰点
	core["骰点"] = []types.TextTemplateItem{
		{"{$t原因句子}{$t玩家}{$t代骰信息}掷出了 {$t结果文本} 喵～", 1},
		{"{$t原因句子}{$t玩家}{$t代骰信息}投骰结果是 {$t结果文本} 喵！", 1},
	}
	core["骰点_单项结果文本"] = []types.TextTemplateItem{
		{`{$t表达式文本}{$t计算过程}...是{$t计算结果}喵！`, 1},
	}
	core["骰点_多轮"] = []types.TextTemplateItem{
		{`{$t原因句子}{$t玩家}{$t代骰信息}要投{$t次数}次喵？好多喵～\n{$t结果文本}\n都投完啦喵！`, 1},
		{`{$t原因句子}{$t玩家}{$t代骰信息}要投这么多次({$t次数}次)呀喵！\n{$t结果文本}\n呼...累死人家了喵～`, 1},
	}

	// COC检定
	coc["检定"] = []types.TextTemplateItem{
		{`{$t原因 ? $t原因 + '喵？'}{$t玩家}的"{$t属性表达式文本}"检定结果：{$t结果文本} 喵～`, 1},
		{`帮{$t玩家}检定了{$t原因 ? $t原因 + '的' : ''}"{$t属性表达式文本}"喵！结果是 {$t结果文本} 喵！`, 1},
	}
	coc["检定_单项结果文本"] = []types.TextTemplateItem{
		{`{$t检定表达式文本}={$t骰子出目}/{$t判定值}{$t检定计算过程}，{$t判定结果}喵`, 1},
	}
	coc["检定_多轮"] = []types.TextTemplateItem{
		{`{$t玩家}的"{$t属性表达式文本}"要检定{$t次数}次喵！\n{$t结果文本}\n全部搞定喵～`, 1},
	}

	// COC判定结果 - 猫娘版
	coc["判定_大失败"] = []types.TextTemplateItem{
		{"大失败喵...好可惜喵...", 1},
		{"大失败了喵！要抱抱安慰一下吗喵？", 1},
		{"居然大失败了喵！呜呜喵...", 1},
	}
	coc["判定_失败"] = []types.TextTemplateItem{
		{"失败了喵...不过没关系的喵", 1},
		{"没成功喵～下次会更好的喵！", 1},
		{"失败喵...要不要再试一次喵？", 1},
	}
	coc["判定_成功"] = []types.TextTemplateItem{
		{"成功啦喵！好厉害喵～", 1},
		{"成功了喵！喵呜～", 1},
		{"检定通过喵！干得漂亮喵！", 1},
	}
	coc["判定_成功_困难"] = []types.TextTemplateItem{
		{"困难成功喵！太棒啦喵～！", 1},
		{"哇！困难都成功了喵！好强喵！", 1},
		{"困难成功喵！人家都惊呆了喵！", 1},
	}
	coc["判定_成功_极难"] = []types.TextTemplateItem{
		{"极难成功喵喵喵！这也太厉害了喵！！", 1},
		{"天啊喵！极难都成功了喵！简直是奇迹喵！", 1},
		{"极难成功喵！人家要给你鼓掌喵～啪啪啪！", 1},
	}
	coc["判定_大成功"] = []types.TextTemplateItem{
		{"大成功喵喵喵！！幸运爆棚喵～！", 1},
		{"哇！大成功喵！运气好到爆炸喵！", 1},
		{"大成功喵！这手气人家都嫉妒啦喵～", 1},
	}

	return cloned
}

// 所有输出重写风格及其名称
var outputRewriteStyles = []struct {
	Name    string
	TextMap types.TextTemplateWithWeightDict
}{
	{"[重写1:极简]", buildOutputRewriteMinimal()},
	{"[重写2:女仆]", buildOutputRewriteMaid()},
	{"[重写3:猫娘]", buildOutputRewriteNekomimi()},
}
