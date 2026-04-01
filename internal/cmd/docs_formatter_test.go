package cmd

import "testing"

func TestMarkdownToDocsRequests_BaseIndex(t *testing.T) {
	elements := []MarkdownElement{{Type: MDParagraph, Content: "**bold**"}}
	requests, text, tables := MarkdownToDocsRequests(elements, 42)

	if text != "bold\n" {
		t.Fatalf("unexpected text: %q", text)
	}
	if len(tables) != 0 {
		t.Fatalf("unexpected tables: %d", len(tables))
	}
	if len(requests) != 1 || requests[0].UpdateTextStyle == nil {
		t.Fatalf("expected one text-style request, got %#v", requests)
	}

	rng := requests[0].UpdateTextStyle.Range
	if rng.StartIndex != 42 || rng.EndIndex != 46 {
		t.Fatalf("unexpected range: [%d,%d]", rng.StartIndex, rng.EndIndex)
	}
}

func TestMarkdownToDocsRequests_TableStartIndexUsesBase(t *testing.T) {
	elements := []MarkdownElement{
		{Type: MDParagraph, Content: "A"},
		{Type: MDTable, TableCells: [][]string{{"h1", "h2"}, {"v1", "v2"}}},
	}
	_, text, tables := MarkdownToDocsRequests(elements, 10)

	if text != "A\n\n" {
		t.Fatalf("unexpected text: %q", text)
	}
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
	if tables[0].StartIndex != 12 {
		t.Fatalf("unexpected table start index: %d", tables[0].StartIndex)
	}
}

func TestMarkdownToDocsRequests_UsesParagraphBulletsForLists(t *testing.T) {
	elements := []MarkdownElement{
		{Type: MDListItem, Content: "**First**"},
		{Type: MDNumberedList, Content: "Second"},
	}
	requests, text, tables := MarkdownToDocsRequests(elements, 5)

	if text != "First\nSecond\n" {
		t.Fatalf("unexpected text: %q", text)
	}
	if len(tables) != 0 {
		t.Fatalf("unexpected tables: %d", len(tables))
	}
	if len(requests) != 3 {
		t.Fatalf("expected 3 requests (2 bullets + 1 style), got %d", len(requests))
	}
	if requests[0].CreateParagraphBullets == nil {
		t.Fatalf("expected first request to create bullets, got %#v", requests[0])
	}
	if requests[0].CreateParagraphBullets.BulletPreset != bulletPresetDisc {
		t.Fatalf("unexpected bullet preset: %q", requests[0].CreateParagraphBullets.BulletPreset)
	}
	if got := requests[0].CreateParagraphBullets.Range; got.StartIndex != 5 || got.EndIndex != 11 {
		t.Fatalf("unexpected bullet range: [%d,%d]", got.StartIndex, got.EndIndex)
	}
	if requests[1].UpdateTextStyle == nil {
		t.Fatalf("expected second request to style bold list text, got %#v", requests[1])
	}
	if got := requests[1].UpdateTextStyle.Range; got.StartIndex != 5 || got.EndIndex != 10 {
		t.Fatalf("unexpected bold range: [%d,%d]", got.StartIndex, got.EndIndex)
	}
	if requests[2].CreateParagraphBullets == nil {
		t.Fatalf("expected third request to create numbered bullets, got %#v", requests[2])
	}
	if requests[2].CreateParagraphBullets.BulletPreset != "NUMBERED_DECIMAL_NESTED" {
		t.Fatalf("unexpected numbered preset: %q", requests[2].CreateParagraphBullets.BulletPreset)
	}
	if got := requests[2].CreateParagraphBullets.Range; got.StartIndex != 11 || got.EndIndex != 18 {
		t.Fatalf("unexpected numbered range: [%d,%d]", got.StartIndex, got.EndIndex)
	}
}
