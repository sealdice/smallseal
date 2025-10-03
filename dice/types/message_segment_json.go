package types

import "encoding/json"

type MessageSegments []IMessageElement

// MarshalJSON 实现 MessageSegments 的 JSON 序列化
func (ms MessageSegments) MarshalJSON() ([]byte, error) {
	var segments []map[string]any
	for _, elem := range ms {
		segment := make(map[string]any)
		segment["type"] = elem.Type()
		switch e := elem.(type) {
		case *TextElement:
			segment["data"] = map[string]any{
				"content": e.Content,
			}
		case *AtElement:
			segment["data"] = map[string]any{
				"target": e.Target,
			}
		case *ReplyElement:
			// 递归序列化嵌套的消息元素
			elementsData := make([]map[string]any, 0, len(e.Elements))
			for _, nestedElem := range e.Elements {
				nestedSegment := make(map[string]any)
				nestedSegment["type"] = nestedElem.Type()
				switch ne := nestedElem.(type) {
				case *TextElement:
					nestedSegment["data"] = map[string]any{
						"content": ne.Content,
					}
				case *AtElement:
					nestedSegment["data"] = map[string]any{
						"target": ne.Target,
					}
				// 可以根据需要添加更多类型
				default:
					nestedSegment["data"] = map[string]any{}
				}
				elementsData = append(elementsData, nestedSegment)
			}
			segment["data"] = map[string]any{
				"replySeq": e.ReplySeq,
				"sender":   e.Sender,
				"groupID":  e.GroupID,
				"elements": elementsData,
			}
		case *TTSElement:
			segment["data"] = map[string]any{
				"content": e.Content,
			}
		case *ImageElement:
			segment["data"] = map[string]any{
				"url": e.URL,
			}
		case *FaceElement:
			segment["data"] = map[string]any{
				"faceID": e.FaceID,
			}
		case *PokeElement:
			segment["data"] = map[string]any{
				"target": e.Target,
			}
		case *FileElement:
			segment["data"] = map[string]any{
				"contentType": e.ContentType,
				"file":        e.File,
				"url":         e.URL,
			}
		case *RecordElement:
			segment["data"] = map[string]any{
				"file": e.File,
			}
		default:
			// 未知类型，使用空数据
			segment["data"] = map[string]any{}
		}
		segments = append(segments, segment)
	}
	return json.Marshal(segments)
}

// UnmarshalJSON 实现 MessageSegments 的 JSON 反序列化
func (ms *MessageSegments) UnmarshalJSON(data []byte) error {
	var segments []map[string]any
	if err := json.Unmarshal(data, &segments); err != nil {
		return err
	}
	*ms = make(MessageSegments, 0, len(segments))
	for _, segment := range segments {
		typeVal, ok := segment["type"]
		if !ok {
			continue
		}
		// 类型可能是数字或字符串
		var elemType ElementType
		switch t := typeVal.(type) {
		case float64:
			elemType = ElementType(int(t))
		case int:
			elemType = ElementType(t)
		case string:
			// 如果是字符串，需要解析
			switch t {
			case "text", "0":
				elemType = Text
			case "at", "1":
				elemType = At
			case "file", "2":
				elemType = File
			case "image", "3":
				elemType = Image
			case "tts", "4":
				elemType = TTS
			case "reply", "5":
				elemType = Reply
			case "record", "6":
				elemType = Record
			case "face", "7":
				elemType = Face
			case "poke", "8":
				elemType = Poke
			default:
				continue // 跳过未知类型
			}
		default:
			continue // 跳过无效类型
		}
		dataVal, ok := segment["data"]
		if !ok {
			continue
		}
		dataMap, ok := dataVal.(map[string]any)
		if !ok {
			continue
		}
		var elem IMessageElement
		switch elemType {
		case Text:
			if content, ok := dataMap["content"].(string); ok {
				elem = &TextElement{Content: content}
			}
		case At:
			if target, ok := dataMap["target"].(string); ok {
				elem = &AtElement{Target: target}
			}
		case Reply:
			replyElem := &ReplyElement{}
			if replySeq, ok := dataMap["replySeq"].(string); ok {
				replyElem.ReplySeq = replySeq
			}
			if sender, ok := dataMap["sender"].(string); ok {
				replyElem.Sender = sender
			}
			if groupID, ok := dataMap["groupID"].(string); ok {
				replyElem.GroupID = groupID
			}
			// 处理嵌套的 elements
			if elementsData, ok := dataMap["elements"].([]any); ok {
				for _, elemData := range elementsData {
					if elemMap, ok := elemData.(map[string]any); ok {
						// 递归解析嵌套元素
						nestedSegments := []map[string]any{elemMap}
						nestedData, _ := json.Marshal(nestedSegments)
						var nestedMs MessageSegments
						if err := nestedMs.UnmarshalJSON(nestedData); err == nil && len(nestedMs) > 0 {
							replyElem.Elements = append(replyElem.Elements, nestedMs[0])
						}
					}
				}
			}
			elem = replyElem
		case TTS:
			if content, ok := dataMap["content"].(string); ok {
				elem = &TTSElement{Content: content}
			}
		case Image:
			imageElem := &ImageElement{}
			if url, ok := dataMap["url"].(string); ok {
				imageElem.URL = url
			}
			elem = imageElem
		case Face:
			if faceID, ok := dataMap["faceID"].(string); ok {
				elem = &FaceElement{FaceID: faceID}
			}
		case Poke:
			if target, ok := dataMap["target"].(string); ok {
				elem = &PokeElement{Target: target}
			}
		case File:
			fileElem := &FileElement{}
			if contentType, ok := dataMap["contentType"].(string); ok {
				fileElem.ContentType = contentType
			}
			if file, ok := dataMap["file"].(string); ok {
				fileElem.File = file
			}
			if url, ok := dataMap["url"].(string); ok {
				fileElem.URL = url
			}
			elem = fileElem
		case Record:
			recordElem := &RecordElement{}
			if file, ok := dataMap["file"]; ok {
				if fileElem, ok := file.(*FileElement); ok {
					recordElem.File = fileElem
				}
			}
			elem = recordElem
		default:
			continue // 跳过未知类型
		}
		if elem != nil {
			*ms = append(*ms, elem)
		}
	}
	return nil
}
