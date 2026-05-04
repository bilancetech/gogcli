package cmd

import "testing"

func TestMarkdownToDocsRequests_BaseIndex(t *testing.T) {
	elements := []MarkdownElement{{Type: MDParagraph, Content: "**bold**"}}
	requests, text, tables := MarkdownToDocsRequests(elements, 42, "")

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
	_, text, tables := MarkdownToDocsRequests(elements, 10, "")

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
	requests, text, tables := MarkdownToDocsRequests(elements, 5, "")

	if text != "• First\n1. Second\n" {
		t.Fatalf("unexpected text: %q", text)
	}
	if len(tables) != 0 {
		t.Fatalf("unexpected tables: %d", len(tables))
	}
	if len(requests) != 1 {
		t.Fatalf("expected 1 style request, got %d", len(requests))
	}
	if requests[0].UpdateTextStyle == nil {
		t.Fatalf("expected request to style bold list text, got %#v", requests[0])
	}
	if got := requests[0].UpdateTextStyle.Range; got.StartIndex != 7 || got.EndIndex != 12 {
		t.Fatalf("unexpected bold range: [%d,%d]", got.StartIndex, got.EndIndex)
	}
}
