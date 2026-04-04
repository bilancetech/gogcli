package cmd

import (
	"context"
	"os"
	"strings"

	"google.golang.org/api/docs/v1"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

type DocsAddTabCmd struct {
	DocID       string `arg:"" name:"docId" help:"Doc ID"`
	Title       string `arg:"" name:"title" help:"Title for the new tab"`
	ParentTabID string `name:"parent-tab-id" help:"Parent tab ID for a nested tab"`
	Index       *int64 `name:"index" help:"Zero-based index within the parent tab"`
	Emoji       string `name:"emoji" help:"Emoji icon to show with the tab"`
}

func (c *DocsAddTabCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	docID := strings.TrimSpace(c.DocID)
	title := strings.TrimSpace(c.Title)
	if docID == "" {
		return usage("empty docId")
	}
	if title == "" {
		return usage("empty title")
	}

	payload := map[string]any{
		"document_id": docID,
		"title":       title,
	}
	if c.ParentTabID != "" {
		payload["parent_tab_id"] = c.ParentTabID
	}
	if c.Index != nil {
		payload["index"] = *c.Index
	}
	if c.Emoji != "" {
		payload["emoji"] = c.Emoji
	}
	if err := dryRunExit(ctx, flags, "docs.add-tab", payload); err != nil {
		return err
	}

	svc, err := requireDocsService(ctx, flags)
	if err != nil {
		return err
	}

	tabProps := &docs.TabProperties{
		Title: title,
	}
	if c.ParentTabID != "" {
		tabProps.ParentTabId = c.ParentTabID
	}
	if c.Index != nil {
		tabProps.Index = *c.Index
		tabProps.ForceSendFields = append(tabProps.ForceSendFields, "Index")
	}
	if c.Emoji != "" {
		tabProps.IconEmoji = c.Emoji
	}

	resp, err := svc.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				AddDocumentTab: &docs.AddDocumentTabRequest{
					TabProperties: tabProps,
				},
			},
		},
	}).Context(ctx).Do()
	if err != nil {
		return err
	}

	result := map[string]any{
		"documentId": docID,
		"title":      title,
	}
	if resp != nil && len(resp.Replies) > 0 && resp.Replies[0] != nil && resp.Replies[0].AddDocumentTab != nil && resp.Replies[0].AddDocumentTab.TabProperties != nil {
		props := resp.Replies[0].AddDocumentTab.TabProperties
		result["tabId"] = props.TabId
		result["index"] = props.Index
		if props.Title != "" {
			result["title"] = props.Title
		}
		if props.ParentTabId != "" {
			result["parentTabId"] = props.ParentTabId
		}
		if props.IconEmoji != "" {
			result["emoji"] = props.IconEmoji
		}
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, result)
	}

	u.Out().Printf("documentId\t%s", docID)
	if tabID, ok := result["tabId"].(string); ok && tabID != "" {
		u.Out().Printf("tabId\t%s", tabID)
	}
	if titleOut, ok := result["title"].(string); ok && titleOut != "" {
		u.Out().Printf("title\t%s", titleOut)
	}
	if index, ok := result["index"].(int64); ok {
		u.Out().Printf("index\t%d", index)
	}
	if parentID, ok := result["parentTabId"].(string); ok && parentID != "" {
		u.Out().Printf("parentTabId\t%s", parentID)
	}
	if emoji, ok := result["emoji"].(string); ok && emoji != "" {
		u.Out().Printf("emoji\t%s", emoji)
	}
	return nil
}
