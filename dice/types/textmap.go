package types

type (
	TextTemplateItem     = []any // 实际上是 [string, int] 类型
	TextTemplateItemList []TextTemplateItem
)

type (
	TextTemplateWithWeight     = map[string][]TextTemplateItem
	TextTemplateWithWeightDict = map[string]TextTemplateWithWeight
)
