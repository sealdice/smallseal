# TextMap 重写 Demo

这个示例展示了两种不同的文本重写机制，帮助理解小海豹的文本模板系统，并演示了如何通过自定义文本模板创建具有不同性格的骰子输出。

或许可称为 Persona 机制。

输入重写应该能兼容全部的指令，但目前还不能确定所有的指令都能被输出重写所覆盖。

## 文件结构

```
main_cli_textmap/
├── main.go                   # 主程序入口，注册 Hook
├── output_rewrite_tools.go   # 通用工具函数（克隆、重建等）
├── input_rewrite.go          # 输入重写文本模板（2种风格）
├── output_rewrite.go         # 输出重写文本模板（3种角色）
└── README.md                 # 项目说明文档
```

## 两种重写原理

### 1. 输入重写（Input Rewrite）
**时机**: 消息进入系统时（MessageInHook）  
**原理**: 替换 `MsgContext.TextTemplateMap`，影响后续的计算和格式化  
**文件**: `input_rewrite.go`  
**风格数**: 2种

- **风格1**: 正式书面语
  - "投掷结果为"
  - "进行了X轮投掷"

- **风格2**: 口语化表达
  - "摇出了"
  - "连续摇了X次骰子"

**特点**: 
- 直接影响原始输出
- 每次随机选择一种风格
- 修改的是"源头"，后续所有渲染都使用这个模板

### 2. 输出重写（Output Rewrite）
**时机**: 消息输出时（MessageOutHook）  
**原理**: 通过 `CommandFormatInfo.LoadRecords` 还原变量状态，用新模板重新渲染  
**文件**: `output_rewrite.go`  
**角色数**: 3种，每种都有独特的语言风格

#### 角色1: 极简输出
简洁直白的表达风格，不带任何修饰：
- 骰点: "投掷结果：D100=62"
- 检定: "因调查，CLI的"图书馆"检定：D100=45/60 成功"
- 判定: 只保留核心词汇（成功/失败/大成功等）

#### 角色2: 女仆
礼貌得体的服务性语言，每个判定有2个随机变体：
- 骰点: "掷出的结果为 D100=62，请您过目" / "的投骰结果是 D100=62，已为您准备好了"
- 检定: "由于调查，CLI的"图书馆"检定结果为：D100=45/60，判定为成功了，恭喜您，请您确认"
- 判定大失败: "大失败...请您节哀" / "很遗憾，这是大失败"
- 判定成功: "成功了，恭喜您" / "判定成功，做得很好呢"
- 判定困难成功: "困难成功！相当出色" / "成功跨越了困难难度，令人钦佩"
- 判定极难成功: "极难成功！这是非凡的表现" / "突破极难难度，实在是太精彩了"
- 判定大成功: "大成功！请容我向您致以敬意" / "大成功！这是命运的眷顾呢"

#### 角色3: 猫娘
活泼可爱的表达，大量使用"喵"，每个判定有3个随机变体：
- 骰点: "掷出了 D100=62 喵～" / "投骰结果是 D100=62 喵！"
- 检定: "调查喵？CLI的"图书馆"检定结果：D100=45/60，成功啦喵！好厉害喵～ 喵～"
- 判定大失败: "大失败喵...好可惜喵..." / "大失败了喵！要抱抱安慰一下吗喵？" / "居然大失败了喵！呜呜喵..."
- 判定失败: "失败了喵...不过没关系的喵" / "没成功喵～下次会更好的喵！" / "失败喵...要不要再试一次喵？"
- 判定成功: "成功啦喵！好厉害喵～" / "成功了喵！喵呜～" / "检定通过喵！干得漂亮喵！"
- 判定困难成功: "困难成功喵！太棒啦喵～！" / "哇！困难都成功了喵！好强喵！" / "困难成功喵！人家都惊呆了喵！"
- 判定极难成功: "极难成功喵喵喵！这也太厉害了喵！！" / "天啊喵！极难都成功了喵！简直是奇迹喵！" / "极难成功喵！人家要给你鼓掌喵～啪啪啪！"
- 判定大成功: "大成功喵喵喵！！幸运爆棚喵～！" / "哇！大成功喵！运气好到爆炸喵！" / "大成功喵！这手气人家都嫉妒啦喵～"

**特点**: 
- 不影响原始计算，只改变输出表达
- 同时展示3种角色的不同表达方式
- 支持核心骰点和 COC 检定两大模板类别
- 展示"同样的数据，不同性格的表达"

## 运行方式

```bash
# 编译
go build -o textmap_demo.exe .

# 运行
./textmap_demo.exe

# 或直接运行
go run .
```

## 使用示例

启动后输入骰点命令：

```
>>> .r
>>> .r d100
>>> .r d20+5 攻击
>>> .r 3#d6
>>> .ra 图书馆 60 调查
```

## 输出示例

### 示例1: 简单骰点
```
>>> .r d100

[原始输出]
CLI 投掷结果为 D100 => 62

[重写1:极简]
CLI投掷结果：D100=62

[重写2:女仆]
CLI掷出的结果为 D100=62，请您过目

[重写3:猫娘]
CLI掷出了 D100=62 喵～
================
```

### 示例2: COC检定
```
>>> .ra 图书馆 60 调查

[原始输出]
由于调查，CLI的"图书馆"检定结果为: D100=45/60 成功

[重写1:极简]
因调查，CLI的"图书馆"检定：D100=45/60 成功

[重写2:女仆]
由于调查，CLI的"图书馆"检定结果为：D100=45/60，判定为成功了，恭喜您，请您确认

[重写3:猫娘]
调查喵？CLI的"图书馆"检定结果：D100=45/60，成功啦喵！好厉害喵～ 喵～
================
```

### 示例3: 多轮骰点
```
>>> .r 3#d6

[原始输出]
CLI 投掷3次:
D6=4
D6=2
D6=5

[重写1:极简]
CLI投掷3次:
D6=4
D6=2
D6=5

[重写2:女仆]
CLI投掷3次的结果如下：
D6=4
D6=2
D6=5
谨此呈上，请您过目

[重写3:猫娘]
CLI要投3次喵？好多喵～
D6=4
D6=2
D6=5
都投完啦喵！
================
```

## 扩展方法

### 添加新的输入重写风格

在 `input_rewrite.go` 中添加：

```go
func buildInputRewriteStyle3() types.TextTemplateWithWeightDict {
    cloned := cloneTextTemplateDict(dice.DefaultTextMap)
    core := ensureCategory(cloned, "核心")

    core["骰点"] = []types.TextTemplateItem{
        {"{$t玩家}投出了 {$t结果文本}", 1},
    }
    // ... 其他模板

    return cloned
}

// 添加到数组
var inputRewriteStyles = []types.TextTemplateWithWeightDict{
    buildInputRewriteStyle1(),
    buildInputRewriteStyle2(),
    buildInputRewriteStyle3(), // 新增
}
```

### 添加新的输出重写角色

在 `output_rewrite.go` 中添加新角色，需要实现以下模板：

```go
func buildOutputRewriteRobot() types.TextTemplateWithWeightDict {
    cloned := cloneTextTemplateDict(dice.DefaultTextMap)
    core := ensureCategory(cloned, "核心")
    coc := ensureCategory(cloned, "COC")

    // 核心骰点模板
    core["骰点"] = []types.TextTemplateItem{
        {"[系统] {$t原因句子}{$t玩家}{$t代骰信息}执行投掷 >> {$t结果文本}", 1},
    }
    core["骰点_单项结果文本"] = []types.TextTemplateItem{
        {"{$t表达式文本}{$t计算过程}={$t计算结果}", 1},
    }
    core["骰点_多轮"] = []types.TextTemplateItem{
        {"[系统] {$t原因句子}{$t玩家}{$t代骰信息}执行{$t次数}次投掷：\n{$t结果文本}", 1},
    }

    // COC检定模板
    coc["检定"] = []types.TextTemplateItem{
        {"[检定] {$t原因 ? $t原因 + ' - '}{$t玩家}:{$t属性表达式文本} >> {$t结果文本}", 1},
    }
    coc["检定_单项结果文本"] = []types.TextTemplateItem{
        {"{$t检定表达式文本}={$t骰子出目}/{$t判定值}{$t检定计算过程} [{$t判定结果}]", 1},
    }
    coc["检定_多轮"] = []types.TextTemplateItem{
        {"[检定x{$t次数}] {$t玩家}:{$t属性表达式文本}\n{$t结果文本}", 1},
    }

    // COC判定结果
    coc["判定_大失败"] = []types.TextTemplateItem{
        {"FUMBLE", 1},
    }
    coc["判定_失败"] = []types.TextTemplateItem{
        {"FAIL", 1},
    }
    coc["判定_成功"] = []types.TextTemplateItem{
        {"SUCCESS", 1},
    }
    coc["判定_成功_困难"] = []types.TextTemplateItem{
        {"HARD_SUCCESS", 1},
    }
    coc["判定_成功_极难"] = []types.TextTemplateItem{
        {"EXTREME_SUCCESS", 1},
    }
    coc["判定_大成功"] = []types.TextTemplateItem{
        {"CRITICAL", 1},
    }

    return cloned
}

// 添加到数组
var outputRewriteStyles = []struct {
    Name    string
    TextMap types.TextTemplateWithWeightDict
}{
    {"[重写1:极简]", buildOutputRewriteMinimal()},
    {"[重写2:女仆]", buildOutputRewriteMaid()},
    {"[重写3:猫娘]", buildOutputRewriteNekomimi()},
    {"[重写4:机器人]", buildOutputRewriteRobot()}, // 新增
}
```

## 技术要点

1. **CommandFormatInfo 的作用**
   - 记录每一步格式化时使用的模板键（Key）
   - 记录当时的所有变量值（LoadRecords）
   - 使得输出重写能够完美还原计算状态

2. **变量还原机制**
   - 临时变量（$t开头）使用 `vm.StoreNameLocal()`
   - 其他变量使用 `exts.VarSetValue()`
   - 确保所有计算中间值都能被正确还原

3. **模板克隆**
   - 使用 `cloneTextTemplateDict()` 深度克隆默认模板
   - 只覆盖需要修改的部分
   - 保持其他模板（如COC、DND）不受影响

## 实际应用场景

### 1. 多语言支持
输入重写可以实现不同语言的命令处理，例如：
- 中文群组使用中文模板
- 英文群组使用英文模板
- 日文群组使用日文模板

### 2. 个性化输出
输出重写可以让不同用户看到不同风格的结果：
- 用户可以设置自己喜欢的角色风格（极简/女仆/猫娘等）
- 根据用户设置动态切换输出风格
- 不影响其他用户的体验

### 3. 群组主题
根据群组属性显示不同风格：
- RP群可以使用更有角色感的文本（女仆、猫娘）
- 正式游戏群使用简洁专业的文本（极简）
- 娱乐群使用更活泼的文本（猫娘）

### 4. A/B测试
测试不同文案对用户体验的影响：
- 随机分配不同文案版本
- 收集用户反馈
- 优化最佳表达方式

### 5. 内容审核与过滤
在输出前进行内容处理：
- 敏感词替换
- 内容长度控制
- 格式化调整

### 6. 规则系统适配
不同规则系统使用不同的表达习惯：
- COC：使用"检定"、"成功"等术语
- DND：使用"豁免"、"AC"等术语
- 自定义规则：定制专属术语体系

## 设计理念

本 Demo 的核心目的是展示如何通过 TextMap 机制实现：

1. **数据与表达分离**：同样的骰点数据，可以用不同方式表达
2. **不影响计算**：所有重写都不影响原始计算结果
3. **灵活扩展**：可以轻松添加新的角色或风格
4. **变体支持**：同一角色可以有多个随机变体，增加趣味性
5. **完整的模板覆盖**：不仅支持基础骰点，还支持 COC 检定等复杂场景
