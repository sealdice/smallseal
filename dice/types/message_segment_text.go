package types

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// SegmentText 表示带占位符的文本与原始消息元素之间的映射。
type SegmentText struct {
	Text         string
	Placeholders map[int]IMessageElement
}

// ToSegmentText 将消息元素转换为带占位符的文本表示，并返回占位符与原始元素的映射。
func (ms MessageSegments) ToSegmentText() SegmentText {
	var placeholders map[int]IMessageElement
	var builder strings.Builder
	for idx, elem := range ms {
		if textElem, ok := elem.(*TextElement); ok {
			builder.WriteString(textElem.Content)
			continue
		}
		placeholderIndex := idx + 1
		if placeholders == nil {
			placeholders = make(map[int]IMessageElement)
		}
		placeholders[placeholderIndex] = elem
		builder.WriteByte('$')
		builder.WriteString(strconv.Itoa(placeholderIndex))
	}
	return SegmentText{
		Text:         builder.String(),
		Placeholders: placeholders,
	}
}

// ToMessageSegments 根据占位符映射还原消息元素切片。
func (st SegmentText) ToMessageSegments() MessageSegments {
	if st.Text == "" {
		return nil
	}
	var result MessageSegments
	var builder strings.Builder
	for i := 0; i < len(st.Text); {
		if st.Text[i] != '$' {
			r, size := utf8.DecodeRuneInString(st.Text[i:])
			builder.WriteRune(r)
			i += size
			continue
		}
		if i+1 >= len(st.Text) || st.Text[i+1] < '0' || st.Text[i+1] > '9' {
			builder.WriteByte('$')
			i++
			continue
		}
		j := i + 1
		for j < len(st.Text) && st.Text[j] >= '0' && st.Text[j] <= '9' {
			j++
		}
		idxStr := st.Text[i+1 : j]
		placeholderIndex, err := strconv.Atoi(idxStr)
		if err != nil {
			builder.WriteByte('$')
			builder.WriteString(idxStr)
			i = j
			continue
		}
		elem, ok := st.Placeholders[placeholderIndex]
		if !ok || elem == nil {
			builder.WriteByte('$')
			builder.WriteString(idxStr)
			i = j
			continue
		}
		if builder.Len() > 0 {
			result = append(result, &TextElement{Content: builder.String()})
			builder.Reset()
		}
		result = append(result, elem)
		i = j
	}
	if builder.Len() > 0 {
		result = append(result, &TextElement{Content: builder.String()})
	}
	return result
}

// ToText 返回只包含文本和占位符的字符串表示。
func (ms MessageSegments) ToText() string {
	return ms.ToSegmentText().Text
}

// ParseSegmentText 根据文本和占位符映射构造消息元素列表，用于替换 ConvertStringMessage 中的 ImageRewrite 处理。
func ParseSegmentText(text string, placeholders map[int]IMessageElement) MessageSegments {
	segmentText := SegmentText{Text: text, Placeholders: placeholders}
	return segmentText.ToMessageSegments()
}
