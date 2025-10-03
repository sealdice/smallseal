package types

import "testing"

func TestSegmentTextRoundTrip(t *testing.T) {
	segments := MessageSegments{
		&TextElement{Content: "第一段文本 "},
		&ImageElement{URL: "https://example.com/image.png"},
		&TextElement{Content: " 第二段文本"},
	}

	representation := segments.ToSegmentText()

	expectedText := "第一段文本 $2 第二段文本"
	if representation.Text != expectedText {
		t.Fatalf("unexpected text representation: got %q, want %q", representation.Text, expectedText)
	}

	if len(representation.Placeholders) != 1 {
		t.Fatalf("unexpected placeholder count: got %d, want 1", len(representation.Placeholders))
	}

	placeholder, ok := representation.Placeholders[2]
	if !ok {
		t.Fatal("missing placeholder for index 2")
	}

	imageElem, ok := placeholder.(*ImageElement)
	if !ok {
		t.Fatalf("placeholder type mismatch: got %T, want *ImageElement", placeholder)
	}

	if imageElem.URL != "https://example.com/image.png" {
		t.Fatalf("image url mismatch: got %q", imageElem.URL)
	}

	rebuilt := representation.ToMessageSegments()
	if len(rebuilt) != len(segments) {
		t.Fatalf("unexpected segment count after rebuild: got %d, want %d", len(rebuilt), len(segments))
	}

	firstText, ok := rebuilt[0].(*TextElement)
	if !ok || firstText.Content != "第一段文本 " {
		t.Fatalf("unexpected first segment after rebuild: %#v", rebuilt[0])
	}

	if rebuilt[1] != segments[1] {
		t.Fatalf("non-text segment should be reused, got %#v", rebuilt[1])
	}

	lastText, ok := rebuilt[2].(*TextElement)
	if !ok || lastText.Content != " 第二段文本" {
		t.Fatalf("unexpected last segment after rebuild: %#v", rebuilt[2])
	}
}

func TestSegmentTextReorder(t *testing.T) {
	segments := MessageSegments{
		&TextElement{Content: "part-1 "},
		&ImageElement{URL: "https://example.com/image.png"},
		&TextElement{Content: "part-2"},
	}

	representation := segments.ToSegmentText()

	reordered := SegmentText{
		Text:         "part-1 part-2 $2",
		Placeholders: representation.Placeholders,
	}

	rebuilt := reordered.ToMessageSegments()
	if len(rebuilt) != 2 {
		t.Fatalf("unexpected segment count after reorder: got %d, want 2", len(rebuilt))
	}

	textSeg, ok := rebuilt[0].(*TextElement)
	if !ok {
		t.Fatalf("first segment should be text, got %T", rebuilt[0])
	}

	if textSeg.Content != "part-1 part-2 " {
		t.Fatalf("unexpected text content after reorder: got %q", textSeg.Content)
	}

	if rebuilt[1] != segments[1] {
		t.Fatalf("second segment should reuse placeholder element, got %#v", rebuilt[1])
	}
}

func TestSegmentTextUnknownPlaceholder(t *testing.T) {
	segments := SegmentText{
		Text:         "hello $9 world",
		Placeholders: nil,
	}

	rebuilt := segments.ToMessageSegments()
	if len(rebuilt) != 1 {
		t.Fatalf("expected single text segment, got %d", len(rebuilt))
	}

	textSeg, ok := rebuilt[0].(*TextElement)
	if !ok {
		t.Fatalf("rebuilt segment should be text, got %T", rebuilt[0])
	}

	if textSeg.Content != "hello $9 world" {
		t.Fatalf("unexpected text content: got %q", textSeg.Content)
	}
}
