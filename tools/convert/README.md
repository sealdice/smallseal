# 规则模板转换工具

该工具用于将老版本的规则模板（模板 v1）批量迁移到 SmallSeal 使用的模板 v2 结构。

## 转换流程
1. **读取输入**：`main.go` 会根据 `-input` 指定的文件或标准输入读取原始文本，并调用 `detectInputFormat` 尝试识别是 YAML 还是 JSON。
2. **解析旧模板**：`converter.ParseTemplate` 依据识别出的格式完成反序列化，结构体完整覆盖 v1 中的各类配置（属性、命令、别名、文本映射等）。
3. **结构映射**：`converter.Convert` 将 v1 字段搬运到 `types.GameSystemTemplateV2`，同时执行必要的字段重命名与默认值修正。
4. **序列化输出**：`converter.MarshalOutput` 负责将结果写成 YAML（默认）或 JSON；CLI 根据 `-output` 与 `-format` 选项选择写入文件或标准输出。

## 字段对应关系
| v1 字段 | v2 目标 | 说明 |
|---------|---------|------|
| `name`, `fullName`, `authors`, `version`, `updatedTime`, `templateVer` | 对应同名字段 | 直接复制；若 `templateVer` 为空，则使用 CLI 中的 `-template-ver`，默认 `2.0`。
| `preloadCode` | `InitScript` | 预加载脚本在 v2 中统一命名为 `InitScript`。
| `defaults` | `Attrs.Defaults` | 数值默认值完全拷贝。
| `defaultsComputed` | `Attrs.DefaultsComputed` | 计算型默认值保持原表达式字符串。
| `detailOverwrite` | `Attrs.DetailOverwrite` | 映射到 v2 的 detail 重写表，额外做格式规范化（见下）。
| `attrConfig.top` | `Commands.St.Show.Top` | 显示优先级顺序。
| `attrConfig.sortBy` | `Commands.St.Show.SortBy` | 属性排序方式。
| `attrConfig.ignores` | `Commands.St.Show.Ignores` | 隐藏的属性列表。
| `attrConfig.showAs` | `Commands.St.Show.ShowValueAs` | 对显示值的替换表达式。
| `attrConfig.showAsKey` | `Commands.St.Show.ShowKeyAs` | 对属性名的替换表达式。
| `attrConfig.itemsPerLine` | `Commands.St.Show.ItemsPerLine` | 每行展示的属性数量。
| `setConfig.enableTip` | `Commands.Set.EnableTip` | `.set` 命令提示文本。
| `setConfig.keys` | `Commands.Set.Keys` | `.set` 可用的快捷关键词。
| `setConfig.relatedExt` | `Commands.Set.RelatedExt` | 关联扩展列表。
| `setConfig.diceSidesExpr` / `setConfig.diceSides` | `Commands.Set.DiceSideExpr` | 优先使用表达式；若为空且存在整数面数，则转换为字符串写入。
| `nameTemplate` | `Commands.Sn` | 名片模板键值复制为 `SnTemplate` 结构。
| `alias` | `Alias` | 别名映射直接保留（空映射时置为 `nil`）。
| `textMap` | `textMap`（输出包装层字段） | 原样透传，便于扩展保留权重模板。
| `textMapHelpInfo` | `textMapHelpInfo`（输出包装层字段） | 与 v1 相同，仅调整大小写。

## 额外处理
- 空切片 / 空 map 会在转换后重置为 `nil`，以遵循 v2 运行时代码对空值的判断习惯。
- `detailOverwrite` 字段如含有函数调用或表达式（例如 `savingDetail('力量')`），会自动加上花括号 `{...}` 以满足 v2 detail 覆写对表达式格式的要求；原本为空字符串的条目保持为空。
- `setConfig.diceSides` 与 `diceSidesExpr` 同时存在时保留表达式，避免覆盖作者编写的公式。
- CLI 默认输出 YAML；如需 JSON，请通过 `-format json` 或输出文件后缀为 `.json`。

通过拆分成 `converter` 包，命令行工具只负责输入输出，转换逻辑可被测试或其他工具复用。
